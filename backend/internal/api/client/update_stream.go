package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"software-web-manager/backend/internal/middleware"
	"strings"
	"time"

	"software-web-manager/backend/internal/models"
	"software-web-manager/backend/internal/services/clientupdate"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	clientUpdateKeepAliveInterval = 25 * time.Second
	clientUpdateDefaultReconnect  = 1500
)

type updateStreamQuery struct {
	DeviceID       string `form:"device_id" binding:"required"`
	ChannelCode    string `form:"channel_code"`
	Platform       string `form:"platform" binding:"required"`
	Arch           string `form:"arch" binding:"required"`
	CurrentVersion string `form:"current_version"`
	VersionCode    *int   `form:"version_code"`
}

func (h *Handler) HandleClientUpdateStream(c *gin.Context) {
	if h.ClientUpdateHub == nil {
		h.ClientUpdateHub = clientupdate.NewHub()
	}

	var req updateStreamQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Platform = strings.ToLower(strings.TrimSpace(req.Platform))
	req.Arch = strings.ToLower(strings.TrimSpace(req.Arch))
	req.ChannelCode = strings.ToLower(strings.TrimSpace(req.ChannelCode))
	req.DeviceID = strings.TrimSpace(req.DeviceID)

	app, _, ok := middleware.ClientAppOrgFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if h.CheckDeviceBlocked(c, app.ID, req.DeviceID) {
		return
	}

	if req.ChannelCode == "" {
		var channel models.Channel
		if err := h.DB.Where("app_id = ? AND is_default = true", app.ID).First(&channel).Error; err == nil {
			req.ChannelCode = channel.Code
		}
	}
	if req.ChannelCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel_code required"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Header("Access-Control-Allow-Origin", "*")

	connKey := fmt.Sprintf("%s|%s", c.ClientIP(), app.ID.String())
	sub := &clientupdate.Subscription{
		ConnKey:     connKey,
		OrgID:       app.OrgID.String(),
		AppID:       app.ID.String(),
		DeviceID:    req.DeviceID,
		ChannelCode: req.ChannelCode,
		Platform:    req.Platform,
		Arch:        req.Arch,
		Send:        make(chan clientupdate.Event, 32),
	}
	if ok := h.ClientUpdateHub.Subscribe(sub); !ok {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many stream connections"})
		return
	}
	defer h.ClientUpdateHub.Unsubscribe(sub.ID)

	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "stream unsupported"})
		return
	}

	_, _ = fmt.Fprintf(w, "event: connected\ndata: {\"ok\":true,\"reconnect_ms\":%d}\n\n", clientUpdateDefaultReconnect)
	flusher.Flush()

	keepAlive := time.NewTicker(clientUpdateKeepAliveInterval)
	defer keepAlive.Stop()

	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-keepAlive.C:
			_, _ = fmt.Fprint(w, ": keep-alive\n\n")
			flusher.Flush()
		case evt := <-sub.Send:
			data, err := json.Marshal(evt)
			if err != nil {
				continue
			}
			_, _ = fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", evt.ID, evt.EventType, data)
			flusher.Flush()
		}
	}
}

// EmitDeviceShutdown notifies a connected client that its device has been blocked
// so it can fail closed immediately, instead of waiting for the next update check.
func (h *Handler) EmitDeviceShutdown(appID uuid.UUID, deviceID, reason string) {
	if h == nil || h.ClientUpdateHub == nil || h.DB == nil {
		return
	}
	deviceID = strings.TrimSpace(deviceID)
	if appID == uuid.Nil || deviceID == "" {
		return
	}
	var app models.App
	if err := h.DB.Where("id = ?", appID).First(&app).Error; err != nil {
		return
	}
	h.ClientUpdateHub.Publish(clientupdate.Event{
		ID:          uuid.NewString(),
		EventType:   "device_shutdown",
		OrgID:       app.OrgID.String(),
		AppID:       app.ID.String(),
		DeviceID:    deviceID,
		ChannelCode: "",
		Platform:    "universal",
		Arch:        "universal",
		ReleaseID:   "",
		PublishedAt: time.Now(),
		Reason:      strings.TrimSpace(reason),
	})
}
