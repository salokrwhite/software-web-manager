package client

import (
	"software-web-manager/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

// Handler serves client-domain endpoints (device management + client event/heartbeat ingest).
type Handler struct {
	*handlers.Handler
}

// New builds a client handler over the shared core.
func New(core *handlers.Handler) *Handler {
	return &Handler{Handler: core}
}

// RegisterRoutes wires the org-scoped device-management routes onto the authenticated API group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/devices", h.ListDevices)
	rg.POST("/devices/batch-delete", h.BatchDeleteDevices)
	rg.GET("/apps/:id/blocked-devices", h.ListBlockedDevices)
	rg.POST("/apps/:id/blocked-devices", h.BlockDeviceByDeviceID)
	rg.POST("/devices/:id/block", h.BlockDevice)
	rg.POST("/devices/:id/unblock", h.UnblockDevice)
}

// RegisterClientRoutes wires the public client (SDK) routes onto the client API group.
func (h *Handler) RegisterClientRoutes(rg *gin.RouterGroup) {
	rg.POST("/events", h.IngestEvents)
	rg.POST("/heartbeat", h.ClientHeartbeat)
}
