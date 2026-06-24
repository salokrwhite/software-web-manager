package handlers

import (
	"errors"
	"net/http"
	"strings"

	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"
	"software-web-manager/backend/internal/services/system"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// systemSvc builds a system-settings service from the handler's dependencies.
func (h *Handler) systemSvc() *system.Service {
	return system.NewService(h.DB)
}

// The following thin methods preserve the handler-level API used by sibling handlers
// (auth, registration, org management). They delegate to the system service.

func (h *Handler) ListSystemSettings() (map[string]models.SystemSetting, error) {
	return h.systemSvc().ListSettings()
}

func (h *Handler) AllowUserRegisterEnabled() (bool, error) {
	return h.systemSvc().AllowUserRegister()
}

func (h *Handler) AllowEnterpriseRegisterEnabled() (bool, error) {
	return h.systemSvc().AllowEnterpriseRegister()
}

func (h *Handler) GetOrgPlanTypes() ([]string, error) {
	return h.systemSvc().OrgPlanTypes()
}

func (h *Handler) GetSMTPConfigFromSettings(items map[string]models.SystemSetting) system.SMTPConfig {
	return h.systemSvc().SMTPConfigFromSettings(items)
}

func (h *Handler) GetSMTPPasswordFromSettings(items map[string]models.SystemSetting) (string, bool, error) {
	return h.systemSvc().SMTPPasswordFromSettings(items)
}

func (h *Handler) GetSystemSettings(c *gin.Context) {
	svc := h.systemSvc()
	if !svc.HasSettingsTable() {
		resp := system.BuildSettingsResponse(nil)
		resp.SSORedirectURI = SSODeriveRedirectURI(c)
		c.JSON(http.StatusOK, resp)
		return
	}
	items, err := svc.ListSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load system settings"})
		return
	}
	items = svc.RefreshServiceStatusHeartbeat(items)
	resp := system.BuildSettingsResponse(items)
	// Surface the effective callback URL (auto-derived from the backend domain
	// unless an explicit override was stored), so the admin can register it at
	// the IdP without typing it by hand.
	if strings.TrimSpace(resp.SSORedirectURI) == "" {
		resp.SSORedirectURI = SSODeriveRedirectURI(c)
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetPublicSettings(c *gin.Context) {
	svc := h.systemSvc()
	if !svc.HasSettingsTable() {
		c.JSON(http.StatusOK, system.BuildSettingsResponse(nil))
		return
	}
	items, err := svc.ListSettings()
	if err != nil {
		c.JSON(http.StatusOK, system.BuildSettingsResponse(nil))
		return
	}
	items = svc.RefreshServiceStatusHeartbeat(items)
	c.JSON(http.StatusOK, system.BuildSettingsResponse(items))
}

func (h *Handler) UpdateSystemSettings(c *gin.Context) {
	var req system.UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	resp, err := h.systemSvc().UpdateSettings(req, userUUID)
	if err != nil {
		var ve *system.ValidationError
		if errors.As(err, &ve) {
			c.JSON(http.StatusBadRequest, gin.H{"error": ve.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) TestSystemSMTP(c *gin.Context) {
	var req system.TestSMTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.systemSvc().SendTestMail(req); err != nil {
		var ve *system.ValidationError
		if errors.As(err, &ve) {
			c.JSON(http.StatusBadRequest, gin.H{"error": ve.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
