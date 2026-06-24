package profile

import (
	"software-web-manager/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

// Handler serves the profile-domain endpoints (org-user profile and system-admin profile).
type Handler struct {
	*handlers.Handler
}

// New builds a profile handler over the shared core.
func New(core *handlers.Handler) *Handler {
	return &Handler{Handler: core}
}

// RegisterRoutes wires the org-user profile routes onto the authenticated API group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/profile", h.GetProfile)
	rg.POST("/profile/password", h.UpdateProfilePassword)
	rg.POST("/profile/avatar", h.UpdateProfileAvatar)
	rg.POST("/profile/2fa/setup", h.SetupProfile2FA)
	rg.POST("/profile/2fa/confirm", h.ConfirmProfile2FA)
	rg.POST("/profile/2fa/require", h.ToggleProfile2FA)
	rg.POST("/profile/2fa/disable", h.DisableProfile2FA)
}

// RegisterSystemRoutes wires the system-admin profile routes onto the system API group.
func (h *Handler) RegisterSystemRoutes(rg *gin.RouterGroup) {
	rg.GET("/profile", h.GetSystemProfile)
	rg.POST("/profile/password", h.UpdateSystemPassword)
	rg.POST("/profile/avatar", h.UpdateSystemAvatar)
}
