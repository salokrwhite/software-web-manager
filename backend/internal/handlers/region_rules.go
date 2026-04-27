package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
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
	if !h.hasPermission(c, "release.manage") && !h.hasAppPermission(userID, appID, "release.manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}
	orgID := c.GetString(middleware.ContextOrgID)
	app, err := h.getAppForOrg(orgID, appID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	personal, err := h.isPersonalOrg(orgID)
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
	rules := normalizeRegionRules(req.RegionRules)
	if err := h.DB.Model(&models.App{}).Where("id = ?", appID).Update("region_rules_json", rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update region rules"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"region_rules": rules})
}

func normalizeRegionRules(raw json.RawMessage) datatypes.JSON {
	if len(raw) == 0 {
		return nil
	}
	if strings.EqualFold(strings.TrimSpace(string(raw)), "null") {
		return nil
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	payload = normalizeRegionRulesMap(payload)
	b, _ := json.Marshal(payload)
	return datatypes.JSON(b)
}

func normalizeRegionRulesValue(value interface{}) datatypes.JSON {
	if value == nil {
		return nil
	}
	m, ok := value.(map[string]interface{})
	if ok {
		b, _ := json.Marshal(normalizeRegionRulesMap(m))
		return datatypes.JSON(b)
	}
	b, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(b, &payload); err != nil {
		return nil
	}
	payload = normalizeRegionRulesMap(payload)
	out, _ := json.Marshal(payload)
	return datatypes.JSON(out)
}

func normalizeRegionRulesMap(payload map[string]interface{}) map[string]interface{} {
	if payload == nil {
		return map[string]interface{}{
			"mode":  "allow_deny",
			"allow": normalizeRegionRuleGroup(nil),
			"deny":  normalizeRegionRuleGroup(nil),
		}
	}
	payload["mode"] = "allow_deny"
	allow := normalizeRegionRuleGroup(payload["allow"])
	deny := normalizeRegionRuleGroup(payload["deny"])
	payload["allow"] = allow
	payload["deny"] = deny
	if templates, ok := payload["templates"].([]interface{}); ok {
		normTemplates := make([]map[string]interface{}, 0, len(templates))
		for _, t := range templates {
			tm, ok := t.(map[string]interface{})
			if !ok {
				continue
			}
			id := strings.TrimSpace(toString(tm["id"]))
			name := strings.TrimSpace(toString(tm["name"]))
			normTemplates = append(normTemplates, map[string]interface{}{
				"id":    id,
				"name":  name,
				"allow": normalizeRegionRuleGroup(tm["allow"]),
				"deny":  normalizeRegionRuleGroup(tm["deny"]),
			})
		}
		payload["templates"] = normTemplates
		if _, ok := payload["active_template_id"]; !ok {
			payload["active_template_id"] = ""
		}
	}
	return payload
}

func normalizeRegionRuleGroup(value interface{}) map[string][]string {
	out := map[string][]string{
		"countries": {},
		"provinces": {},
		"cities":    {},
	}
	m, ok := value.(map[string]interface{})
	if !ok {
		return out
	}
	out["countries"] = normalizeStringList(m["countries"])
	out["provinces"] = normalizeStringList(m["provinces"])
	out["cities"] = normalizeStringList(m["cities"])
	return out
}

func normalizeStringList(value interface{}) []string {
	items := []string{}
	switch v := value.(type) {
	case []interface{}:
		for _, item := range v {
			s := strings.TrimSpace(toString(item))
			if s != "" {
				items = append(items, s)
			}
		}
	case []string:
		for _, item := range v {
			s := strings.TrimSpace(item)
			if s != "" {
				items = append(items, s)
			}
		}
	}
	return items
}




