package org

import (
	"software-web-manager/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

// Handler serves the org-domain endpoints (orgs, members, roles, invites, join requests).
type Handler struct {
	*handlers.Handler
}

// New builds an org handler over the shared core.
func New(core *handlers.Handler) *Handler {
	return &Handler{Handler: core}
}

// RegisterRoutes wires the authenticated org routes onto the API group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/orgs", h.ListOrgs)
	rg.POST("/orgs", h.CreateOrg)
	rg.GET("/orgs/:id/public", h.GetOrgPublic)
	rg.POST("/orgs/:id/join-requests", h.CreateOrgJoinRequest)
	rg.GET("/orgs/:id/join-requests", h.ListOrgJoinRequests)
	rg.POST("/orgs/:id/join-requests/batch-delete", h.BatchDeleteOrgJoinRequests)
	rg.POST("/orgs/:id/join-requests/:request_id/approve", h.ApproveOrgJoinRequest)
	rg.POST("/orgs/:id/join-requests/:request_id/reject", h.RejectOrgJoinRequest)
	rg.PATCH("/orgs/:id", h.UpdateOrg)
	rg.POST("/orgs/upgrade", h.UpgradeOrg)
	rg.POST("/orgs/:id/switch", h.SwitchOrg)
	rg.POST("/orgs/:id/transfer-owner", h.TransferOwner)
	rg.DELETE("/orgs/:id", h.DeleteOrg)
	rg.GET("/orgs/:id/members", h.ListOrgMembers)
	rg.POST("/orgs/:id/members", h.AddOrgMember)
	rg.PATCH("/orgs/:id/members/:user_id", h.UpdateOrgMember)
	rg.DELETE("/orgs/:id/members/:user_id", h.DeleteOrgMember)
	rg.GET("/orgs/:id/roles", h.ListOrgRoles)
	rg.POST("/orgs/:id/roles", h.CreateOrgRole)
	rg.PATCH("/orgs/:id/roles/:role_name", h.UpdateOrgRole)
	rg.DELETE("/orgs/:id/roles/:role_name", h.DeleteOrgRole)
	rg.GET("/orgs/:id/permissions", h.ListOrgPermissions)
	rg.GET("/orgs/:id/roles/:role_name/permissions", h.GetRolePermissions)
	rg.PUT("/orgs/:id/roles/:role_name/permissions", h.PutRolePermissions)
	rg.POST("/orgs/:id/users", h.CreateOrgUser)
	rg.POST("/orgs/:id/invites", h.CreateOrgInvite)
	rg.GET("/orgs/:id/invites", h.ListOrgInvites)
	rg.DELETE("/orgs/:id/invites/:invite_id", h.RevokeOrgInvite)
	rg.POST("/orgs/:id/invites/batch-delete", h.BatchDeleteOrgInvites)
	rg.GET("/org-invites", h.ListUserOrgInvites)
	rg.POST("/org-invites/accept-by-id/:invite_id", h.AcceptOrgInviteByID)
	rg.GET("/org-join-requests/mine", h.ListMyOrgJoinRequests)
	rg.POST("/org-join-requests/mine/batch-delete", h.BatchDeleteMyOrgJoinRequests)
	rg.POST("/org-join-requests/mine/batch-withdraw", h.BatchWithdrawMyOrgJoinRequests)
}
