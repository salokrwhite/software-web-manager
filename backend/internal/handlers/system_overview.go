package handlers

import (
	"net/http"
	"strings"
	"github.com/gin-gonic/gin"
)

func (h *Handler) SystemOverview(c *gin.Context) {
	orgID := strings.TrimSpace(c.Query("org_id"))
	from, to := parseDateRange(c)

	orgs := map[string]int64{"total": 0, "pending": 0, "active": 0, "disabled": 0}
	users := map[string]int64{"total": 0, "pending": 0, "active": 0, "disabled": 0}
	apps := map[string]int64{"total": 0}
	devices := map[string]int64{"total": 0}
	events := map[string]int64{"total": 0}

	// org counts
	var orgRows []struct {
		Status string
		Count  int64
	}
	if orgID != "" {
		if err := h.DB.Raw(`SELECT status, COUNT(*) as count FROM orgs WHERE id = ? GROUP BY status`, orgID).Scan(&orgRows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load org stats"})
			return
		}
	} else {
		if err := h.DB.Raw(`SELECT status, COUNT(*) as count FROM orgs GROUP BY status`).Scan(&orgRows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load org stats"})
			return
		}
	}
	for _, row := range orgRows {
		key := strings.ToLower(strings.TrimSpace(row.Status))
		if key == "" {
			continue
		}
		orgs[key] = row.Count
		orgs["total"] += row.Count
	}

	// user counts
	var userRows []struct {
		Status string
		Count  int64
	}
	if orgID != "" {
		if err := h.DB.Raw(`
			SELECT u.status, COUNT(DISTINCT u.id) as count
			FROM users u
			JOIN org_members om ON om.user_id = u.id
			WHERE om.org_id = ?
			GROUP BY u.status
		`, orgID).Scan(&userRows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load user stats"})
			return
		}
	} else {
		if err := h.DB.Raw(`SELECT status, COUNT(*) as count FROM users GROUP BY status`).Scan(&userRows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load user stats"})
			return
		}
	}
	for _, row := range userRows {
		key := strings.ToLower(strings.TrimSpace(row.Status))
		if key == "" {
			continue
		}
		users[key] = row.Count
		users["total"] += row.Count
	}

	// apps count
	var appCount int64
	if orgID != "" {
		if err := h.DB.Raw(`SELECT COUNT(*) FROM apps WHERE org_id = ?`, orgID).Scan(&appCount).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load app stats"})
			return
		}
	} else {
		if err := h.DB.Raw(`SELECT COUNT(*) FROM apps`).Scan(&appCount).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load app stats"})
			return
		}
	}
	apps["total"] = appCount

	// devices count
	var deviceCount int64
	if orgID != "" {
		if err := h.DB.Raw(`
			SELECT COUNT(*) FROM devices d
			JOIN apps a ON a.id = d.app_id
			WHERE a.org_id = ?
		`, orgID).Scan(&deviceCount).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load device stats"})
			return
		}
	} else {
		if err := h.DB.Raw(`SELECT COUNT(*) FROM devices`).Scan(&deviceCount).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load device stats"})
			return
		}
	}
	devices["total"] = deviceCount

	// events count
	var eventCount int64
	if orgID != "" {
		if err := h.DB.Raw(`
			SELECT COUNT(*) FROM events e
			JOIN apps a ON a.id = e.app_id
			WHERE a.org_id = ?
		`, orgID).Scan(&eventCount).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load event stats"})
			return
		}
	} else {
		if err := h.DB.Raw(`SELECT COUNT(*) FROM events`).Scan(&eventCount).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load event stats"})
			return
		}
	}
	events["total"] = eventCount

	// daily events
	type dailyRow struct {
		Date       string
		EventCount int64
	}
	var daily []dailyRow
	if orgID != "" {
		if err := h.DB.Raw(`
			SELECT DATE(e.event_time) as date, COUNT(*) as event_count
			FROM events e
			JOIN apps a ON a.id = e.app_id
			WHERE a.org_id = ? AND e.event_time >= ? AND e.event_time <= ?
			GROUP BY DATE(e.event_time)
			ORDER BY date ASC
		`, orgID, from, to).Scan(&daily).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load event timeline"})
			return
		}
	} else {
		if err := h.DB.Raw(`
			SELECT DATE(event_time) as date, COUNT(*) as event_count
			FROM events
			WHERE event_time >= ? AND event_time <= ?
			GROUP BY DATE(event_time)
			ORDER BY date ASC
		`, from, to).Scan(&daily).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load event timeline"})
			return
		}
	}
	dailyItems := make([]gin.H, 0, len(daily))
	for _, row := range daily {
		dailyItems = append(dailyItems, gin.H{"date": row.Date, "event_count": row.EventCount})
	}

	resp := systemOverviewResponse{
		Orgs:    orgs,
		Users:   users,
		Apps:    apps,
		Devices: devices,
		Events:  events,
		Daily:   dailyItems,
	}
	c.JSON(http.StatusOK, resp)
}

