// Package org provides org-domain data access and role resolution that is
// independent of the HTTP layer (no gin, no response writing).
package org

import (
	"errors"
	"strings"

	"software-web-manager/backend/internal/db/schema"
	"software-web-manager/backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Service exposes org queries and membership/role resolution over a single
// database handle.
type Service struct {
	DB *gorm.DB
}

// NewService builds an org service from a database handle.
func NewService(db *gorm.DB) *Service {
	return &Service{DB: db}
}

// NormalizeSystemRole lower-cases/trims a system role, defaulting to "none".
func NormalizeSystemRole(role string) string {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		return "none"
	}
	return role
}

// AppWriteBlock explains why a personal-org app may not be written to.
type AppWriteBlock struct {
	Code   string // "app_pending_review" | "app_rejected"
	Status string
}

// AppWriteBlockForPersonal returns a block reason when an app in a personal org
// is in a state that forbids writes, or nil when the app is writable. Active and
// rejected apps remain writable; an empty status is treated as writable too.
func AppWriteBlockForPersonal(status string) *AppWriteBlock {
	normalized := strings.ToLower(strings.TrimSpace(status))
	if normalized == "" || normalized == "active" || normalized == "rejected" {
		return nil
	}
	code := "app_rejected"
	if normalized == "pending" {
		code = "app_pending_review"
	}
	return &AppWriteBlock{Code: code, Status: normalized}
}

// GetMember loads a user's membership in an org scope.
func (s *Service) GetMember(orgID string, userID string) (models.OrgMember, error) {
	var member models.OrgMember
	if err := s.DB.Where("scope_id = ? AND user_id = ?", orgID, userID).First(&member).Error; err != nil {
		return member, err
	}
	return member, nil
}

// CountOwners returns how many owners an org has.
func (s *Service) CountOwners(orgID string) (int64, error) {
	var count int64
	if err := s.DB.Model(&models.OrgMember{}).Where("scope_id = ? AND role = ?", orgID, "owner").Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// IsEnterpriseOwner reports whether the user owns at least one non-personal org.
func (s *Service) IsEnterpriseOwner(userID string) (bool, error) {
	if strings.TrimSpace(userID) == "" {
		return false, nil
	}
	var count int64
	if schema.HasOrgTypeColumn(s.DB) {
		err := s.DB.Raw(`
			SELECT COUNT(*)
			FROM memberships om
			JOIN orgs o ON o.id = om.scope_id
			WHERE om.scope_type = 'org' AND om.user_id = ? AND om.role = 'owner' AND (o.org_type IS NULL OR o.org_type <> 'personal')
		`, userID).Scan(&count).Error
		return count > 0, err
	}
	if err := s.DB.Model(&models.OrgMember{}).
		Where("scope_type = ? AND user_id = ? AND role = ?", models.ScopeOrg, userID, "owner").
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ResolveSystemRole normalizes a system role and downgrades a stale org_admin to
// "none" when the user no longer owns any enterprise org.
func (s *Service) ResolveSystemRole(userID string, systemRole string) (string, error) {
	normalized := NormalizeSystemRole(systemRole)
	if normalized != "org_admin" {
		return normalized, nil
	}
	isOwner, err := s.IsEnterpriseOwner(userID)
	if err != nil {
		return normalized, err
	}
	if isOwner {
		return normalized, nil
	}
	if strings.TrimSpace(userID) != "" {
		if err := s.DB.Model(&models.User{}).Where("id = ?", userID).Update("system_role", "none").Error; err != nil {
			return normalized, err
		}
	}
	return "none", nil
}

// IsPersonal reports whether an org is a personal workspace.
func (s *Service) IsPersonal(orgID string) (bool, error) {
	if strings.TrimSpace(orgID) == "" {
		return false, nil
	}
	if !schema.HasOrgTypeColumn(s.DB) {
		return false, nil
	}
	var org models.Org
	if err := s.DB.Select("org_type").Where("id = ?", orgID).First(&org).Error; err != nil {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(org.OrgType), "personal"), nil
}

// EnsurePersonalMember returns the user's personal org and membership, creating
// them if they do not yet exist.
func (s *Service) EnsurePersonalMember(userID string) (models.Org, models.OrgMember, error) {
	var org models.Org
	var member models.OrgMember
	if !schema.HasOrgTypeColumn(s.DB) {
		return org, member, nil
	}
	userUUID, err := uuid.Parse(strings.TrimSpace(userID))
	if err != nil {
		return org, member, err
	}

	if err := s.DB.Raw(`
		SELECT o.* FROM orgs o
		JOIN memberships m ON m.scope_id = o.id AND m.scope_type = 'org'
		WHERE m.user_id = ? AND o.org_type = 'personal'
		ORDER BY o.created_at DESC
		LIMIT 1
	`, userID).Scan(&org).Error; err != nil {
		return org, member, err
	}
	if org.ID != (uuid.UUID{}) {
		if err := s.DB.Where("scope_id = ? AND user_id = ?", org.ID, userID).First(&member).Error; err != nil {
			return models.Org{}, models.OrgMember{}, err
		}
		return org, member, nil
	}

	if err := s.DB.Where("created_by = ? AND org_type = ?", userID, "personal").Order("created_at desc").First(&org).Error; err == nil {
		if err := s.DB.Where("scope_id = ? AND user_id = ?", org.ID, userID).First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				member = models.OrgMember{OrgID: org.ID, UserID: userUUID, Role: "owner"}
				if err := s.DB.Create(&member).Error; err != nil {
					return models.Org{}, models.OrgMember{}, err
				}
				return org, member, nil
			}
			return models.Org{}, models.OrgMember{}, err
		}
		return org, member, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Org{}, models.OrgMember{}, err
	}

	org = models.Org{
		Name:      "个人空间",
		Plan:      "free",
		Status:    "active",
		CreatedBy: userUUID,
		OrgType:   "personal",
	}
	member = models.OrgMember{OrgID: org.ID, UserID: userUUID, Role: "owner"}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&org).Error; err != nil {
			return err
		}
		member.OrgID = org.ID
		return tx.Create(&member).Error
	}); err != nil {
		return models.Org{}, models.OrgMember{}, err
	}
	return org, member, nil
}
