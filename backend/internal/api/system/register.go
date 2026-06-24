package system

import (
	"software-web-manager/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

// Handler serves the platform-admin (system) endpoints.
type Handler struct {
	*handlers.Handler
}

// New builds a system handler over the shared core.
func New(core *handlers.Handler) *Handler {
	return &Handler{Handler: core}
}

// RegisterRoutes wires the system-admin routes onto the system API group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/overview", h.SystemOverview)
	rg.GET("/orgs", h.ListSystemOrgs)
	rg.GET("/orgs/:id/materials", h.ListOrgRegistrationMaterials)
	rg.POST("/orgs", h.CreateSystemOrg)
	rg.POST("/orgs/:id/approve", h.ApproveSystemOrg)
	rg.POST("/orgs/:id/reject", h.RejectSystemOrg)
	rg.POST("/orgs/:id/disable", h.DisableSystemOrg)
	rg.POST("/orgs/batch-delete", h.BatchDeleteSystemOrgs)
	rg.POST("/apps/batch-delete", h.BatchDeleteSystemApps)
	rg.GET("/apps", h.ListSystemApps)
	rg.POST("/apps/:id/approve", h.ApproveSystemApp)
	rg.POST("/apps/:id/reject", h.RejectSystemApp)
	rg.POST("/apps/:id/disable", h.DisableSystemApp)
	rg.POST("/apps/:id/enable", h.EnableSystemApp)
	rg.GET("/releases", h.ListSystemReleases)
	rg.POST("/releases/:id/approve", h.ApproveSystemRelease)
	rg.POST("/releases/:id/reject", h.RejectSystemRelease)
	rg.GET("/audit-logs", h.ListSystemAuditLogs)
	rg.POST("/audit-logs/delete", h.DeleteSystemAuditLogs)
	rg.POST("/audit-logs/org-approvals/batch-delete", h.BatchDeleteSystemOrgApprovalLogs)
	rg.POST("/audit-logs/app-approvals/batch-delete", h.BatchDeleteSystemAppApprovalLogs)
	rg.POST("/audit-logs/release-approvals/batch-delete", h.BatchDeleteSystemReleaseApprovalLogs)
	rg.GET("/users", h.ListSystemUsers)
	rg.PATCH("/users/:id", h.UpdateSystemUser)
	rg.POST("/users/:id/enable", h.EnableSystemUser)
	rg.POST("/users/:id/disable", h.DisableSystemUser)
	rg.POST("/users/:id/reset-password", h.ResetSystemUserPassword)
	rg.POST("/users/batch-delete", h.BatchDeleteSystemUsers)
	rg.POST("/impersonate", h.Impersonate)
}
