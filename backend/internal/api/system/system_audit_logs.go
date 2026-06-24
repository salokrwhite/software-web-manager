package system

import (
	"net/http"
	"software-web-manager/backend/internal/api/common"
	"software-web-manager/backend/internal/models"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) ListSystemAuditLogs(c *gin.Context) {
	db := h.DB.Table("audit_logs al").
		Select("al.id, al.org_id, o.name as org_name, al.user_id, u.email as user_email, al.action, al.target_type, al.target_id, al.ip_address, al.created_at").
		Joins("LEFT JOIN orgs o ON o.id = al.org_id").
		Joins("LEFT JOIN users u ON u.id = al.user_id")

	if v := strings.TrimSpace(c.Query("org_id")); v != "" {
		db = db.Where("al.org_id = ?", v)
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

	var items []systemAuditItem
	if err := db.Order("al.created_at desc").Limit(limit).Scan(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list audit logs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) DeleteSystemAuditLogs(c *gin.Context) {
	var req deleteSystemAuditLogsRequest
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

	if err := h.DB.Where("id IN ?", ids).Delete(&models.AuditLog{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete audit logs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": len(ids)})
}
