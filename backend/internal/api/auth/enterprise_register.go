package auth

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"software-web-manager/backend/internal/crypto"
	"software-web-manager/backend/internal/handlers"
	"software-web-manager/backend/internal/models"
	"software-web-manager/backend/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type enterpriseStatusResponse struct {
	OrgID           uuid.UUID  `json:"org_id"`
	OrgName         string     `json:"org_name"`
	Status          string     `json:"status"`
	RejectionReason *string    `json:"rejection_reason"`
	AllowResubmit   bool       `json:"allow_resubmit"`
	ResubmitToken   *string    `json:"resubmit_token,omitempty"`
	ReviewedAt      *time.Time `json:"reviewed_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

func (h *Handler) EnterpriseRegister(c *gin.Context) {
	allowEnterpriseRegister, err := h.AllowEnterpriseRegisterEnabled()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load system settings"})
		return
	}
	if !allowEnterpriseRegister {
		c.JSON(http.StatusForbidden, gin.H{"error": "enterprise_register_disabled"})
		return
	}

	if h.Storage == nil {
		store, err := storage.New(context.Background(), h.Cfg)
		if err != nil && h.Cfg.StorageDriver != "local" {
			fallbackCfg := h.Cfg
			fallbackCfg.StorageDriver = "local"
			store, err = storage.New(context.Background(), fallbackCfg)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "storage not configured"})
			return
		}
		h.Storage = store
	}
	if err := c.Request.ParseMultipartForm(64 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
		return
	}

	orgName := strings.TrimSpace(c.PostForm("org_name"))
	adminEmail := strings.ToLower(strings.TrimSpace(c.PostForm("admin_email")))
	password := strings.TrimSpace(c.PostForm("password"))
	if orgName == "" || strings.EqualFold(orgName, "undefined") || strings.EqualFold(orgName, "null") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_name required"})
		return
	}
	if adminEmail == "" || strings.EqualFold(adminEmail, "undefined") || strings.EqualFold(adminEmail, "null") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "admin_email required"})
		return
	}
	if password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password required"})
		return
	}
	if len(password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password too short"})
		return
	}
	if h.Storage == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage not configured"})
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
		return
	}
	files := form.File["materials"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "materials required"})
		return
	}
	for _, file := range files {
		if file.Size > handlers.MaxEnterpriseMaterialSize {
			c.JSON(http.StatusBadRequest, gin.H{"error": "material too large"})
			return
		}
	}

	var existing models.User
	if err := h.DB.Where("email = ?", adminEmail).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query user"})
		return
	}

	hash, err := crypto.HashPassword(password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	orgID := uuid.New()
	materials, statusCode, err := h.StoreAttachments(c, handlers.AttachmentOwnerOrgRegistrationMaterial, orgID, &orgID, nil, "materials", filepath.ToSlash(filepath.Join("orgs", orgID.String(), "registration_materials")), len(files), handlers.MaxEnterpriseMaterialSize)
	if err != nil {
		if statusCode == 0 {
			statusCode = http.StatusInternalServerError
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	var org models.Org
	hasOrgTypeColumn := h.HasOrgTypeColumn()
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		user = models.User{
			Email:        adminEmail,
			PasswordHash: hash,
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
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register enterprise"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pending": true,
		"user":    gin.H{"id": user.ID, "email": user.Email},
		"org":     gin.H{"id": org.ID, "name": org.Name},
	})
}

func (h *Handler) GetEnterpriseStatus(c *gin.Context) {
	orgID := strings.TrimSpace(c.Param("id"))
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org_id"})
		return
	}
	var org models.Org
	if err := h.DB.Where("id = ?", orgUUID).First(&org).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "org not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load org"})
		return
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
			if err := h.DB.Model(&models.Org{}).Where("id = ?", org.ID).Update("resubmit_token", token).Error; err == nil {
				resubmitToken = &token
			} else {
				resubmitToken = nil
			}
		} else {
			resubmitToken = org.ResubmitToken
		}
	}
	c.JSON(http.StatusOK, enterpriseStatusResponse{
		OrgID:           org.ID,
		OrgName:         org.Name,
		Status:          org.Status,
		RejectionReason: org.RejectionReason,
		AllowResubmit:   org.AllowResubmit,
		ResubmitToken:   resubmitToken,
		ReviewedAt:      reviewedAt,
		CreatedAt:       org.CreatedAt,
	})
}

func (h *Handler) EnterpriseResubmit(c *gin.Context) {
	if h.Storage == nil {
		store, err := storage.New(context.Background(), h.Cfg)
		if err != nil && h.Cfg.StorageDriver != "local" {
			fallbackCfg := h.Cfg
			fallbackCfg.StorageDriver = "local"
			store, err = storage.New(context.Background(), fallbackCfg)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "storage not configured"})
			return
		}
		h.Storage = store
	}
	if err := c.Request.ParseMultipartForm(64 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
		return
	}

	orgID := strings.TrimSpace(c.PostForm("org_id"))
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org_id"})
		return
	}

	var org models.Org
	if err := h.DB.Where("id = ?", orgUUID).First(&org).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "org not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load org"})
		return
	}
	if strings.ToLower(strings.TrimSpace(org.Status)) != "rejected" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org not rejected"})
		return
	}
	if !org.AllowResubmit {
		c.JSON(http.StatusForbidden, gin.H{"error": "resubmit not allowed"})
		return
	}

	resubmitToken := strings.TrimSpace(c.PostForm("resubmit_token"))
	if resubmitToken == "" || strings.EqualFold(resubmitToken, "undefined") || strings.EqualFold(resubmitToken, "null") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resubmit_token required"})
		return
	}
	if org.ResubmitToken == nil || strings.TrimSpace(*org.ResubmitToken) == "" || !strings.EqualFold(strings.TrimSpace(*org.ResubmitToken), resubmitToken) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid resubmit token"})
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
		return
	}
	files := form.File["materials"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "materials required"})
		return
	}
	for _, file := range files {
		if file.Size > handlers.MaxEnterpriseMaterialSize {
			c.JSON(http.StatusBadRequest, gin.H{"error": "material too large"})
			return
		}
	}

	materials, statusCode, err := h.StoreAttachments(c, handlers.AttachmentOwnerOrgRegistrationMaterial, org.ID, &org.ID, nil, "materials", filepath.ToSlash(filepath.Join("orgs", org.ID.String(), "registration_materials")), len(files), handlers.MaxEnterpriseMaterialSize)
	if err != nil {
		if statusCode == 0 {
			statusCode = http.StatusInternalServerError
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Org{}).
			Where("id = ?", org.ID).
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
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resubmit"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pending": true,
		"org":     gin.H{"id": org.ID, "name": org.Name},
	})
}
