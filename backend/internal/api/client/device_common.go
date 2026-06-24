package client

import (
	"strings"
	"time"

	"software-web-manager/backend/internal/api/common"
	"software-web-manager/backend/internal/models"

	"github.com/google/uuid"
)

// NormalizeAttributes lower-cases attribute keys and stringifies their values.
func NormalizeAttributes(attrs map[string]interface{}) map[string]string {
	out := map[string]string{}
	for k, v := range attrs {
		key := strings.ToLower(strings.TrimSpace(k))
		if key == "" {
			continue
		}
		out[key] = strings.TrimSpace(common.ToString(v))
	}
	return out
}

func (h *Handler) GetChannelMinVersion(appID uuid.UUID, channelCode string) string {
	var channel models.Channel
	if err := h.DB.Where("app_id = ? AND code = ?", appID, channelCode).First(&channel).Error; err != nil {
		return ""
	}
	return channel.MinSupportedVersion
}

func (h *Handler) UpsertDevice(appID uuid.UUID, deviceID, platform, arch string, attrs map[string]string, appVersion, ip string) error {
	now := time.Now()
	if platform == "" {
		platform = attrs["platform"]
	}
	if arch == "" {
		arch = attrs["arch"]
	}
	if appVersion == "" {
		appVersion = attrs["app_version"]
	}
	var device models.Device
	if err := h.DB.Where("app_id = ? AND device_id = ?", appID, deviceID).First(&device).Error; err == nil {
		err = h.DB.Model(&device).Updates(map[string]interface{}{
			"last_seen_at": now,
			"platform":     platform,
			"arch":         arch,
			"os_version":   attrs["os_version"],
			"country":      attrs["country"],
			"app_version":  appVersion,
			"user_id":      attrs["user_id"],
			"last_ip":      ip,
		}).Error
		if err == nil && h.OnlineTracker != nil {
			h.OnlineTracker.Touch(appID, deviceID, now)
		}
		return err
	}
	device = models.Device{
		AppID:      appID,
		DeviceID:   deviceID,
		Platform:   platform,
		Arch:       arch,
		OSVersion:  attrs["os_version"],
		Country:    attrs["country"],
		AppVersion: appVersion,
		UserID:     attrs["user_id"],
		LastIP:     ip,
	}
	if err := h.DB.Create(&device).Error; err != nil {
		return err
	}
	if h.OnlineTracker != nil {
		h.OnlineTracker.Touch(appID, deviceID, now)
	}
	return nil
}
