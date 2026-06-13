package handlers

import (
	"errors"
	"net/http"
	"strings"

	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (h *Handler) getAppForOrg(orgID, appID string) (models.App, error) {
	var app models.App
	if err := h.DB.Where("id = ? AND org_id = ?", appID, orgID).First(&app).Error; err != nil {
		return app, err
	}
	return app, nil
}

func (h *Handler) getReleaseForOrg(orgID, releaseID string) (models.Release, error) {
	var release models.Release
	if err := h.DB.Raw(`
		SELECT r.* FROM releases r
		JOIN apps a ON a.id = r.app_id
		WHERE r.id = ? AND a.org_id = ?
	`, releaseID, orgID).Scan(&release).Error; err != nil {
		return release, err
	}
	if release.ID == (uuid.UUID{}) {
		return release, gorm.ErrRecordNotFound
	}
	return release, nil
}

func (h *Handler) getOrgMember(orgID string, userID string) (models.OrgMember, error) {
	var member models.OrgMember
	if err := h.DB.Where("scope_id = ? AND user_id = ?", orgID, userID).First(&member).Error; err != nil {
		return member, err
	}
	return member, nil
}

func (h *Handler) countOrgOwners(orgID string) (int64, error) {
	var count int64
	if err := h.DB.Model(&models.OrgMember{}).Where("scope_id = ? AND role = ?", orgID, "owner").Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (h *Handler) isEnterpriseOwner(userID string) (bool, error) {
	if strings.TrimSpace(userID) == "" {
		return false, nil
	}
	var count int64
	if h.hasOrgTypeColumn() {
		err := h.DB.Raw(`
			SELECT COUNT(*)
			FROM memberships om
			JOIN orgs o ON o.id = om.scope_id
			WHERE om.scope_type = 'org' AND om.user_id = ? AND om.role = 'owner' AND (o.org_type IS NULL OR o.org_type <> 'personal')
		`, userID).Scan(&count).Error
		return count > 0, err
	}
	if err := h.DB.Model(&models.OrgMember{}).
		Where("scope_type = ? AND user_id = ? AND role = ?", models.ScopeOrg, userID, "owner").
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (h *Handler) resolveSystemRole(userID string, systemRole string) (string, error) {
	normalized := normalizeSystemRole(systemRole)
	if normalized != "org_admin" {
		return normalized, nil
	}
	isOwner, err := h.isEnterpriseOwner(userID)
	if err != nil {
		return normalized, err
	}
	if isOwner {
		return normalized, nil
	}
	if strings.TrimSpace(userID) != "" {
		if err := h.DB.Model(&models.User{}).Where("id = ?", userID).Update("system_role", "none").Error; err != nil {
			return normalized, err
		}
	}
	return "none", nil
}

func (h *Handler) hasOrgTypeColumn() bool {
	if h == nil || h.DB == nil {
		return false
	}
	return h.DB.Migrator().HasColumn(&models.Org{}, "org_type")
}

func (h *Handler) hasAppFeedbackEnabledColumn() bool {
	if h == nil || h.DB == nil {
		return false
	}
	return h.DB.Migrator().HasColumn(&models.App{}, "feedback_enabled")
}

func (h *Handler) hasAppHeartbeatIntervalColumn() bool {
	if h == nil || h.DB == nil {
		return false
	}
	return h.DB.Migrator().HasColumn(&models.App{}, "heartbeat_interval_seconds")
}

func (h *Handler) hasAppOnlineEnabledColumn() bool {
	if h == nil || h.DB == nil {
		return false
	}
	return h.DB.Migrator().HasColumn(&models.App{}, "online_enabled")
}

func (h *Handler) hasAppPublicKeyColumn() bool {
	if h == nil || h.DB == nil {
		return false
	}
	return h.DB.Migrator().HasColumn(&models.App{}, "public_key")
}

func (h *Handler) hasReleaseExternalDownloadURLColumn() bool {
	if h == nil || h.DB == nil {
		return false
	}
	return h.DB.Migrator().HasColumn(&models.Release{}, "external_download_url")
}

func (h *Handler) hasAppSecretCiphertextColumn() bool {
	if h == nil || h.DB == nil {
		return false
	}
	return h.DB.Migrator().HasColumn(&models.App{}, "app_secret_ciphertext")
}

func (h *Handler) hasAppSecretScopesColumn() bool {
	if h == nil || h.DB == nil {
		return false
	}
	return h.DB.Migrator().HasColumn(&models.App{}, "app_secret_scopes")
}

func (h *Handler) hasAppSecretExpiresAtColumn() bool {
	if h == nil || h.DB == nil {
		return false
	}
	return h.DB.Migrator().HasColumn(&models.App{}, "app_secret_expires_at")
}

func (h *Handler) hasAppSecretNameColumn() bool {
	if h == nil || h.DB == nil {
		return false
	}
	return h.DB.Migrator().HasColumn(&models.App{}, "app_secret_name")
}

func (h *Handler) hasAppSecretsTable() bool {
	if h == nil || h.DB == nil {
		return false
	}
	return h.DB.Migrator().HasTable(&models.AppSecret{})
}

func (h *Handler) hasDeviceControlsTable() bool {
	if h == nil || h.DB == nil {
		return false
	}
	return h.DB.Migrator().HasTable(&models.DeviceControl{})
}

func (h *Handler) hasFeedbackTable() bool {
	if h == nil || h.DB == nil {
		return false
	}
	return h.DB.Migrator().HasTable(&models.Feedback{})
}

func (h *Handler) hasFeedbackWorkflowColumns() bool {
	if h == nil || h.DB == nil || !h.hasFeedbackTable() {
		return false
	}
	return h.DB.Migrator().HasColumn(&models.Feedback{}, "status") &&
		h.DB.Migrator().HasColumn(&models.Feedback{}, "internal_note") &&
		h.DB.Migrator().HasColumn(&models.Feedback{}, "handled_by") &&
		h.DB.Migrator().HasColumn(&models.Feedback{}, "handled_at") &&
		h.DB.Migrator().HasColumn(&models.Feedback{}, "updated_at")
}

func (h *Handler) hasSystemSettingsTable() bool {
	if h == nil || h.DB == nil {
		return false
	}
	return h.DB.Migrator().HasTable(&models.SystemSetting{})
}

func (h *Handler) hasEmailVerificationCodesTable() bool {
	if h == nil || h.DB == nil {
		return false
	}
	return h.DB.Migrator().HasTable(&models.EmailVerificationCode{})
}

func (h *Handler) isPersonalOrg(orgID string) (bool, error) {
	if strings.TrimSpace(orgID) == "" {
		return false, nil
	}
	if !h.hasOrgTypeColumn() {
		return false, nil
	}
	var org models.Org
	if err := h.DB.Select("org_type").Where("id = ?", orgID).First(&org).Error; err != nil {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(org.OrgType), "personal"), nil
}

func (h *Handler) ensurePersonalOrgMember(userID string) (models.Org, models.OrgMember, error) {
	var org models.Org
	var member models.OrgMember
	if !h.hasOrgTypeColumn() {
		return org, member, nil
	}
	userUUID, err := uuid.Parse(strings.TrimSpace(userID))
	if err != nil {
		return org, member, err
	}

	if err := h.DB.Raw(`
		SELECT o.* FROM orgs o
		JOIN memberships m ON m.scope_id = o.id AND m.scope_type = 'org'
		WHERE m.user_id = ? AND o.org_type = 'personal'
		ORDER BY o.created_at DESC
		LIMIT 1
	`, userID).Scan(&org).Error; err != nil {
		return org, member, err
	}
	if org.ID != (uuid.UUID{}) {
		if err := h.DB.Where("scope_id = ? AND user_id = ?", org.ID, userID).First(&member).Error; err != nil {
			return models.Org{}, models.OrgMember{}, err
		}
		return org, member, nil
	}

	if err := h.DB.Where("created_by = ? AND org_type = ?", userID, "personal").Order("created_at desc").First(&org).Error; err == nil {
		if err := h.DB.Where("scope_id = ? AND user_id = ?", org.ID, userID).First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				member = models.OrgMember{OrgID: org.ID, UserID: userUUID, Role: "owner"}
				if err := h.DB.Create(&member).Error; err != nil {
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
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
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

func (h *Handler) ensureAppWritable(c *gin.Context, orgID, appID string) (models.App, bool) {
	app, err := h.getAppForOrg(orgID, appID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return app, false
	}
	personal, err := h.isPersonalOrg(orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load org"})
		return app, false
	}
	if personal {
		status := strings.ToLower(strings.TrimSpace(app.Status))
		if status != "" && status != "active" && status != "rejected" {
			code := "app_rejected"
			if status == "pending" {
				code = "app_pending_review"
			}
			c.JSON(http.StatusForbidden, gin.H{
				"error":            code,
				"status":           status,
				"rejection_reason": app.RejectionReason,
			})
			return app, false
		}
	}
	return app, true
}
