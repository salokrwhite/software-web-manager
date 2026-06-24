package core

import (
	"strings"

	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/services/audit"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Audit records an audit entry, taking the actor and org from the request
// context. The persistence lives in services/audit.
func (h *Handler) Audit(c *gin.Context, action, targetType string, targetID uuid.UUID, before any, after any) {
	orgID, err := uuid.Parse(c.GetString(middleware.ContextOrgID))
	if err != nil {
		return
	}
	userID, err := uuid.Parse(c.GetString(middleware.ContextUserID))
	if err != nil {
		return
	}
	_ = audit.NewService(h.DB).Record(orgID, userID, action, targetType, targetID,
		strings.TrimSpace(c.ClientIP()), strings.TrimSpace(c.GetHeader("User-Agent")), before, after)
}

// AuditWithOrg records an audit entry against an explicit org (the actor still
// comes from the request context).
func (h *Handler) AuditWithOrg(c *gin.Context, orgID uuid.UUID, action, targetType string, targetID uuid.UUID, before any, after any) {
	userID, err := uuid.Parse(c.GetString(middleware.ContextUserID))
	if err != nil {
		return
	}
	_ = audit.NewService(h.DB).Record(orgID, userID, action, targetType, targetID,
		strings.TrimSpace(c.ClientIP()), strings.TrimSpace(c.GetHeader("User-Agent")), before, after)
}
