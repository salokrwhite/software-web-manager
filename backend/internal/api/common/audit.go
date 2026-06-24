package common

import (
	"strings"

	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/services/audit"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Audit records an audit entry, taking the actor and org from the request
// context. The persistence lives in services/audit.
func Audit(db *gorm.DB, c *gin.Context, action, targetType string, targetID uuid.UUID, before any, after any) {
	orgID, err := uuid.Parse(c.GetString(middleware.ContextOrgID))
	if err != nil {
		return
	}
	userID, err := uuid.Parse(c.GetString(middleware.ContextUserID))
	if err != nil {
		return
	}
	_ = audit.NewService(db).Record(orgID, userID, action, targetType, targetID,
		strings.TrimSpace(c.ClientIP()), strings.TrimSpace(c.GetHeader("User-Agent")), before, after)
}

// AuditWithOrg records an audit entry against an explicit org (the actor still
// comes from the request context).
func AuditWithOrg(db *gorm.DB, c *gin.Context, orgID uuid.UUID, action, targetType string, targetID uuid.UUID, before any, after any) {
	userID, err := uuid.Parse(c.GetString(middleware.ContextUserID))
	if err != nil {
		return
	}
	_ = audit.NewService(db).Record(orgID, userID, action, targetType, targetID,
		strings.TrimSpace(c.ClientIP()), strings.TrimSpace(c.GetHeader("User-Agent")), before, after)
}
