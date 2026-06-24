package release

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	appsvc "software-web-manager/backend/internal/services/app"
	channelsvc "software-web-manager/backend/internal/services/channel"
	"software-web-manager/backend/internal/services/clientupdate"
	orgsvc "software-web-manager/backend/internal/services/org"
	"time"

	"software-web-manager/backend/internal/api/common"
	"software-web-manager/backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
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

func (h *Handler) ListReleaseChannels(c *gin.Context) {
	appID := c.Param("id")
	orgID := c.GetString(middleware.ContextOrgID)
	if _, err := appsvc.NewService(h.DB).GetForOrg(orgID, appID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	items, err := channelsvc.NewService(h.DB).List(appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list release channels"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) CreateReleaseChannel(c *gin.Context) {
	appID := c.Param("id")
	orgID := c.GetString(middleware.ContextOrgID)
	userID := c.GetString(middleware.ContextUserID)
	if _, ok := common.EnsureAppWritable(h.DB, c, orgID, appID); !ok {
		return
	}
	if !common.HasPermission(c, "release.manage") && !orgsvc.NewService(h.DB).HasAppPermission(userID, appID, "release.manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}
	var req createReleaseChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := channelsvc.NewService(h.DB).Create(channelsvc.CreateInput{
		AppID:          appID,
		ReleaseID:      req.ReleaseID,
		ChannelCode:    req.ChannelCode,
		RolloutPercent: req.RolloutPercent,
		Paused:         req.Paused,
		Whitelist:      req.Whitelist,
		RolloutStartAt: req.RolloutStartAt,
		RolloutEndAt:   req.RolloutEndAt,
	})
	if err != nil {
		switch {
		case errors.Is(err, channelsvc.ErrInvalidRolloutPercent):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rollout_percent"})
		case errors.Is(err, channelsvc.ErrReleaseNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "release not found"})
		case errors.Is(err, channelsvc.ErrReleaseNotApproved):
			c.JSON(http.StatusBadRequest, gin.H{"error": "release not approved"})
		case errors.Is(err, channelsvc.ErrChannelCodeRequired):
			c.JSON(http.StatusBadRequest, gin.H{"error": "channel_code required"})
		case errors.Is(err, channelsvc.ErrChannelNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create release channel"})
		}
		return
	}

	relChannel := result.ReleaseChannel
	if clientupdate.NewService(h.DB, h.ClientUpdateHub).ShouldEmitImmediateReleaseChannel(relChannel) {
		published := result.PublishedAt
		if relChannel.PublishedAt != nil {
			published = *relChannel.PublishedAt
		}
		clientupdate.NewService(h.DB, h.ClientUpdateHub).EmitReleaseClientUpdate(
			"release_channel_updated",
			"manual_publish",
			result.Release.AppID,
			result.Release.ID,
			result.ChannelCode,
			published,
		)
	}

	common.Audit(h.DB, c, "release_channel.create", "release_channel", relChannel.ID, nil, relChannel)
	c.JSON(http.StatusOK, gin.H{"release_channel": relChannel})
}

func (h *Handler) UpdateReleaseChannel(c *gin.Context) {
	relChannelID := c.Param("id")
	userID := c.GetString(middleware.ContextUserID)

	svc := channelsvc.NewService(h.DB)
	relChannel, release, err := svc.LoadForUpdate(relChannelID)
	if err != nil {
		if errors.Is(err, channelsvc.ErrReleaseNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "release not found"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "release channel not found"})
		return
	}
	orgID := c.GetString(middleware.ContextOrgID)
	if _, ok := common.EnsureAppWritable(h.DB, c, orgID, release.AppID.String()); !ok {
		return
	}
	if !common.HasPermission(c, "release.manage") && !orgsvc.NewService(h.DB).HasAppPermission(userID, release.AppID.String(), "release.manage") {
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
	updated, channelCode, err := svc.ApplyUpdate(relChannelID, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update release channel"})
		return
	}
	published := time.Now()
	if updated.PublishedAt != nil {
		published = *updated.PublishedAt
	}
	clientupdate.NewService(h.DB, h.ClientUpdateHub).EmitReleaseClientUpdate(
		"release_channel_updated",
		"channel_update",
		release.AppID,
		updated.ReleaseID,
		channelCode,
		published,
	)
	common.Audit(h.DB, c, "release_channel.update", "release_channel", updated.ID, before, updated)
	c.JSON(http.StatusOK, gin.H{"release_channel": updated})
}

func (h *Handler) ReleaseChannelMetrics(c *gin.Context) {
	appID := c.Param("id")
	orgID := c.GetString(middleware.ContextOrgID)
	if _, err := appsvc.NewService(h.DB).GetForOrg(orgID, appID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	relChannelID := c.Param("rcId")

	from, to := common.ParseDateRange(c)
	result, err := channelsvc.NewService(h.DB).Metrics(appID, relChannelID, from, to)
	if err != nil {
		if errors.Is(err, channelsvc.ErrReleaseChannelNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "release channel not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load metrics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"release_id":         result.ReleaseID,
		"channel_code":       result.ChannelCode,
		"summary":            result.Summary,
		"timeline":           result.Timeline,
		"has_release_events": result.HasReleaseEvents,
		"from":               from.Format("2006-01-02"),
		"to":                 to.Format("2006-01-02"),
	})
}
