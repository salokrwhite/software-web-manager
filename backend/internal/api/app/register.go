package app

import (
	"software-web-manager/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

// Handler serves the app-domain endpoints (apps, app channels, app members, app secrets).
type Handler struct {
	*handlers.Handler
}

// New builds an app handler over the shared core.
func New(core *handlers.Handler) *Handler {
	return &Handler{Handler: core}
}

// RegisterRoutes wires the app routes onto the authenticated API group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/apps", h.ListApps)
	rg.POST("/apps", h.CreateApp)
	rg.GET("/apps/:id", h.GetApp)
	rg.PATCH("/apps/:id", h.UpdateApp)
	rg.DELETE("/apps/:id", h.DeleteApp)
	rg.POST("/apps/:id/channels", h.CreateChannel)
	rg.GET("/apps/:id/channels", h.ListChannels)
	rg.DELETE("/apps/:id/channels/:channel_id", h.DeleteChannel)
	rg.GET("/apps/:id/members", h.ListAppMembers)
	rg.POST("/apps/:id/members", h.AddAppMember)
	rg.GET("/apps/:id/region-rules", h.GetAppRegionRules)
	rg.PATCH("/apps/:id/region-rules", h.UpdateAppRegionRules)
	rg.POST("/apps/:id/app-secrets", h.CreateAppSecret)
	rg.GET("/apps/:id/app-secrets", h.ListAppSecrets)
	rg.PATCH("/app-secrets/:id/policy", h.UpdateAppSecretPolicy)
	rg.DELETE("/app-secrets/:id", h.RevokeAppSecret)
}
