package analytics

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"software-web-manager/backend/internal/handlers"
	"software-web-manager/backend/internal/jobs"
	"software-web-manager/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

// Handler serves the analytics endpoints.
type Handler struct {
	*handlers.Handler
}

// New builds an analytics handler over the shared core.
func New(core *handlers.Handler) *Handler {
	return &Handler{Handler: core}
}

// RegisterRoutes wires the analytics routes onto the authenticated API group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/analytics/overview", h.AnalyticsOverview)
	rg.POST("/analytics/refresh", h.AnalyticsRefresh)
	rg.GET("/analytics/funnel", h.AnalyticsFunnel)
	rg.GET("/analytics/versions", h.AnalyticsVersions)
	rg.GET("/analytics/failures", h.AnalyticsFailures)
}

var analyticsRefreshLocks sync.Map

type analyticsRefreshRequest struct {
	AppID string `json:"app_id" binding:"required"`
	From  string `json:"from"`
	To    string `json:"to"`
}

func (h *Handler) AnalyticsRefresh(c *gin.Context) {
	var req analyticsRefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.AppID = strings.TrimSpace(req.AppID)
	if req.AppID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id required"})
		return
	}
	orgID := c.GetString(middleware.ContextOrgID)
	if _, err := h.GetAppForOrg(orgID, req.AppID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	if _, loaded := analyticsRefreshLocks.LoadOrStore(req.AppID, struct{}{}); loaded {
		c.JSON(http.StatusConflict, gin.H{"error": "analytics_refresh_in_progress"})
		return
	}
	defer analyticsRefreshLocks.Delete(req.AppID)

	from, to := handlers.ParseDateRangeWithValues(req.From, req.To)
	from = dayStart(from)
	to = dayStart(to)
	endExclusive := to.AddDate(0, 0, 1)
	rowsAffected, err := jobs.AggregateAppRange(h.DB, req.AppID, from, endExclusive)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to refresh analytics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":            true,
		"rows_affected": rowsAffected,
		"from":          from.Format("2006-01-02"),
		"to":            to.Format("2006-01-02"),
		"refreshed_at":  time.Now().Format(time.RFC3339),
	})
}

func (h *Handler) AnalyticsOverview(c *gin.Context) {
	if !h.RequirePermission(c, handlers.PermissionRoleViewer) {
		return
	}
	appID := c.Query("app_id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id required"})
		return
	}
	orgID := c.GetString(middleware.ContextOrgID)
	if _, err := h.GetAppForOrg(orgID, appID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	from, to := handlers.ParseDateRange(c)
	rows := []struct {
		EventName string
		Count     int64
	}{}
	if err := h.DB.Raw(`
		SELECT event_name, SUM(count) as count
		FROM daily_metrics
		WHERE app_id = ? AND date >= ? AND date <= ?
		GROUP BY event_name
	`, appID, from, to).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load metrics"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows})
}

func (h *Handler) AnalyticsFunnel(c *gin.Context) {
	if !h.RequirePermission(c, handlers.PermissionRoleViewer) {
		return
	}
	appID := c.Query("app_id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id required"})
		return
	}
	orgID := c.GetString(middleware.ContextOrgID)
	if _, err := h.GetAppForOrg(orgID, appID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	from, to := handlers.ParseDateRange(c)
	rows := []struct {
		EventName string
		Count     int64
	}{}
	funnelEvents := []string{"check_update", "update_available", "download_started", "download_completed", "install_completed", "app_started"}
	if err := h.DB.Raw(`
		SELECT event_name, SUM(count) as count
		FROM daily_metrics
		WHERE app_id = ? AND date >= ? AND date <= ? AND event_name IN ?
		GROUP BY event_name
	`, appID, from, to, funnelEvents).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load funnel"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows})
}

func (h *Handler) AnalyticsVersions(c *gin.Context) {
	if !h.RequirePermission(c, handlers.PermissionRoleViewer) {
		return
	}
	appID := c.Query("app_id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id required"})
		return
	}
	orgID := c.GetString(middleware.ContextOrgID)
	if _, err := h.GetAppForOrg(orgID, appID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	from, to := handlers.ParseDateRange(c)
	rows := []struct {
		Version string
		Count   int64
	}{}
	if err := h.DB.Raw(`
		SELECT dim_value as version, SUM(count) as count
		FROM daily_event_dimensions
		WHERE app_id = ? AND event_name = 'app_started' AND dim_key = 'version' AND date >= ? AND date <= ?
		GROUP BY dim_value
		ORDER BY count DESC
		LIMIT 20
	`, appID, from, to).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load versions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows})
}

func (h *Handler) AnalyticsFailures(c *gin.Context) {
	if !h.RequirePermission(c, handlers.PermissionRoleViewer) {
		return
	}
	appID := c.Query("app_id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id required"})
		return
	}
	orgID := c.GetString(middleware.ContextOrgID)
	if _, err := h.GetAppForOrg(orgID, appID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	from, to := handlers.ParseDateRange(c)
	rows := []struct {
		Reason string
		Count  int64
	}{}
	if err := h.DB.Raw(`
		SELECT dim_value as reason, SUM(count) as count
		FROM daily_event_dimensions
		WHERE app_id = ? AND event_name = 'update_failed' AND dim_key = 'reason' AND date >= ? AND date <= ?
		GROUP BY dim_value
		ORDER BY count DESC
		LIMIT 20
	`, appID, from, to).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load failures"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows})
}

func dayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
