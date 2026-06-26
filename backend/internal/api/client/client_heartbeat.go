package client

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"software-web-manager/backend/internal/middleware"
	"strings"
	"time"
)

type heartbeatRequest struct {
	DeviceID    string                 `json:"device_id"`
	ChannelCode string                 `json:"channel_code"`
	AppVersion  string                 `json:"app_version"`
	Platform    string                 `json:"platform"`
	Arch        string                 `json:"arch"`
	UserID      string                 `json:"user_id"`
	Attributes  map[string]interface{} `json:"attributes"`
}

func (h *Handler) ClientHeartbeat(c *gin.Context) {
	app, _, ok := middleware.ClientAppOrgFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req heartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.DeviceID = strings.TrimSpace(req.DeviceID)
	if req.DeviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id required"})
		return
	}
	if h.CheckDeviceBlocked(c, app.ID, req.DeviceID) {
		return
	}

	attrs := NormalizeAttributes(req.Attributes)
	if req.UserID != "" {
		attrs["user_id"] = req.UserID
	}
	if req.AppVersion != "" {
		attrs["app_version"] = req.AppVersion
	}
	if req.Platform != "" {
		attrs["platform"] = req.Platform
	}
	if req.Arch != "" {
		attrs["arch"] = req.Arch
	}
	esa := h.readESAGeo(c)
	realIP := esa.realIPOr(c.ClientIP())
	region := h.ResolveRegion(esa, attrs, realIP)
	if region.ISO != "" && attrs["country_iso"] == "" {
		attrs["country_iso"] = region.ISO
	}
	if attrs["country"] == "" {
		if region.Country != "" {
			attrs["country"] = region.Country
		} else if region.ISO != "" {
			attrs["country"] = region.ISO
		}
	}

	_ = h.UpsertDevice(app.ID, req.DeviceID, req.Platform, req.Arch, attrs, req.AppVersion, realIP)
	if h.OnlineTracker != nil {
		h.OnlineTracker.Touch(app.ID, req.DeviceID, time.Now())
	}

	resp := gin.H{
		"ok":          true,
		"server_time": time.Now().Format(time.RFC3339),
		"maintenance": h.BuildMaintenanceInfo(app),
	}
	// Attach a fresh signed verdict so the client can periodically re-verify it is
	// still authorized (enables near-real-time remote revocation while running).
	if env := h.SignAuthzForRequest(c, app, req.DeviceID); env != nil {
		resp["authz"] = env
	}
	c.JSON(http.StatusOK, resp)
}
