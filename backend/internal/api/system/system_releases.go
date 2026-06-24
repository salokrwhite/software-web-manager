package system

import (
	"net/http"
	"software-web-manager/backend/internal/handlers"
	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) ListSystemReleases(c *gin.Context) {
	orgType := strings.ToLower(strings.TrimSpace(c.Query("org_type")))
	status := strings.ToLower(strings.TrimSpace(c.Query("status")))
	q := strings.TrimSpace(c.Query("q"))
	limit := 50
	offset := 0
	if v := c.Query("limit"); v != "" {
		if n, err := handlers.ParseInt(v); err == nil && n > 0 {
			if n > 200 {
				n = 200
			}
			limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := handlers.ParseInt(v); err == nil && n >= 0 {
			offset = n
		}
	}

	args := []any{}
	where := "WHERE 1=1"
	hasOrgTypeColumn := h.HasOrgTypeColumn()
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
		where += " AND r.status = ?"
		args = append(args, status)
	}
	if q != "" {
		like := "%" + q + "%"
		where += " AND (a.name LIKE ? OR a.slug LIKE ? OR r.version LIKE ?)"
		args = append(args, like, like, like)
	}
	args = append(args, limit, offset)

	selectCols := []string{
		"r.id",
		"r.version",
		"r.status",
		"r.submitted_at",
		"r.created_at",
		"a.id as app_id",
		"a.name as app_name",
		"a.slug as app_slug",
		"o.id as org_id",
		"o.name as org_name",
	}
	if hasOrgTypeColumn {
		selectCols = append(selectCols, "o.org_type as org_type")
	} else {
		selectCols = append(selectCols, "'' as org_type")
	}
	selectCols = append(selectCols,
		"(SELECT u.email FROM memberships om "+
			"JOIN users u ON u.id = om.user_id "+
			"WHERE om.scope_type = 'org' AND om.scope_id = o.id AND om.role = 'owner' "+
			"ORDER BY om.created_at ASC "+
			"LIMIT 1) AS owner_email",
	)

	query := `
		SELECT ` + strings.Join(selectCols, ", ") + `
		FROM releases r
		JOIN apps a ON a.id = r.app_id
		JOIN orgs o ON o.id = a.org_id
	` + where + `
		ORDER BY r.created_at DESC
		LIMIT ? OFFSET ?
	`
	var items []systemReleaseItem
	if err := h.DB.Raw(query, args...).Scan(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list releases"})
		return
	}
	if len(items) > 0 {
		ids := make([]string, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.ID.String())
		}
		notes := h.LoadSubmitNotes("release", "release.submit", ids)
		for i := range items {
			if note, ok := notes[items[i].ID.String()]; ok {
				items[i].SubmitNote = note
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) ApproveSystemRelease(c *gin.Context) {
	releaseID := strings.TrimSpace(c.Param("id"))
	if releaseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "release_id required"})
		return
	}
	adminID, err := uuid.Parse(c.GetString(middleware.ContextUserID))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	type releaseOrgRow struct {
		ID      uuid.UUID
		Status  string
		OrgID   uuid.UUID
		OrgType string
	}
	row := releaseOrgRow{}
	selectCols := []string{
		"r.id",
		"r.status",
		"a.org_id as org_id",
	}
	if h.HasOrgTypeColumn() {
		selectCols = append(selectCols, "o.org_type as org_type")
	} else {
		selectCols = append(selectCols, "'' as org_type")
	}
	query := `
		SELECT ` + strings.Join(selectCols, ", ") + `
		FROM releases r
		JOIN apps a ON a.id = r.app_id
		JOIN orgs o ON o.id = a.org_id
		WHERE r.id = ?
	`
	if err := h.DB.Raw(query, releaseID).Scan(&row).Error; err != nil || row.ID == (uuid.UUID{}) {
		c.JSON(http.StatusNotFound, gin.H{"error": "release not found"})
		return
	}
	if h.HasOrgTypeColumn() && strings.ToLower(strings.TrimSpace(row.OrgType)) != "personal" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org not personal"})
		return
	}
	if strings.ToLower(strings.TrimSpace(row.Status)) != "in_review" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "release not in review"})
		return
	}

	now := time.Now()
	if err := h.DB.Model(&models.Release{}).Where("id = ?", releaseID).
		Updates(map[string]any{"status": "approved", "approved_at": now, "approved_by": adminID}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to approve release"})
		return
	}
	h.AuditWithOrg(c, row.OrgID, "system.release.approve", "release", row.ID, nil, gin.H{"status": "approved"})
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) RejectSystemRelease(c *gin.Context) {
	releaseID := strings.TrimSpace(c.Param("id"))
	if releaseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "release_id required"})
		return
	}

	type releaseOrgRow struct {
		ID      uuid.UUID
		Status  string
		OrgID   uuid.UUID
		OrgType string
	}
	row := releaseOrgRow{}
	selectCols := []string{
		"r.id",
		"r.status",
		"a.org_id as org_id",
	}
	if h.HasOrgTypeColumn() {
		selectCols = append(selectCols, "o.org_type as org_type")
	} else {
		selectCols = append(selectCols, "'' as org_type")
	}
	query := `
		SELECT ` + strings.Join(selectCols, ", ") + `
		FROM releases r
		JOIN apps a ON a.id = r.app_id
		JOIN orgs o ON o.id = a.org_id
		WHERE r.id = ?
	`
	if err := h.DB.Raw(query, releaseID).Scan(&row).Error; err != nil || row.ID == (uuid.UUID{}) {
		c.JSON(http.StatusNotFound, gin.H{"error": "release not found"})
		return
	}
	if h.HasOrgTypeColumn() && strings.ToLower(strings.TrimSpace(row.OrgType)) != "personal" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org not personal"})
		return
	}
	if strings.ToLower(strings.TrimSpace(row.Status)) != "in_review" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "release not in review"})
		return
	}

	if err := h.DB.Model(&models.Release{}).Where("id = ?", releaseID).
		Update("status", "rejected").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reject release"})
		return
	}
	h.AuditWithOrg(c, row.OrgID, "system.release.reject", "release", row.ID, nil, gin.H{"status": "rejected"})
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) BatchDeleteSystemReleaseApprovalLogs(c *gin.Context) {
	var req batchDeleteSystemReleaseApprovalLogsRequest
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

	actions := []string{"system.release.approve", "system.release.reject"}
	res := h.DB.Where("target_type = ? AND target_id IN ? AND action IN ?", "release", ids, actions).Delete(&models.AuditLog{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete audit logs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": res.RowsAffected})
}
