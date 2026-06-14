package handlers

import (
	"net/http"
	"strings"

	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ListDevices(c *gin.Context) {
	if !h.requirePermission(c, PermissionRoleViewer) {
		return
	}
	orgID := c.GetString(middleware.ContextOrgID)
	appID := c.Query("app_id")

	db := h.DB.Model(&models.Device{})
	if appID != "" {
		if _, err := h.getAppForOrg(orgID, appID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
			return
		}
		db = db.Where("app_id = ?", appID)
	} else {
		db = db.Joins("JOIN apps a ON a.id = devices.app_id").Where("a.org_id = ?", orgID)
	}

	if v := c.Query("platform"); v != "" {
		db = db.Where("platform = ?", v)
	}
	if v := c.Query("country"); v != "" {
		db = db.Where("country = ?", v)
	}
	if v := strings.TrimSpace(c.Query("device_id")); v != "" {
		db = db.Where("device_id = ?", v)
	}
	if v := strings.TrimSpace(c.Query("last_ip")); v != "" {
		db = db.Where("last_ip = ?", v)
	}
	if v := c.Query("from"); v != "" {
		if t, err := parseTimeFlexible(v); err == nil {
			db = db.Where("last_seen_at >= ?", t)
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := parseTimeFlexible(v); err == nil {
			db = db.Where("last_seen_at <= ?", t)
		}
	}

	limit := 100
	if v := c.Query("limit"); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 {
			if n > 500 {
				n = 500
			}
			limit = n
		}
	}

	var items []models.Device
	if err := db.Order("last_seen_at desc").Limit(limit).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list devices"})
		return
	}
	h.backfillDeviceCountries(items)
	c.JSON(http.StatusOK, gin.H{"items": items})
}

type batchDeleteDevicesRequest struct {
	IDs []string `json:"ids"`
}

func (h *Handler) BatchDeleteDevices(c *gin.Context) {
	orgID := strings.TrimSpace(c.GetString(middleware.ContextOrgID))
	if orgID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var req batchDeleteDevicesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ids := normalizeDeviceIDs(req.IDs)
	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
		return
	}

	var scopedIDs []string
	if err := h.DB.Table("devices d").
		Select("d.id").
		Joins("JOIN apps a ON a.id = d.app_id").
		Where("a.org_id = ? AND d.id IN ?", orgID, ids).
		Scan(&scopedIDs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load devices"})
		return
	}
	if len(scopedIDs) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "devices not found"})
		return
	}

	if err := h.DB.Where("id IN ?", scopedIDs).Delete(&models.Device{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete devices"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "deleted": len(scopedIDs)})
}

func normalizeDeviceIDs(ids []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		clean := strings.TrimSpace(id)
		if clean == "" {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	return out
}

func (h *Handler) backfillDeviceCountries(items []models.Device) {
	if h == nil || h.RegionResolver == nil {
		return
	}
	for i := range items {
		if strings.TrimSpace(items[i].Country) != "" {
			continue
		}
		ip := strings.TrimSpace(items[i].LastIP)
		if ip == "" {
			continue
		}
		region, err := h.RegionResolver.Resolve(ip)
		if err != nil {
			continue
		}
		country := strings.TrimSpace(region.Country)
		if country == "" {
			country = strings.TrimSpace(region.ISO)
		}
		if country == "" {
			continue
		}
		items[i].Country = country
		_ = h.DB.Model(&models.Device{}).Where("id = ?", items[i].ID).Update("country", country).Error
	}
}

