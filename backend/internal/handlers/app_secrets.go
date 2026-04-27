package handlers

import (
	"net/http"
	"strings"
	"time"

	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"
	"software-web-manager/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type createAppSecretRequest struct {
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	ExpiresAt *string  `json:"expires_at"`
}

type updateAppSecretPolicyRequest struct {
	ExpiresAt *string `json:"expires_at"`
}

func appSecretListItem(secret models.AppSecret) gin.H {
	secretName := strings.TrimSpace(secret.Name)
	if secretName == "" {
		secretName = "app_secret"
	}
	scopes := appSecretScopesFromJSON(secret.ScopesJSON)
	return gin.H{
		"id":           secret.ID,
		"app_id":       secret.AppID,
		"name":         secretName,
		"type":         "app_secret",
		"scopes":       scopes,
		"expires_at":   secret.ExpiresAt,
		"last_used_at": secret.LastUsedAt,
		"created_at":   secret.CreatedAt,
		"updated_at":   secret.UpdatedAt,
	}
}

func (h *Handler) CreateAppSecret(c *gin.Context) {
	if !h.hasAppSecretsTable() {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "missing app_secrets table, run migration 0033_app_secrets"})
		return
	}

	appID := c.Param("id")
	userID := c.GetString(middleware.ContextUserID)

	var req createAppSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orgID := c.GetString(middleware.ContextOrgID)
	if !h.hasPermission(c, "app.manage") && !h.hasAppPermission(userID, appID, "app.manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}
	app, ok := h.ensureAppWritable(c, orgID, appID)
	if !ok {
		return
	}

	now := time.Now()
	secretName := strings.TrimSpace(req.Name)
	if secretName == "" {
		secretName = "app_secret"
	}
	if len(secretName) > 128 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name too long"})
		return
	}

	scopes := sanitizeAppSecretScopes(req.Scopes)
	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		expiresRaw := strings.TrimSpace(*req.ExpiresAt)
		if expiresRaw != "" {
			parsed, parseErr := parseTimeFlexible(expiresRaw)
			if parseErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expires_at"})
				return
			}
			if !parsed.After(now) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "expires_at must be in the future"})
				return
			}
			expiresAt = &parsed
		}
	}

	secret, err := generateAppSecret()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate app secret"})
		return
	}
	secretCipher, err := utils.EncryptAppSecret(h.Cfg.AppSecretMasterKey, secret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt app secret"})
		return
	}

	secretRow := models.AppSecret{
		AppID:            app.ID,
		Name:             secretName,
		SecretCiphertext: secretCipher,
		ScopesJSON:       appSecretScopesJSON(scopes),
		ExpiresAt:        expiresAt,
	}
	if err := h.DB.Create(&secretRow).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save app secret"})
		return
	}

	h.audit(c, "app_secret.create", "app", app.ID, nil, gin.H{
		"app_id":    app.ID,
		"secret_id": secretRow.ID,
		"name":      secretName,
		"scopes":    scopes,
		"expires_at": expiresAt,
	})
	c.JSON(http.StatusOK, gin.H{
		"app_id":     app.ID,
		"app_secret": secret,
		"scopes":     scopes,
		"item":       appSecretListItem(secretRow),
	})
}

func (h *Handler) ListAppSecrets(c *gin.Context) {
	if !h.hasAppSecretsTable() {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "missing app_secrets table, run migration 0033_app_secrets"})
		return
	}

	appID := c.Param("id")
	orgID := c.GetString(middleware.ContextOrgID)
	app, err := h.getAppForOrg(orgID, appID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}

	var secrets []models.AppSecret
	if err := h.DB.
		Where("app_id = ? AND revoked_at IS NULL", app.ID).
		Order("created_at DESC").
		Find(&secrets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list app secrets"})
		return
	}

	items := make([]gin.H, 0, len(secrets))
	for i := range secrets {
		items = append(items, appSecretListItem(secrets[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) RevokeAppSecret(c *gin.Context) {
	if !h.hasAppSecretsTable() {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "missing app_secrets table, run migration 0033_app_secrets"})
		return
	}

	keyID := strings.TrimSpace(c.Param("id"))
	userID := c.GetString(middleware.ContextUserID)
	orgID := c.GetString(middleware.ContextOrgID)
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key id"})
		return
	}
	keyUUID, err := uuid.Parse(keyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key id"})
		return
	}

	var secret models.AppSecret
	if err := h.DB.Where("id = ?", keyUUID).First(&secret).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "app secret not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query app secret"})
		return
	}
	if secret.RevokedAt != nil {
		c.JSON(http.StatusOK, gin.H{"status": "revoked"})
		return
	}

	app, err := h.getAppForOrg(orgID, secret.AppID.String())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app secret not found"})
		return
	}
	if !h.hasPermission(c, "app.manage") && !h.hasAppPermission(userID, app.ID.String(), "app.manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}
	if _, ok := h.ensureAppWritable(c, orgID, app.ID.String()); !ok {
		return
	}

	now := time.Now()
	if err := h.DB.Model(&models.AppSecret{}).
		Where("id = ? AND revoked_at IS NULL", secret.ID).
		Update("revoked_at", now).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke app secret"})
		return
	}

	h.audit(c, "app_secret.revoke", "app", app.ID, gin.H{
		"app_id":    app.ID,
		"secret_id": secret.ID,
	}, nil)
	c.JSON(http.StatusOK, gin.H{"status": "revoked"})
}

func (h *Handler) UpdateAppSecretPolicy(c *gin.Context) {
	if !h.hasAppSecretsTable() {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "missing app_secrets table, run migration 0033_app_secrets"})
		return
	}

	keyID := strings.TrimSpace(c.Param("id"))
	userID := c.GetString(middleware.ContextUserID)
	orgID := c.GetString(middleware.ContextOrgID)
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key id"})
		return
	}
	keyUUID, err := uuid.Parse(keyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key id"})
		return
	}

	var secret models.AppSecret
	if err := h.DB.Where("id = ?", keyUUID).First(&secret).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "app secret not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query app secret"})
		return
	}
	if secret.RevokedAt != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app secret not found"})
		return
	}

	app, err := h.getAppForOrg(orgID, secret.AppID.String())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app secret not found"})
		return
	}
	if !h.hasPermission(c, "app.manage") && !h.hasAppPermission(userID, app.ID.String(), "app.manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}
	if _, ok := h.ensureAppWritable(c, orgID, app.ID.String()); !ok {
		return
	}

	var req updateAppSecretPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		raw := strings.TrimSpace(*req.ExpiresAt)
		if raw != "" {
			parsed, err := parseTimeFlexible(raw)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expires_at"})
				return
			}
			if !parsed.After(time.Now()) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "expires_at must be in the future"})
				return
			}
			expiresAt = &parsed
		}
	}

	if err := h.DB.Model(&models.AppSecret{}).
		Where("id = ? AND revoked_at IS NULL", secret.ID).
		Update("expires_at", expiresAt).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update app secret policy"})
		return
	}

	var updated models.AppSecret
	if err := h.DB.Where("id = ?", secret.ID).First(&updated).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load app secret"})
		return
	}

	h.audit(c, "app_secret.policy_update", "app", app.ID, gin.H{
		"app_id":    app.ID,
		"secret_id": secret.ID,
	}, gin.H{
		"app_id":     app.ID,
		"secret_id":  secret.ID,
		"expires_at": updated.ExpiresAt,
	})
	c.JSON(http.StatusOK, gin.H{
		"item": appSecretListItem(updated),
	})
}
