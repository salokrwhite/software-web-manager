package handlers

import (
	"net/http"
	"strings"
	"time"

	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (h *Handler) ListSystemApps(c *gin.Context) {
	orgID := strings.TrimSpace(c.Query("org_id"))
	q := strings.TrimSpace(c.Query("q"))
	orgType := strings.ToLower(strings.TrimSpace(c.Query("org_type")))
	status := strings.ToLower(strings.TrimSpace(c.Query("status")))
	limit := 50
	offset := 0
	if v := c.Query("limit"); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 {
			if n > 200 {
				n = 200
			}
			limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := parseInt(v); err == nil && n >= 0 {
			offset = n
		}
	}

	args := []any{}
	where := "WHERE 1=1"
	migrator := h.DB.Migrator()
	hasOrgTypeColumn := h.hasOrgTypeColumn()
	hasStatus := migrator.HasColumn(&models.App{}, "status")
	hasSubmittedAt := migrator.HasColumn(&models.App{}, "submitted_at")
	hasRejectionReason := migrator.HasColumn(&models.App{}, "rejection_reason")
	if orgID != "" {
		where += " AND a.org_id = ?"
		args = append(args, orgID)
	}
	if orgType != "" {
		if hasOrgTypeColumn {
			if orgType == "personal" {
				where += " AND (o.org_type = ? OR o.org_type IS NULL OR o.org_type = '')"
				args = append(args, orgType)
			} else {
				where += " AND o.org_type = ?"
				args = append(args, orgType)
			}
		}
	}
	if status != "" {
		if hasStatus {
			where += " AND a.status = ?"
			args = append(args, status)
		} else if status != "active" {
			c.JSON(http.StatusOK, gin.H{"items": []systemAppItem{}})
			return
		}
	}
	if q != "" {
		where += " AND (a.name LIKE ? OR a.slug LIKE ?)"
		like := "%" + q + "%"
		args = append(args, like, like)
	}
	args = append(args, limit, offset)

	selectCols := []string{
		"a.id",
		"a.name",
		"a.slug",
		"a.org_id",
		"o.name as org_name",
		"o.status as org_status",
	}
	if hasOrgTypeColumn {
		selectCols = append(selectCols, "o.org_type as org_type")
	} else {
		selectCols = append(selectCols, "'' as org_type")
	}
	if hasStatus {
		selectCols = append(selectCols, "a.status")
	} else {
		selectCols = append(selectCols, "'active' AS status")
	}
	if hasSubmittedAt {
		selectCols = append(selectCols, "a.submitted_at")
	} else {
		selectCols = append(selectCols, "NULL AS submitted_at")
	}
	if hasRejectionReason {
		selectCols = append(selectCols, "a.rejection_reason")
	} else {
		selectCols = append(selectCols, "NULL AS rejection_reason")
	}
	selectCols = append(selectCols,
		"(SELECT u.email FROM memberships om "+
			"JOIN users u ON u.id = om.user_id "+
			"WHERE om.scope_type = 'org' AND om.scope_id = o.id AND om.role = 'owner' "+
			"ORDER BY om.created_at ASC "+
			"LIMIT 1) AS owner_email",
		"a.created_at",
		"(SELECT COUNT(*) FROM releases r WHERE r.app_id = a.id) AS release_count",
		"(SELECT COUNT(*) FROM memberships am WHERE am.scope_type = 'app' AND am.scope_id = a.id) AS member_count",
		"(SELECT COUNT(*) FROM devices d WHERE d.app_id = a.id) AS device_count",
	)

	query := `
		SELECT ` + strings.Join(selectCols, ", ") + `
		FROM apps a
		JOIN orgs o ON o.id = a.org_id
	` + where + `
		ORDER BY a.created_at DESC
		LIMIT ? OFFSET ?
	`
	var items []systemAppItem
	if err := h.DB.Raw(query, args...).Scan(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list apps"})
		return
	}
	if len(items) > 0 {
		ids := make([]string, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.ID.String())
		}
		notes := h.loadSubmitNotes("app", "app.submit", ids)
		for i := range items {
			if note, ok := notes[items[i].ID.String()]; ok {
				items[i].SubmitNote = note
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) ApproveSystemApp(c *gin.Context) {
	appID := strings.TrimSpace(c.Param("id"))
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id required"})
		return
	}
	adminID, err := uuid.Parse(c.GetString(middleware.ContextUserID))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	var app models.App
	if err := h.DB.Where("id = ?", appID).First(&app).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	var org models.Org
	if err := h.DB.Select("id", "org_type").Where("id = ?", app.OrgID).First(&org).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "org not found"})
		return
	}
	if !strings.EqualFold(strings.TrimSpace(org.OrgType), "personal") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org not personal"})
		return
	}
	if strings.ToLower(strings.TrimSpace(app.Status)) != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app not pending"})
		return
	}
	now := time.Now()
	before := app
	if err := h.DB.Model(&models.App{}).Where("id = ?", appID).Updates(map[string]any{
		"status":           "active",
		"approved_by":      adminID,
		"approved_at":      now,
		"rejection_reason": nil,
		"rejected_by":      nil,
		"rejected_at":      nil,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to approve app"})
		return
	}
	var after models.App
	if err := h.DB.Where("id = ?", appID).First(&after).Error; err == nil {
		h.auditWithOrg(c, after.OrgID, "system.app.approve", "app", after.ID, before, after)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) RejectSystemApp(c *gin.Context) {
	appID := strings.TrimSpace(c.Param("id"))
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id required"})
		return
	}
	var req rejectSystemAppRequest
	_ = c.ShouldBindJSON(&req)
	adminID, err := uuid.Parse(c.GetString(middleware.ContextUserID))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	var app models.App
	if err := h.DB.Where("id = ?", appID).First(&app).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	var org models.Org
	if err := h.DB.Select("id", "org_type").Where("id = ?", app.OrgID).First(&org).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "org not found"})
		return
	}
	if !strings.EqualFold(strings.TrimSpace(org.OrgType), "personal") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org not personal"})
		return
	}
	if strings.ToLower(strings.TrimSpace(app.Status)) != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app not pending"})
		return
	}
	now := time.Now()
	reason := ""
	if req.Reason != nil {
		reason = strings.TrimSpace(*req.Reason)
	}
	updates := map[string]any{
		"status":      "rejected",
		"rejected_by": adminID,
		"rejected_at": now,
		"approved_by": nil,
		"approved_at": nil,
	}
	if reason == "" {
		updates["rejection_reason"] = nil
	} else {
		updates["rejection_reason"] = reason
	}
	before := app
	if err := h.DB.Model(&models.App{}).Where("id = ?", appID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reject app"})
		return
	}
	var after models.App
	if err := h.DB.Where("id = ?", appID).First(&after).Error; err == nil {
		h.auditWithOrg(c, after.OrgID, "system.app.reject", "app", after.ID, before, after)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) BatchDeleteSystemApps(c *gin.Context) {
	var req batchDeleteSystemAppsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
		return
	}

	seen := make(map[string]struct{}, len(req.IDs))
	ids := make([]string, 0, len(req.IDs))
	for _, raw := range req.IDs {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, err := uuid.Parse(value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		ids = append(ids, value)
	}
	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
		return
	}

	var apps []models.App
	if err := h.DB.Where("id IN ?", ids).Find(&apps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load apps"})
		return
	}
	if len(apps) == 0 || len(apps) != len(ids) {
		c.JSON(http.StatusNotFound, gin.H{"error": "apps not found"})
		return
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		var releaseIDs []string
		if err := tx.Model(&models.Release{}).Where("app_id IN ?", ids).Pluck("id", &releaseIDs).Error; err != nil {
			return err
		}
		if len(releaseIDs) > 0 {
			if err := tx.Where("release_id IN ?", releaseIDs).Delete(&models.Artifact{}).Error; err != nil {
				return err
			}
			if err := tx.Where("release_id IN ?", releaseIDs).Delete(&models.ReleaseChannel{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("app_id IN ?", ids).Delete(&models.Release{}).Error; err != nil {
			return err
		}
		if err := tx.Where("app_id IN ?", ids).Delete(&models.Channel{}).Error; err != nil {
			return err
		}
		if err := tx.Where("scope_id IN ?", ids).Delete(&models.AppMember{}).Error; err != nil {
			return err
		}
		if tx.Migrator().HasTable(&models.Feedback{}) {
			var feedbackIDs []string
			if err := tx.Model(&models.Feedback{}).Where("app_id IN ?", ids).Pluck("id", &feedbackIDs).Error; err != nil {
				return err
			}
			if err := deleteAttachmentsByOwners(tx, attachmentOwnerFeedback, feedbackIDs); err != nil {
				return err
			}
			if err := tx.Where("app_id IN ?", ids).Delete(&models.Feedback{}).Error; err != nil {
				return err
			}
		}
		if tx.Migrator().HasTable(&models.AppSecret{}) {
			if err := tx.Where("app_id IN ?", ids).Delete(&models.AppSecret{}).Error; err != nil {
				return err
			}
		}
		return tx.Where("id IN ?", ids).Delete(&models.App{}).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete apps"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": len(ids)})
}

func (h *Handler) BatchDeleteSystemAppApprovalLogs(c *gin.Context) {
	var req batchDeleteSystemAppApprovalLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
		return
	}

	seen := make(map[string]struct{}, len(req.IDs))
	ids := make([]string, 0, len(req.IDs))
	for _, raw := range req.IDs {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, err := uuid.Parse(value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		ids = append(ids, value)
	}
	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
		return
	}

	actions := []string{"system.app.approve", "system.app.reject"}
	res := h.DB.Where("target_type = ? AND target_id IN ? AND action IN ?", "app", ids, actions).Delete(&models.AuditLog{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete audit logs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": res.RowsAffected})
}
