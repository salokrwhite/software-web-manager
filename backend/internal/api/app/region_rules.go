package app

import (
	"encoding/json"
	"net/http"
	"strings"

	"software-web-manager/backend/internal/api/common"
	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
)

type updateRegionRulesRequest struct {
	RegionRules json.RawMessage `json:"region_rules"`
}

func (h *Handler) GetAppRegionRules(c *gin.Context) {
	orgID := c.GetString(middleware.ContextOrgID)
	appID := c.Param("id")
	var app models.App
	if err := h.DB.Where("id = ? AND org_id = ?", appID, orgID).First(&app).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"region_rules": app.RegionRulesJSON})
}

func (h *Handler) UpdateAppRegionRules(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	appID := c.Param("id")
	if !h.HasPermission(c, "release.manage") && !h.HasAppPermission(userID, appID, "release.manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}
	orgID := c.GetString(middleware.ContextOrgID)
	app, err := h.GetAppForOrg(orgID, appID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	personal, err := h.IsPersonalOrg(orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load org"})
		return
	}
	if personal {
		status := strings.ToLower(strings.TrimSpace(app.Status))
		if status == "pending" {
			c.JSON(http.StatusForbidden, gin.H{"error": "app_pending_review", "status": status})
			return
		}
	}
	var req updateRegionRulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rules := common.NormalizeRegionRules(req.RegionRules)
	if err := h.DB.Model(&models.App{}).Where("id = ?", appID).Update("region_rules_json", rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update region rules"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"region_rules": rules})
}
