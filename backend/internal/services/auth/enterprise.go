package auth

import (
	"errors"
	"strings"
	"time"

	"software-web-manager/backend/internal/db/schema"
	"software-web-manager/backend/internal/models"
	systemsvc "software-web-manager/backend/internal/services/system"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EnterpriseStatusResult carries the resolved enterprise registration status.
type EnterpriseStatusResult struct {
	Org           models.Org
	ReviewedAt    *time.Time
	ResubmitToken *string
}

// EnsureEnterpriseRegisterAllowed verifies enterprise registration is enabled.
func (s *Service) EnsureEnterpriseRegisterAllowed() error {
	allow, err := systemsvc.NewService(s.DB).AllowEnterpriseRegister()
	if err != nil {
		return newError(500, "failed to load system settings")
	}
	if !allow {
		return newError(403, "enterprise_register_disabled")
	}
	return nil
}

// CreateEnterprise creates the pending admin user, enterprise org, owner
// membership, and registration material rows in one transaction.
func (s *Service) CreateEnterprise(orgID uuid.UUID, orgName, adminEmail, passwordHash string, materials []models.Attachment) (models.User, models.Org, error) {
	var user models.User
	var org models.Org
	hasOrgTypeColumn := schema.HasOrgTypeColumn(s.DB)
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		user = models.User{
			Email:        adminEmail,
			PasswordHash: passwordHash,
			Status:       "pending",
			SystemRole:   "org_admin",
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		org = models.Org{
			ID:        orgID,
			Name:      orgName,
			Plan:      "free",
			OrgType:   "enterprise",
			Status:    "pending",
			CreatedBy: user.ID,
		}
		if !hasOrgTypeColumn {
			org.OrgType = ""
		}
		if hasOrgTypeColumn {
			if err := tx.Create(&org).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Omit("org_type").Create(&org).Error; err != nil {
				return err
			}
		}
		member := models.OrgMember{
			OrgID:  orgID,
			UserID: user.ID,
			Role:   "owner",
		}
		if err := tx.Create(&member).Error; err != nil {
			return err
		}
		for i := range materials {
			if err := tx.Create(&materials[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return user, org, err
}

// EnterpriseStatus loads an org and resolves its review timestamp and (for
// rejected orgs allowing resubmission) a resubmit token, minting one when absent.
func (s *Service) EnterpriseStatus(orgUUID uuid.UUID) (*EnterpriseStatusResult, error) {
	var org models.Org
	if err := s.DB.Where("id = ?", orgUUID).First(&org).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, newError(404, "org not found")
		}
		return nil, newError(500, "failed to load org")
	}
	status := strings.ToLower(strings.TrimSpace(org.Status))
	var reviewedAt *time.Time
	if status == "active" && org.ApprovedAt != nil {
		reviewedAt = org.ApprovedAt
	} else if status == "rejected" && org.RejectedAt != nil {
		reviewedAt = org.RejectedAt
	}
	var resubmitToken *string
	if status == "rejected" && org.AllowResubmit {
		if org.ResubmitToken == nil || strings.TrimSpace(*org.ResubmitToken) == "" {
			token := uuid.NewString()
			if err := s.DB.Model(&models.Org{}).Where("id = ?", org.ID).Update("resubmit_token", token).Error; err == nil {
				resubmitToken = &token
			} else {
				resubmitToken = nil
			}
		} else {
			resubmitToken = org.ResubmitToken
		}
	}
	return &EnterpriseStatusResult{Org: org, ReviewedAt: reviewedAt, ResubmitToken: resubmitToken}, nil
}

// ValidateEnterpriseResubmit loads an org and verifies it is rejected, allows
// resubmission, and the provided resubmit token matches.
func (s *Service) ValidateEnterpriseResubmit(orgUUID uuid.UUID, resubmitToken string) (models.Org, error) {
	var org models.Org
	if err := s.DB.Where("id = ?", orgUUID).First(&org).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return org, newError(404, "org not found")
		}
		return org, newError(500, "failed to load org")
	}
	if strings.ToLower(strings.TrimSpace(org.Status)) != "rejected" {
		return org, newError(400, "org not rejected")
	}
	if !org.AllowResubmit {
		return org, newError(403, "resubmit not allowed")
	}
	resubmitToken = strings.TrimSpace(resubmitToken)
	if resubmitToken == "" || strings.EqualFold(resubmitToken, "undefined") || strings.EqualFold(resubmitToken, "null") {
		return org, newError(400, "resubmit_token required")
	}
	if org.ResubmitToken == nil || strings.TrimSpace(*org.ResubmitToken) == "" || !strings.EqualFold(strings.TrimSpace(*org.ResubmitToken), resubmitToken) {
		return org, newError(401, "invalid resubmit token")
	}
	return org, nil
}

// ResubmitEnterprise resets a rejected org back to pending and appends the new
// registration material rows in one transaction.
func (s *Service) ResubmitEnterprise(orgID uuid.UUID, materials []models.Attachment) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Org{}).
			Where("id = ?", orgID).
			Updates(map[string]any{
				"status":           "pending",
				"rejection_reason": nil,
				"allow_resubmit":   false,
				"resubmit_token":   nil,
				"rejected_by":      nil,
				"rejected_at":      nil,
			}).Error; err != nil {
			return err
		}
		for i := range materials {
			if err := tx.Create(&materials[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
