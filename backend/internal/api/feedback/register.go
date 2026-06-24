package feedback

import (
	"software-web-manager/backend/internal/core"

	"github.com/gin-gonic/gin"
)

// Handler serves the feedback-domain endpoints (org-scoped management + client submit).
type Handler struct {
	*core.Handler
}

// New builds a feedback handler over the shared core.
func New(core *core.Handler) *Handler {
	return &Handler{Handler: core}
}

// RegisterRoutes wires the org-scoped feedback management routes onto the authenticated API group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/apps/:id/feedback", h.ListAppFeedback)
	rg.GET("/apps/:id/feedback/:fid", h.GetAppFeedbackDetail)
	rg.PATCH("/apps/:id/feedback/:fid", h.UpdateAppFeedback)
}
