// Package audit holds the audit-log HTTP endpoints and their route registration.
// The cross-cutting audit-write helpers (Audit/AuditWithOrg) remain on the shared
// handlers core, since they are used by every domain.
package audit

import (
	"net/http"
	"strings"
	"time"

	"software-web-manager/backend/internal/api/common"
	"software-web-manager/backend/internal/core"
	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler serves the org-scoped audit-log endpoints.
type Handler struct {
	*core.Handler
}

// New builds an audit handler over the shared core.
func New(core *core.Handler) *Handler {
	return &Handler{Handler: core}
}

// RegisterRoutes wires the audit routes onto the authenticated API group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/audit-logs", h.ListAuditLogs)
	rg.POST("/audit-logs/delete", h.DeleteAuditLogs)
}

type auditLogItem struct {
	ID         string    `json:"id" gorm:"column:id"`
	OrgID      string    `json:"org_id" gorm:"column:org_id"`
	UserID     string    `json:"user_id" gorm:"column:user_id"`
	UserEmail  string    `json:"user_email" gorm:"column:user_email"`
	Action     string    `json:"action" gorm:"column:action"`
	TargetType string    `json:"target_type" gorm:"column:target_type"`
	TargetID   string    `json:"target_id" gorm:"column:target_id"`
	IPAddress  string    `json:"ip_address" gorm:"column:ip_address"`
	CreatedAt  time.Time `json:"created_at" gorm:"column:created_at"`
}

func (h *Handler) ListAuditLogs(c *gin.Context) {
	orgID := c.GetString(middleware.ContextOrgID)
	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	db := h.DB.Table("audit_logs al").
		Select("al.id, al.org_id, al.user_id, u.email as user_email, al.action, al.target_type, al.target_id, al.ip_address, al.created_at").
		Joins("LEFT JOIN users u ON u.id = al.user_id").
		Where("al.org_id = ?", orgID)
	if !common.HasPermission(c, "audit_log.view") {
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
			return
		}
		db = db.Where("al.user_id = ?", userID)
	}
	if v := c.Query("action"); v != "" {
		db = db.Where("al.action = ?", v)
	}
	if v := c.Query("target_type"); v != "" {
		db = db.Where("al.target_type = ?", v)
	}
	if v := c.Query("target_id"); v != "" {
		db = db.Where("al.target_id = ?", v)
	}
	if v := c.Query("from"); v != "" {
		if t, err := common.ParseTimeFlexible(v); err == nil {
			db = db.Where("al.created_at >= ?", t)
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := common.ParseTimeFlexible(v); err == nil {
			db = db.Where("al.created_at <= ?", t)
		}
	}
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := common.ParseInt(v); err == nil && n > 0 {
			if n > 200 {
				n = 200
			}
			limit = n
		}
	}
	var items []auditLogItem
	if err := db.Order("al.created_at desc").Limit(limit).Scan(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list audit logs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

type deleteAuditLogsRequest struct {
	IDs []string `json:"ids" binding:"required"`
}

func (h *Handler) DeleteAuditLogs(c *gin.Context) {
	if !common.RequirePermission(c, "audit_log.view") {
		return
	}
	orgID := c.GetString(middleware.ContextOrgID)
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}

	var req deleteAuditLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
		return
	}
	ids := make([]uuid.UUID, 0, len(req.IDs))
	for _, raw := range req.IDs {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		parsed, err := uuid.Parse(value)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		ids = append(ids, parsed)
	}
	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
		return
	}

	if err := h.DB.Where("org_id = ? AND id IN ?", orgID, ids).Delete(&models.AuditLog{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete audit logs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": len(ids)})
}
