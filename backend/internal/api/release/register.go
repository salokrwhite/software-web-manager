package release

import (
	"software-web-manager/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

// Handler serves the release-domain endpoints (releases, release channels, templates, artifacts).
type Handler struct {
	*handlers.Handler
}

// New builds a release handler over the shared core.
func New(core *handlers.Handler) *Handler {
	return &Handler{Handler: core}
}

// RegisterRoutes wires the release routes onto the authenticated API group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/apps/:id/release-channels", h.ListReleaseChannels)
	rg.POST("/apps/:id/release-channels", h.CreateReleaseChannel)
	rg.GET("/apps/:id/release-channels/:rcId/metrics", h.ReleaseChannelMetrics)
	rg.POST("/apps/:id/releases", h.CreateRelease)
	rg.GET("/apps/:id/releases", h.ListReleases)
	rg.PATCH("/releases/:id", h.UpdateRelease)
	rg.PATCH("/releases/:id/template", h.SetReleaseTemplate)
	rg.POST("/releases/:id/submit", h.SubmitRelease)
	rg.POST("/releases/:id/approve", h.ApproveRelease)
	rg.POST("/releases/:id/reject", h.RejectRelease)
	rg.POST("/releases/:id/publish", h.PublishRelease)
	rg.POST("/releases/:id/rollback", h.RollbackRelease)
	rg.POST("/releases/:id/revoke", h.RevokeRelease)
	rg.DELETE("/releases/:id", h.DeleteRelease)
	rg.GET("/release-templates", h.ListReleaseTemplates)
	rg.POST("/release-templates", h.CreateReleaseTemplate)
	rg.PATCH("/release-templates/:id", h.UpdateReleaseTemplate)
	rg.DELETE("/release-templates/:id", h.DeleteReleaseTemplate)
	rg.PATCH("/release-channels/:id", h.UpdateReleaseChannel)
	rg.POST("/releases/:id/artifacts", h.UploadArtifact)
	rg.GET("/releases/:id/artifacts", h.ListArtifacts)
	rg.GET("/artifacts/:id/download", h.DownloadArtifact)
}
