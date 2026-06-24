package system

import (
	"errors"
	"net/http"
	"strings"

	"software-web-manager/backend/internal/api/common"
	"software-web-manager/backend/internal/middleware"
	systemsvc "software-web-manager/backend/internal/services/system"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) GetSystemSettings(c *gin.Context) {
	svc := systemsvc.NewService(h.DB)
	if !svc.HasSettingsTable() {
		resp := systemsvc.BuildSettingsResponse(nil)
		resp.SSORedirectURI = common.SSODeriveRedirectURI(c)
		c.JSON(http.StatusOK, resp)
		return
	}
	items, err := svc.ListSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load system settings"})
		return
	}
	items = svc.RefreshServiceStatusHeartbeat(items)
	resp := systemsvc.BuildSettingsResponse(items)
	// Surface the effective callback URL (auto-derived from the backend domain
	// unless an explicit override was stored), so the admin can register it at
	// the IdP without typing it by hand.
	if strings.TrimSpace(resp.SSORedirectURI) == "" {
		resp.SSORedirectURI = common.SSODeriveRedirectURI(c)
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetPublicSettings(c *gin.Context) {
	svc := systemsvc.NewService(h.DB)
	if !svc.HasSettingsTable() {
		c.JSON(http.StatusOK, systemsvc.BuildSettingsResponse(nil))
		return
	}
	items, err := svc.ListSettings()
	if err != nil {
		c.JSON(http.StatusOK, systemsvc.BuildSettingsResponse(nil))
		return
	}
	items = svc.RefreshServiceStatusHeartbeat(items)
	c.JSON(http.StatusOK, systemsvc.BuildSettingsResponse(items))
}

func (h *Handler) UpdateSystemSettings(c *gin.Context) {
	var req systemsvc.UpdateSettingsRequest
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

	resp, err := systemsvc.NewService(h.DB).UpdateSettings(req, userUUID)
	if err != nil {
		var ve *systemsvc.ValidationError
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
	var req systemsvc.TestSMTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := systemsvc.NewService(h.DB).SendTestMail(req); err != nil {
		var ve *systemsvc.ValidationError
		if errors.As(err, &ve) {
			c.JSON(http.StatusBadRequest, gin.H{"error": ve.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
