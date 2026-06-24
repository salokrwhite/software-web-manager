package handlers

import (
	"net/http"
	"strings"

	"software-web-manager/backend/internal/models"
	"software-web-manager/backend/internal/services/device"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const deviceBlockedCode = "device_blocked"

func (h *Handler) WriteDeviceBlocked(c *gin.Context, reason *string) {
	message := "device is blocked"
	if reason != nil && strings.TrimSpace(*reason) != "" {
		message = "device is blocked: " + strings.TrimSpace(*reason)
	}
	c.JSON(http.StatusForbidden, gin.H{
		"error": gin.H{
			"code":    deviceBlockedCode,
			"message": message,
		},
	})
}

func (h *Handler) IsDeviceBlocked(appID uuid.UUID, deviceID string) (bool, *models.DeviceControl, error) {
	if h == nil || h.DB == nil {
		return false, nil, nil
	}
	return device.NewService(h.DB).IsBlocked(appID, deviceID)
}

func (h *Handler) CheckDeviceBlocked(c *gin.Context, appID uuid.UUID, deviceID string) bool {
	blocked, control, err := h.IsDeviceBlocked(appID, deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check device status"})
		return true
	}
	if blocked {
		h.WriteDeviceBlocked(c, control.Reason)
		return true
	}
	return false
}
