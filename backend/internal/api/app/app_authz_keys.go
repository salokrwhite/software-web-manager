package app

import (
	"net/http"
	"strings"
	"time"

	"software-web-manager/backend/internal/api/common"
	"software-web-manager/backend/internal/db/schema"
	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"
	appsvc "software-web-manager/backend/internal/services/app"
	orgsvc "software-web-manager/backend/internal/services/org"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const authzKeysMissingTableMsg = "missing app_authz_keys table, run migration 0006_app_authz_keys"

type createAppAuthzKeyRequest struct {
	KeyID string `json:"key_id"`
}

// appAuthzKeyListItem renders a key for the console. It never exposes the
// encrypted private seed — only the public key id and material developers embed.
func appAuthzKeyListItem(key models.AppAuthzKey) gin.H {
	return gin.H{
		"id":           key.ID,
		"app_id":       key.AppID,
		"key_id":       key.KeyID,
		"algorithm":    key.Algorithm,
		"public_key":   key.PublicKey,
		"status":       key.Status,
		"created_at":   key.CreatedAt,
		"activated_at": key.ActivatedAt,
		"rotated_at":   key.RotatedAt,
		"revoked_at":   key.RevokedAt,
		"updated_at":   key.UpdatedAt,
	}
}

// validAuthzKeyID accepts a conservative public-identifier charset so the id is
// safe to embed in clients and stable as a map key.
func validAuthzKeyID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, ch := range id {
		switch {
		case ch >= 'a' && ch <= 'z', ch >= 'A' && ch <= 'Z', ch >= '0' && ch <= '9':
		case ch == '-', ch == '_', ch == '.':
		default:
			return false
		}
	}
	return true
}

// authorizeAppAuthzWrite enforces the same RBAC + writability gate app_secrets
// uses for management mutations on /apps/:id/authz-keys.
func (h *Handler) authorizeAppAuthzWrite(c *gin.Context, appID string) (models.App, bool) {
	userID := c.GetString(middleware.ContextUserID)
	orgID := c.GetString(middleware.ContextOrgID)
	if !common.HasPermission(c, "app.manage") && !orgsvc.NewService(h.DB).HasAppPermission(userID, appID, "app.manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return models.App{}, false
	}
	return common.EnsureAppWritable(h.DB, c, orgID, appID)
}

// loadAuthzKeyForWrite resolves /authz-keys/:id to its key + app, scoped to the
// caller's org, and enforces RBAC + writability.
func (h *Handler) loadAuthzKeyForWrite(c *gin.Context) (models.AppAuthzKey, models.App, bool) {
	keyID := strings.TrimSpace(c.Param("id"))
	keyUUID, err := uuid.Parse(keyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key id"})
		return models.AppAuthzKey{}, models.App{}, false
	}
	var key models.AppAuthzKey
	if err := h.DB.Where("id = ?", keyUUID).First(&key).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "authz key not found"})
			return models.AppAuthzKey{}, models.App{}, false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query authz key"})
		return models.AppAuthzKey{}, models.App{}, false
	}
	orgID := c.GetString(middleware.ContextOrgID)
	if _, err := appsvc.NewService(h.DB).GetForOrg(orgID, key.AppID.String()); err != nil {
		// Hide cross-org keys behind a 404 rather than leaking existence.
		c.JSON(http.StatusNotFound, gin.H{"error": "authz key not found"})
		return models.AppAuthzKey{}, models.App{}, false
	}
	app, ok := h.authorizeAppAuthzWrite(c, key.AppID.String())
	if !ok {
		return models.AppAuthzKey{}, models.App{}, false
	}
	return key, app, true
}

// CreateAppAuthzKey generates a new keypair for the app and stores it as a
// pending key (the server does not sign with it until it is activated).
func (h *Handler) CreateAppAuthzKey(c *gin.Context) {
	if !schema.HasAppAuthzKeysTable(h.DB) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": authzKeysMissingTableMsg})
		return
	}
	appID := c.Param("id")

	var req createAppAuthzKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	app, ok := h.authorizeAppAuthzWrite(c, appID)
	if !ok {
		return
	}

	row, ok := h.createPendingAuthzKey(c, app.ID, strings.TrimSpace(req.KeyID))
	if !ok {
		return
	}

	common.Audit(h.DB, c, "app_authz_key.create", "app", app.ID, nil, gin.H{
		"app_id": app.ID,
		"key_id": row.KeyID,
		"status": row.Status,
	})
	c.JSON(http.StatusOK, gin.H{
		"app_id":     app.ID,
		"key_id":     row.KeyID,
		"public_key": row.PublicKey,
		"item":       appAuthzKeyListItem(row),
	})
}

// createPendingAuthzKey generates + persists a pending key, defaulting/validating
// the key id. Shared by Create and Rotate. On failure it writes the response.
func (h *Handler) createPendingAuthzKey(c *gin.Context, appID uuid.UUID, requestedKeyID string) (models.AppAuthzKey, bool) {
	keyID := requestedKeyID
	if keyID == "" {
		keyID = appsvc.DefaultAuthzKeyID(appID)
	} else if !validAuthzKeyID(keyID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key_id (allowed: letters, digits, '-', '_', '.', max 64)"})
		return models.AppAuthzKey{}, false
	}

	var existing int64
	if err := h.DB.Model(&models.AppAuthzKey{}).Where("app_id = ? AND key_id = ?", appID, keyID).Count(&existing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check key id"})
		return models.AppAuthzKey{}, false
	}
	if existing > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "key_id already exists for this app"})
		return models.AppAuthzKey{}, false
	}

	seedHex, pubHex, err := appsvc.GenerateAuthzKeypair()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate keypair"})
		return models.AppAuthzKey{}, false
	}
	row, err := appsvc.BuildAuthzKeyRow(appID, keyID, seedHex, pubHex, appsvc.AuthzKeyStatusPending, h.Cfg.AppSecretMasterKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt key"})
		return models.AppAuthzKey{}, false
	}
	if err := h.DB.Create(&row).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			c.JSON(http.StatusConflict, gin.H{"error": "key_id already exists for this app"})
			return models.AppAuthzKey{}, false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save authz key"})
		return models.AppAuthzKey{}, false
	}
	return row, true
}

// ListAppAuthzKeys lists an app's keys (public material + status only).
func (h *Handler) ListAppAuthzKeys(c *gin.Context) {
	if !schema.HasAppAuthzKeysTable(h.DB) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": authzKeysMissingTableMsg})
		return
	}
	appID := c.Param("id")
	orgID := c.GetString(middleware.ContextOrgID)
	app, err := appsvc.NewService(h.DB).GetForOrg(orgID, appID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}

	var keys []models.AppAuthzKey
	if err := h.DB.
		Where("app_id = ?", app.ID).
		Order("created_at DESC").
		Find(&keys).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list authz keys"})
		return
	}
	items := make([]gin.H, 0, len(keys))
	for i := range keys {
		items = append(items, appAuthzKeyListItem(keys[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// ActivateAppAuthzKey promotes a pending key to active and retires the app's
// previous active key, so the server starts signing with the new key.
func (h *Handler) ActivateAppAuthzKey(c *gin.Context) {
	if !schema.HasAppAuthzKeysTable(h.DB) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": authzKeysMissingTableMsg})
		return
	}
	key, app, ok := h.loadAuthzKeyForWrite(c)
	if !ok {
		return
	}
	if key.RevokedAt != nil || key.Status == appsvc.AuthzKeyStatusRetired {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot activate a retired or revoked key"})
		return
	}
	if key.Status == appsvc.AuthzKeyStatusActive {
		c.JSON(http.StatusOK, gin.H{"status": "active", "item": appAuthzKeyListItem(key)})
		return
	}

	now := time.Now()
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		// Retire any other currently-active keys for this app (single-active invariant).
		if err := tx.Model(&models.AppAuthzKey{}).
			Where("app_id = ? AND status = ? AND id <> ? AND revoked_at IS NULL", app.ID, appsvc.AuthzKeyStatusActive, key.ID).
			Updates(map[string]interface{}{"status": appsvc.AuthzKeyStatusRetired, "rotated_at": now}).Error; err != nil {
			return err
		}
		return tx.Model(&models.AppAuthzKey{}).
			Where("id = ?", key.ID).
			Updates(map[string]interface{}{"status": appsvc.AuthzKeyStatusActive, "activated_at": now}).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to activate authz key"})
		return
	}
	h.AuthzSignerCache.Invalidate(app.ID.String())

	var updated models.AppAuthzKey
	_ = h.DB.Where("id = ?", key.ID).First(&updated).Error
	common.Audit(h.DB, c, "app_authz_key.activate", "app", app.ID, gin.H{"key_id": key.KeyID, "status": key.Status}, gin.H{"key_id": key.KeyID, "status": appsvc.AuthzKeyStatusActive})
	c.JSON(http.StatusOK, gin.H{"status": "active", "item": appAuthzKeyListItem(updated)})
}

// RotateAppAuthzKey is a convenience alias: it creates a fresh pending key for
// the same app as the addressed key. The caller then publishes the new public
// key to clients and activates it (see the rotation flow in the design doc).
func (h *Handler) RotateAppAuthzKey(c *gin.Context) {
	if !schema.HasAppAuthzKeysTable(h.DB) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": authzKeysMissingTableMsg})
		return
	}
	key, app, ok := h.loadAuthzKeyForWrite(c)
	if !ok {
		return
	}
	row, ok := h.createPendingAuthzKey(c, app.ID, "")
	if !ok {
		return
	}
	common.Audit(h.DB, c, "app_authz_key.rotate", "app", app.ID, gin.H{"from_key_id": key.KeyID}, gin.H{
		"app_id": app.ID,
		"key_id": row.KeyID,
		"status": row.Status,
	})
	c.JSON(http.StatusOK, gin.H{
		"app_id":     app.ID,
		"key_id":     row.KeyID,
		"public_key": row.PublicKey,
		"item":       appAuthzKeyListItem(row),
	})
}

// RevokeAppAuthzKey hard-stops a key: the server immediately stops signing with
// it. Revoking an app's only active key is refused when there is no platform
// fallback, to avoid leaving the app with no signer (which would lock out new
// clients that require a verdict).
func (h *Handler) RevokeAppAuthzKey(c *gin.Context) {
	if !schema.HasAppAuthzKeysTable(h.DB) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": authzKeysMissingTableMsg})
		return
	}
	key, app, ok := h.loadAuthzKeyForWrite(c)
	if !ok {
		return
	}
	if key.RevokedAt != nil {
		c.JSON(http.StatusOK, gin.H{"status": "revoked"})
		return
	}

	if key.Status == appsvc.AuthzKeyStatusActive && !h.Cfg.AuthzPlatformFallback {
		activeCount, err := appsvc.NewService(h.DB).CountActiveAuthzKeys(app.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count active keys"})
			return
		}
		if activeCount <= 1 {
			c.JSON(http.StatusConflict, gin.H{"error": "cannot revoke the only active key while platform fallback is disabled"})
			return
		}
	}

	now := time.Now()
	if err := h.DB.Model(&models.AppAuthzKey{}).
		Where("id = ? AND revoked_at IS NULL", key.ID).
		Updates(map[string]interface{}{"status": appsvc.AuthzKeyStatusRetired, "revoked_at": now}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke authz key"})
		return
	}
	h.AuthzSignerCache.Invalidate(app.ID.String())

	common.Audit(h.DB, c, "app_authz_key.revoke", "app", app.ID, gin.H{"key_id": key.KeyID, "status": key.Status}, nil)
	c.JSON(http.StatusOK, gin.H{"status": "revoked"})
}

// PurgeAppAuthzKey permanently deletes a key row. Only non-active keys (retired
// or revoked) can be purged; the live active key must be revoked/rotated first.
// This is a cleanup of keys no longer in use and does not affect signing.
func (h *Handler) PurgeAppAuthzKey(c *gin.Context) {
	if !schema.HasAppAuthzKeysTable(h.DB) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": authzKeysMissingTableMsg})
		return
	}
	key, app, ok := h.loadAuthzKeyForWrite(c)
	if !ok {
		return
	}
	if key.Status == appsvc.AuthzKeyStatusActive && key.RevokedAt == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "cannot delete an active key; revoke it first"})
		return
	}
	if err := h.DB.Where("id = ?", key.ID).Delete(&models.AppAuthzKey{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete authz key"})
		return
	}
	common.Audit(h.DB, c, "app_authz_key.delete", "app", app.ID, gin.H{"key_id": key.KeyID, "status": key.Status}, nil)
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
