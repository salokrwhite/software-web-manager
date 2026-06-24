package auth

import (
	"net/http"
	"path/filepath"
	"software-web-manager/backend/internal/api/common"
	attachment "software-web-manager/backend/internal/services/attachment"
	authsvc "software-web-manager/backend/internal/services/auth"
	"strings"
	"time"

	"software-web-manager/backend/internal/crypto"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	svc := authsvc.NewService(h.DB, h.Cfg)
	if err := svc.EnsureEnterpriseRegisterAllowed(); err != nil {
		h.writeAuthError(c, err)
		return
	}

	if err := h.EnsureStorage(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage not configured"})
		return
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
		if file.Size > common.MaxEnterpriseMaterialSize {
			c.JSON(http.StatusBadRequest, gin.H{"error": "material too large"})
			return
		}
	}

	registered, err := svc.EmailRegistered(adminEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query user"})
		return
	}
	if registered {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	hash, err := crypto.HashPassword(password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	orgID := uuid.New()
	materials, statusCode, err := common.StoreAttachments(h.Storage, h.Cfg.StorageDriver, c, attachment.OwnerOrgRegistrationMaterial, orgID, &orgID, nil, "materials", filepath.ToSlash(filepath.Join("orgs", orgID.String(), "registration_materials")), len(files), common.MaxEnterpriseMaterialSize)
	if err != nil {
		if statusCode == 0 {
			statusCode = http.StatusInternalServerError
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	user, org, err := svc.CreateEnterprise(orgID, orgName, adminEmail, hash, materials)
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

	result, err := authsvc.NewService(h.DB, h.Cfg).EnterpriseStatus(orgUUID)
	if err != nil {
		h.writeAuthError(c, err)
		return
	}
	org := result.Org
	c.JSON(http.StatusOK, enterpriseStatusResponse{
		OrgID:           org.ID,
		OrgName:         org.Name,
		Status:          org.Status,
		RejectionReason: org.RejectionReason,
		AllowResubmit:   org.AllowResubmit,
		ResubmitToken:   result.ResubmitToken,
		ReviewedAt:      result.ReviewedAt,
		CreatedAt:       org.CreatedAt,
	})
}

func (h *Handler) EnterpriseResubmit(c *gin.Context) {
	if err := h.EnsureStorage(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage not configured"})
		return
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

	svc := authsvc.NewService(h.DB, h.Cfg)
	org, err := svc.ValidateEnterpriseResubmit(orgUUID, c.PostForm("resubmit_token"))
	if err != nil {
		h.writeAuthError(c, err)
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
		if file.Size > common.MaxEnterpriseMaterialSize {
			c.JSON(http.StatusBadRequest, gin.H{"error": "material too large"})
			return
		}
	}

	materials, statusCode, err := common.StoreAttachments(h.Storage, h.Cfg.StorageDriver, c, attachment.OwnerOrgRegistrationMaterial, org.ID, &org.ID, nil, "materials", filepath.ToSlash(filepath.Join("orgs", org.ID.String(), "registration_materials")), len(files), common.MaxEnterpriseMaterialSize)
	if err != nil {
		if statusCode == 0 {
			statusCode = http.StatusInternalServerError
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	if err := svc.ResubmitEnterprise(org.ID, materials); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resubmit"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pending": true,
		"org":     gin.H{"id": org.ID, "name": org.Name},
	})
}
