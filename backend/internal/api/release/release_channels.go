package release

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"software-web-manager/backend/internal/api/common"
	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type updateReleaseChannelRequest struct {
	RolloutPercent *int                   `json:"rollout_percent"`
	Paused         *bool                  `json:"paused"`
	Status         *string                `json:"status"`
	TargetingRules map[string]interface{} `json:"targeting_rules"`
	Whitelist      *[]string              `json:"whitelist"`
	RegionRules    *json.RawMessage       `json:"region_rules"`
	RolloutStartAt *time.Time             `json:"rollout_start_at"`
	RolloutEndAt   *time.Time             `json:"rollout_end_at"`
}

type createReleaseChannelRequest struct {
	ReleaseID      string     `json:"release_id" binding:"required"`
	ChannelCode    string     `json:"channel_code" binding:"required"`
	RolloutPercent int        `json:"rollout_percent"`
	Paused         bool       `json:"paused"`
	Whitelist      []string   `json:"whitelist"`
	RolloutStartAt *time.Time `json:"rollout_start_at"`
	RolloutEndAt   *time.Time `json:"rollout_end_at"`
}

type releaseChannelItem struct {
	ID             uuid.UUID      `json:"id"`
	ReleaseID      uuid.UUID      `json:"release_id"`
	ChannelID      uuid.UUID      `json:"channel_id"`
	ChannelCode    string         `json:"channel_code"`
	ChannelName    string         `json:"channel_name"`
	ReleaseVersion string         `json:"release_version"`
	ReleaseStatus  string         `json:"release_status"`
	RolloutPercent int            `json:"rollout_percent"`
	Mandatory      bool           `json:"mandatory"`
	Whitelist      datatypes.JSON `json:"whitelist" gorm:"column:whitelist"`
	Status         string         `json:"status"`
	Paused         bool           `json:"paused"`
	TargetingRules datatypes.JSON `json:"targeting_rules" gorm:"column:targeting_rules"`
	RegionRules    datatypes.JSON `json:"region_rules" gorm:"column:region_rules"`
	RolloutStartAt *time.Time     `json:"rollout_start_at"`
	RolloutEndAt   *time.Time     `json:"rollout_end_at"`
	PublishedAt    *time.Time     `json:"published_at"`
}

func (h *Handler) ListReleaseChannels(c *gin.Context) {
	appID := c.Param("id")
	orgID := c.GetString(middleware.ContextOrgID)
	if _, err := h.GetAppForOrg(orgID, appID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	var items []releaseChannelItem
	if err := h.DB.Raw(`
		SELECT rc.id, rc.release_id, rc.channel_id, c.code as channel_code, c.name as channel_name,
			r.version as release_version, r.status as release_status,
			rc.rollout_percent, rc.mandatory, rc.whitelist_json as whitelist, rc.status, rc.paused, rc.targeting_rules_json as targeting_rules, rc.region_rules_json as region_rules,
			rc.rollout_start_at, rc.rollout_end_at, rc.published_at
		FROM release_channels rc
		JOIN releases r ON r.id = rc.release_id
		JOIN channels c ON c.id = rc.channel_id
		WHERE c.app_id = ?
		ORDER BY rc.published_at DESC
	`, appID).Scan(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list release channels"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) CreateReleaseChannel(c *gin.Context) {
	appID := c.Param("id")
	orgID := c.GetString(middleware.ContextOrgID)
	userID := c.GetString(middleware.ContextUserID)
	if _, ok := h.EnsureAppWritable(c, orgID, appID); !ok {
		return
	}
	if !h.HasPermission(c, "release.manage") && !h.HasAppPermission(userID, appID, "release.manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}
	var req createReleaseChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rolloutPercent := req.RolloutPercent
	if rolloutPercent == 0 {
		rolloutPercent = 100
	}
	if rolloutPercent <= 0 || rolloutPercent > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rollout_percent"})
		return
	}

	var release models.Release
	if err := h.DB.Where("id = ?", req.ReleaseID).First(&release).Error; err != nil || release.AppID.String() != appID {
		c.JSON(http.StatusNotFound, gin.H{"error": "release not found"})
		return
	}
	releaseStatus := strings.ToLower(strings.TrimSpace(release.Status))
	if releaseStatus != "approved" && releaseStatus != "published" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "release not approved"})
		return
	}

	var channel models.Channel
	channelCode := strings.ToLower(strings.TrimSpace(req.ChannelCode))
	if channelCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel_code required"})
		return
	}
	if err := h.DB.Where("app_id = ? AND code = ?", appID, channelCode).First(&channel).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	whitelist := req.Whitelist
	if whitelist == nil {
		whitelist = []string{}
	}
	whitelistBytes, _ := json.Marshal(whitelist)
	publishedAt := time.Now()
	var relChannel models.ReleaseChannel
	keepScheduled := false

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Where("release_id = ? AND channel_id = ?", release.ID, channel.ID).First(&relChannel).Error
		if err == nil {
			// 若该通道由发布模板预约中(scheduled)，保留其预约状态/激活时间/时间窗，
			// 只应用灰度参数，避免创建灰度策略时静默绕过预约直接上线。
			keepScheduled = strings.EqualFold(strings.TrimSpace(relChannel.Status), "scheduled")
			updates := map[string]interface{}{
				"rollout_percent": rolloutPercent,
				"paused":          req.Paused,
				"whitelist_json":  datatypes.JSON(whitelistBytes),
			}
			if keepScheduled {
				updates["status"] = "scheduled"
			} else {
				updates["status"] = "active"
				updates["rollout_start_at"] = req.RolloutStartAt
				updates["rollout_end_at"] = req.RolloutEndAt
				if relChannel.PublishedAt == nil {
					updates["published_at"] = publishedAt
				}
			}
			if err := tx.Model(&models.ReleaseChannel{}).Where("id = ?", relChannel.ID).Updates(updates).Error; err != nil {
				return err
			}
			if err := tx.Where("id = ?", relChannel.ID).First(&relChannel).Error; err != nil {
				return err
			}
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			relChannel = models.ReleaseChannel{
				ID:             uuid.New(),
				ReleaseID:      release.ID,
				ChannelID:      channel.ID,
				RolloutPercent: rolloutPercent,
				Mandatory:      false,
				WhitelistJSON:  datatypes.JSON(whitelistBytes),
				Status:         "active",
				Paused:         req.Paused,
				RolloutStartAt: req.RolloutStartAt,
				RolloutEndAt:   req.RolloutEndAt,
				PublishedAt:    &publishedAt,
			}
			if err := tx.Create(&relChannel).Error; err != nil {
				return err
			}
		} else {
			return err
		}
		if !keepScheduled && releaseStatus == "approved" {
			if err := tx.Model(&models.Release{}).Where("id = ?", release.ID).Updates(map[string]interface{}{
				"status":       "published",
				"published_at": publishedAt,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create release channel"})
		return
	}
	if h.ShouldEmitImmediateReleaseChannel(relChannel) {
		published := publishedAt
		if relChannel.PublishedAt != nil {
			published = *relChannel.PublishedAt
		}
		h.EmitReleaseClientUpdate(
			"release_channel_updated",
			"manual_publish",
			release.AppID,
			release.ID,
			channelCode,
			published,
		)
	}

	h.Audit(c, "release_channel.create", "release_channel", relChannel.ID, nil, relChannel)
	c.JSON(http.StatusOK, gin.H{"release_channel": relChannel})
}

func (h *Handler) UpdateReleaseChannel(c *gin.Context) {
	relChannelID := c.Param("id")
	userID := c.GetString(middleware.ContextUserID)
	var relChannel models.ReleaseChannel
	if err := h.DB.Where("id = ?", relChannelID).First(&relChannel).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "release channel not found"})
		return
	}
	var release models.Release
	if err := h.DB.Where("id = ?", relChannel.ReleaseID).First(&release).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "release not found"})
		return
	}
	orgID := c.GetString(middleware.ContextOrgID)
	if _, ok := h.EnsureAppWritable(c, orgID, release.AppID.String()); !ok {
		return
	}
	if !h.HasPermission(c, "release.manage") && !h.HasAppPermission(userID, release.AppID.String(), "release.manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	// 解析两次：present 记录请求里实际出现了哪些字段，req 解析具体值。
	// 这样才能区分「字段未传(保持原值)」与「字段显式传 null(清空)」——例如清空灰度时间窗。
	var present map[string]json.RawMessage
	if err := json.Unmarshal(rawBody, &present); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req updateReleaseChannelRequest
	if err := json.Unmarshal(rawBody, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{}
	if req.RolloutPercent != nil {
		if *req.RolloutPercent <= 0 || *req.RolloutPercent > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rollout_percent"})
			return
		}
		updates["rollout_percent"] = *req.RolloutPercent
	}
	if req.Paused != nil {
		updates["paused"] = *req.Paused
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.TargetingRules != nil {
		if b, err := json.Marshal(req.TargetingRules); err == nil {
			updates["targeting_rules_json"] = datatypes.JSON(b)
		}
	}
	if req.Whitelist != nil {
		if b, err := json.Marshal(*req.Whitelist); err == nil {
			updates["whitelist_json"] = datatypes.JSON(b)
		}
	}
	// 用 present 判断而非 req.RegionRules：*json.RawMessage 遇到 JSON null 会被解析成 nil 指针，
	// 导致「继承(下发 null)」时清不掉通道规则。改用原始字节，null 经 common.NormalizeRegionRules 返回 nil 清空列。
	if rr, ok := present["region_rules"]; ok {
		updates["region_rules_json"] = common.NormalizeRegionRules(json.RawMessage(rr))
	}
	if _, ok := present["rollout_start_at"]; ok {
		updates["rollout_start_at"] = req.RolloutStartAt
	}
	if _, ok := present["rollout_end_at"]; ok {
		updates["rollout_end_at"] = req.RolloutEndAt
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no updates"})
		return
	}
	before := relChannel
	if err := h.DB.Model(&models.ReleaseChannel{}).Where("id = ?", relChannelID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update release channel"})
		return
	}
	_ = h.DB.Where("id = ?", relChannelID).First(&relChannel).Error
	published := time.Now()
	if relChannel.PublishedAt != nil {
		published = *relChannel.PublishedAt
	}
	var channel models.Channel
	channelCode := ""
	if err := h.DB.Where("id = ?", relChannel.ChannelID).First(&channel).Error; err == nil {
		channelCode = channel.Code
	}
	h.EmitReleaseClientUpdate(
		"release_channel_updated",
		"channel_update",
		release.AppID,
		relChannel.ReleaseID,
		channelCode,
		published,
	)
	h.Audit(c, "release_channel.update", "release_channel", relChannel.ID, before, relChannel)
	c.JSON(http.StatusOK, gin.H{"release_channel": relChannel})
}

func (h *Handler) ReleaseChannelMetrics(c *gin.Context) {
	appID := c.Param("id")
	orgID := c.GetString(middleware.ContextOrgID)
	if _, err := h.GetAppForOrg(orgID, appID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	relChannelID := c.Param("rcId")
	var row struct {
		ReleaseID   uuid.UUID
		ChannelCode string
	}
	if err := h.DB.Raw(`
		SELECT rc.release_id, c.code as channel_code
		FROM release_channels rc
		JOIN channels c ON c.id = rc.channel_id
		WHERE rc.id = ? AND c.app_id = ?
	`, relChannelID, appID).Scan(&row).Error; err != nil || row.ReleaseID == (uuid.UUID{}) {
		c.JSON(http.StatusNotFound, gin.H{"error": "release channel not found"})
		return
	}

	from, to := common.ParseDateRange(c)
	funnelEvents := []string{"check_update", "update_available", "download_started", "download_completed", "install_completed", "update_failed", "app_started"}
	summaryRows := []struct {
		EventName string
		Count     int64
	}{}
	releaseID := row.ReleaseID.String()
	if err := h.DB.Raw(`
		SELECT event_name, COUNT(1) as count
		FROM events
		WHERE app_id = ? AND channel_code = ? AND release_id = ? AND event_time >= ? AND event_time <= ? AND event_name IN ?
		GROUP BY event_name
	`, appID, row.ChannelCode, releaseID, from, to, funnelEvents).Scan(&summaryRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load metrics"})
		return
	}
	summary := map[string]int64{}
	var total int64
	for _, name := range funnelEvents {
		summary[name] = 0
	}
	for _, r := range summaryRows {
		summary[r.EventName] = r.Count
		total += r.Count
	}

	timelineRows := []struct {
		Date      time.Time
		EventName string
		Count     int64
	}{}
	if err := h.DB.Raw(`
		SELECT DATE(event_time) as date, event_name, COUNT(1) as count
		FROM events
		WHERE app_id = ? AND channel_code = ? AND release_id = ? AND event_time >= ? AND event_time <= ? AND event_name IN ?
		GROUP BY DATE(event_time), event_name
		ORDER BY date ASC
	`, appID, row.ChannelCode, releaseID, from, to, funnelEvents).Scan(&timelineRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load metrics timeline"})
		return
	}
	dateMap := map[string]map[string]int64{}
	for _, tr := range timelineRows {
		dateKey := tr.Date.Format("2006-01-02")
		if _, ok := dateMap[dateKey]; !ok {
			dateMap[dateKey] = map[string]int64{}
		}
		dateMap[dateKey][tr.EventName] = tr.Count
	}
	dates := make([]string, 0, len(dateMap))
	for d := range dateMap {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	timeline := make([]map[string]interface{}, 0, len(dates))
	for _, d := range dates {
		item := map[string]interface{}{"date": d}
		for _, name := range funnelEvents {
			item[name] = dateMap[d][name]
		}
		timeline = append(timeline, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"release_id":         releaseID,
		"channel_code":       row.ChannelCode,
		"summary":            summary,
		"timeline":           timeline,
		"has_release_events": total > 0,
		"from":               from.Format("2006-01-02"),
		"to":                 to.Format("2006-01-02"),
	})
}
