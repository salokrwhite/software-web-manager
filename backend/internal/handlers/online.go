package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"software-web-manager/backend/internal/auth"
	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) GetOnlineCount(c *gin.Context) {
	orgID := c.GetString(middleware.ContextOrgID)
	appID := strings.TrimSpace(c.Param("id"))
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id required"})
		return
	}
	app, err := h.getAppForOrg(orgID, appID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	if !h.hasAppOnlineEnabledColumn() {
		app.OnlineEnabled = true
	}

	windowSeconds := h.Cfg.OnlineWindowSeconds
	if windowSeconds <= 0 {
		windowSeconds = 120
	}
	now := time.Now()
	count := int64(0)
	if app.OnlineEnabled {
		count = h.countOnlineForApp(app.ID, now, windowSeconds)
	}

	c.JSON(http.StatusOK, gin.H{
		"online":         count,
		"window_seconds": windowSeconds,
		"server_time":    now.Format(time.RFC3339),
	})
}

func (h *Handler) StreamOnlineCount(c *gin.Context) {
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token required"})
		return
	}
	claims, err := auth.ParseToken(h.Cfg.JWTSecret, token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	userID := strings.TrimSpace(claims.UserID)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	var user models.User
	if err := h.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	if strings.ToLower(strings.TrimSpace(user.Status)) != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "user not active", "code": userStatusCode(user.Status)})
		return
	}

	systemRole := strings.ToLower(strings.TrimSpace(claims.SystemRole))
	orgID := strings.TrimSpace(claims.OrgID)
	var app models.App
	appID := strings.TrimSpace(c.Param("id"))
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id required"})
		return
	}
	if systemRole == "system_admin" {
		if err := h.DB.Where("id = ?", appID).First(&app).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
			return
		}
	} else {
		if orgID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid org"})
			return
		}
		var org models.Org
		if err := h.DB.Where("id = ?", orgID).First(&org).Error; err == nil {
			if strings.ToLower(strings.TrimSpace(org.Status)) != "active" {
				c.JSON(http.StatusForbidden, gin.H{"error": "org not active", "code": orgStatusCode(org.Status)})
				return
			}
		}
		app, err = h.getAppForOrg(orgID, appID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
			return
		}
	}
	if !h.hasAppOnlineEnabledColumn() {
		app.OnlineEnabled = true
	}

	windowSeconds := h.Cfg.OnlineWindowSeconds
	if windowSeconds <= 0 {
		windowSeconds = 120
	}
	intervalSeconds := h.Cfg.OnlineStreamIntervalSeconds
	if intervalSeconds <= 0 {
		intervalSeconds = 3
	}

	writer := c.Writer
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	writer.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "stream not supported"})
		return
	}

	send := func() {
		now := time.Now()
		count := int64(0)
		if app.OnlineEnabled {
			count = h.countOnlineForApp(app.ID, now, windowSeconds)
		}
		payload, _ := json.Marshal(gin.H{
			"online":         count,
			"window_seconds": windowSeconds,
			"server_time":    now.Format(time.RFC3339),
		})
		fmt.Fprintf(writer, "data: %s\n\n", payload)
		flusher.Flush()
	}

	send()
	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			send()
		}
	}
}

func (h *Handler) countOnlineForApp(appID uuid.UUID, now time.Time, windowSeconds int) int64 {
	cutoff := now.Add(-time.Duration(windowSeconds) * time.Second)
	db := h.DB.Model(&models.Device{}).
		Where("app_id = ? AND last_seen_at >= ?", appID, cutoff)
	if h.hasDeviceControlsTable() {
		db = db.Where(`
			NOT EXISTS (
				SELECT 1 FROM device_controls dc
				WHERE dc.app_id = devices.app_id AND dc.device_id = devices.device_id AND dc.blocked = 1
			)
		`)
	}
	var count int64
	_ = db.Count(&count).Error
	return count
}

type onlineDeviceItem struct {
	ID         string    `json:"id"`
	DeviceID   string    `json:"device_id"`
	Platform   string    `json:"platform"`
	Arch       string    `json:"arch"`
	OSVersion  string    `json:"os_version"`
	Country    string    `json:"country"`
	AppVersion string    `json:"app_version"`
	UserID     string    `json:"user_id"`
	LastIP     string    `json:"last_ip"`
	LastSeenAt time.Time `json:"last_seen_at"`
}

func (h *Handler) ListOnlineDevices(c *gin.Context) {
	orgID := c.GetString(middleware.ContextOrgID)
	appID := strings.TrimSpace(c.Param("id"))
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id required"})
		return
	}
	app, err := h.getAppForOrg(orgID, appID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	if !h.hasAppOnlineEnabledColumn() {
		app.OnlineEnabled = true
	}

	page := 1
	pageSize := 20
	if v := c.Query("page"); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := c.Query("page_size"); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 {
			if n > 200 {
				n = 200
			}
			pageSize = n
		}
	}
	offset := (page - 1) * pageSize

	windowSeconds := h.Cfg.OnlineWindowSeconds
	if windowSeconds <= 0 {
		windowSeconds = 120
	}
	now := time.Now()
	if !app.OnlineEnabled {
		c.JSON(http.StatusOK, gin.H{
			"items":          []onlineDeviceItem{},
			"total":          0,
			"page":           page,
			"page_size":      pageSize,
			"window_seconds": windowSeconds,
			"server_time":    now.Format(time.RFC3339),
		})
		return
	}

	cutoff := now.Add(-time.Duration(windowSeconds) * time.Second)
	var total int64
	db := h.DB.Model(&models.Device{}).Where("app_id = ? AND last_seen_at >= ?", app.ID, cutoff)
	if h.hasDeviceControlsTable() {
		db = db.Where(`
			NOT EXISTS (
				SELECT 1 FROM device_controls dc
				WHERE dc.app_id = devices.app_id AND dc.device_id = devices.device_id AND dc.blocked = 1
			)
		`)
	}
	if err := db.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count devices"})
		return
	}
	var devices []models.Device
	if err := db.Order("last_seen_at desc").Limit(pageSize).Offset(offset).Find(&devices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list devices"})
		return
	}
	items := make([]onlineDeviceItem, 0, len(devices))
	for _, d := range devices {
		items = append(items, onlineDeviceItem{
			ID:         d.ID.String(),
			DeviceID:   d.DeviceID,
			Platform:   d.Platform,
			Arch:       d.Arch,
			OSVersion:  d.OSVersion,
			Country:    d.Country,
			AppVersion: d.AppVersion,
			UserID:     d.UserID,
			LastIP:     d.LastIP,
			LastSeenAt: d.LastSeenAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"items":          items,
		"total":          total,
		"page":           page,
		"page_size":      pageSize,
		"window_seconds": windowSeconds,
		"server_time":    now.Format(time.RFC3339),
	})
}

