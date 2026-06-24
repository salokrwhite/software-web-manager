package release

import (
	"net/http"
	"software-web-manager/backend/internal/api/common"
	"strings"
	"time"

	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type releaseTemplateRequest struct {
	Name        string     `json:"name"`
	ScheduleAt  *time.Time `json:"schedule_at"`
	WindowStart *time.Time `json:"window_start"`
	WindowEnd   *time.Time `json:"window_end"`
	Emergency   bool       `json:"emergency"`
}

func (h *Handler) ListReleaseTemplates(c *gin.Context) {
	orgID := c.GetString(middleware.ContextOrgID)
	var items []models.ReleaseTemplate
	if err := h.DB.Where("org_id = ?", orgID).Order("created_at desc").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list release templates"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) CreateReleaseTemplate(c *gin.Context) {
	orgID := c.GetString(middleware.ContextOrgID)
	if !common.RequirePermission(c, "release.manage") {
		return
	}
	var req releaseTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org id"})
		return
	}
	template := models.ReleaseTemplate{
		OrgID:       orgUUID,
		Name:        name,
		ScheduleAt:  req.ScheduleAt,
		WindowStart: req.WindowStart,
		WindowEnd:   req.WindowEnd,
		Emergency:   req.Emergency,
	}
	if err := h.DB.Create(&template).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create release template"})
		return
	}
	common.Audit(h.DB, c, "release_template.create", "release_template", template.ID, nil, template)
	c.JSON(http.StatusOK, gin.H{"template": template})
}

func (h *Handler) UpdateReleaseTemplate(c *gin.Context) {
	orgID := c.GetString(middleware.ContextOrgID)
	if !common.RequirePermission(c, "release.manage") {
		return
	}
	templateID := c.Param("id")
	var req releaseTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	var template models.ReleaseTemplate
	if err := h.DB.Where("id = ? AND org_id = ?", templateID, orgID).First(&template).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "release template not found"})
		return
	}
	updates := map[string]interface{}{
		"name":         name,
		"schedule_at":  req.ScheduleAt,
		"window_start": req.WindowStart,
		"window_end":   req.WindowEnd,
		"emergency":    req.Emergency,
	}
	before := template
	if err := h.DB.Model(&models.ReleaseTemplate{}).Where("id = ?", templateID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update release template"})
		return
	}
	_ = h.DB.Where("id = ?", templateID).First(&template).Error
	common.Audit(h.DB, c, "release_template.update", "release_template", template.ID, before, template)
	c.JSON(http.StatusOK, gin.H{"template": template})
}

func (h *Handler) DeleteReleaseTemplate(c *gin.Context) {
	orgID := c.GetString(middleware.ContextOrgID)
	if !common.RequirePermission(c, "release.manage") {
		return
	}
	templateID := c.Param("id")
	var template models.ReleaseTemplate
	if err := h.DB.Where("id = ? AND org_id = ?", templateID, orgID).First(&template).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "release template not found"})
		return
	}
	var count int64
	if err := h.DB.Model(&models.Release{}).Where("release_template_id = ?", templateID).Count(&count).Error; err == nil && count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "release_template_in_use"})
		return
	}
	if err := h.DB.Where("id = ?", templateID).Delete(&models.ReleaseTemplate{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete release template"})
		return
	}
	common.Audit(h.DB, c, "release_template.delete", "release_template", template.ID, template, nil)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
