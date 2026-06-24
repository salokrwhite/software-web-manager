package app

import (
	"net/http"
	"software-web-manager/backend/internal/db/schema"
	"strings"
	"time"

	"software-web-manager/backend/internal/core"
	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type createAppRequest struct {
	Name        string `json:"name" binding:"required"`
	Slug        string `json:"slug" binding:"required"`
	Description string `json:"description"`
}

type createChannelRequest struct {
	Name                string `json:"name" binding:"required"`
	Code                string `json:"code" binding:"required"`
	IsDefault           bool   `json:"is_default"`
	MinSupportedVersion string `json:"min_supported_version"`
}

type updateAppRequest struct {
	Name                     *string `json:"name"`
	Slug                     *string `json:"slug"`
	Description              *string `json:"description"`
	PublicKey                *string `json:"public_key"`
	FeedbackEnabled          *bool   `json:"feedback_enabled"`
	HeartbeatIntervalSeconds *int    `json:"heartbeat_interval_seconds"`
	OnlineEnabled            *bool   `json:"online_enabled"`
	MaintenanceEnabled       *bool   `json:"maintenance_enabled"`
	MaintenanceStartAt       *string `json:"maintenance_start_at"`
	MaintenanceMessage       *string `json:"maintenance_message"`
}

type addAppMemberRequest struct {
	UserEmail string `json:"user_email" binding:"required,email"`
	Role      string `json:"role" binding:"required"`
}

func (h *Handler) ListApps(c *gin.Context) {
	if !h.RequirePermission(c, core.PermissionRoleViewer) {
		return
	}
	orgID := c.GetString(middleware.ContextOrgID)
	var apps []models.App
	if err := h.DB.Where("org_id = ?", orgID).Order("created_at desc").Find(&apps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list apps"})
		return
	}
	if !schema.HasAppFeedbackEnabledColumn(h.DB) {
		for i := range apps {
			apps[i].FeedbackEnabled = false
		}
	}
	if !schema.HasAppHeartbeatIntervalColumn(h.DB) {
		for i := range apps {
			apps[i].HeartbeatIntervalSeconds = 60
		}
	}
	if !schema.HasAppOnlineEnabledColumn(h.DB) {
		for i := range apps {
			apps[i].OnlineEnabled = false
		}
	}
	if !schema.HasAppMaintenanceColumn(h.DB) {
		for i := range apps {
			apps[i].MaintenanceEnabled = false
			apps[i].MaintenanceStartAt = nil
			apps[i].MaintenanceMessage = ""
		}
	}
	orgType := ""
	limit := 0
	if strings.TrimSpace(orgID) != "" {
		var org models.Org
		if err := h.DB.Where("id = ?", orgID).First(&org).Error; err == nil {
			orgType = strings.ToLower(strings.TrimSpace(org.OrgType))
			if orgType == "personal" && h.Cfg.PersonalAppLimit > 0 {
				limit = h.Cfg.PersonalAppLimit
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"items":    apps,
		"count":    len(apps),
		"limit":    limit,
		"org_type": orgType,
	})
}

func (h *Handler) CreateApp(c *gin.Context) {
	if !h.RequirePermission(c, "app.manage") {
		return
	}
	var req createAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	orgID := c.GetString(middleware.ContextOrgID)
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid org"})
		return
	}
	var org models.Org
	if err := h.DB.Where("id = ?", orgUUID).First(&org).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid org"})
		return
	}
	if strings.ToLower(strings.TrimSpace(org.OrgType)) == "personal" {
		limit := h.Cfg.PersonalAppLimit
		if limit > 0 {
			var count int64
			if err := h.DB.Model(&models.App{}).Where("org_id = ?", orgUUID).Count(&count).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check app limit"})
				return
			}
			if int(count) >= limit {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "personal_app_limit_reached",
					"limit": limit,
					"count": count,
				})
				return
			}
		}
	}
	app := models.App{OrgID: orgUUID, Name: req.Name, Slug: strings.ToLower(req.Slug), Description: req.Description}
	// app_secret is generated in app credential management flow, not app creation.
	app.AppSecretCiphertext = ""
	app.AppSecretUpdatedAt = nil
	app.AppSecretScopesJSON = core.AppSecretScopesJSON(nil)
	app.AppSecretExpiresAt = nil
	app.AppSecretName = "app_secret"
	if strings.ToLower(strings.TrimSpace(org.OrgType)) == "personal" {
		now := time.Now()
		app.Status = "pending"
		app.SubmittedAt = &now
	}
	if schema.HasAppFeedbackEnabledColumn(h.DB) {
		app.FeedbackEnabled = false
	}
	if schema.HasAppHeartbeatIntervalColumn(h.DB) {
		app.HeartbeatIntervalSeconds = 60
	}
	if schema.HasAppOnlineEnabledColumn(h.DB) {
		app.OnlineEnabled = false
	}
	omitFields := make([]string, 0, 6)
	if !schema.HasAppFeedbackEnabledColumn(h.DB) {
		omitFields = append(omitFields, "feedback_enabled")
	}
	if !schema.HasAppHeartbeatIntervalColumn(h.DB) {
		omitFields = append(omitFields, "heartbeat_interval_seconds")
	}
	if !schema.HasAppOnlineEnabledColumn(h.DB) {
		omitFields = append(omitFields, "online_enabled")
	}
	if !schema.HasAppPublicKeyColumn(h.DB) {
		omitFields = append(omitFields, "public_key")
	}
	if !schema.HasAppSecretScopesColumn(h.DB) {
		omitFields = append(omitFields, "app_secret_scopes")
	}
	if !schema.HasAppSecretExpiresAtColumn(h.DB) {
		omitFields = append(omitFields, "app_secret_expires_at")
	}
	if !schema.HasAppSecretNameColumn(h.DB) {
		omitFields = append(omitFields, "app_secret_name")
	}
	db := h.DB.Select("*")
	if len(omitFields) > 0 {
		db = db.Omit(omitFields...)
	}
	if err := db.Create(&app).Error; err != nil {
		if IsAppSecretColumnMissingErr(err) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "missing app_secret_ciphertext column, run migration 0029_app_secret_and_signature"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create app"})
		return
	}
	updates := map[string]interface{}{}
	if schema.HasAppFeedbackEnabledColumn(h.DB) {
		updates["feedback_enabled"] = false
	}
	if schema.HasAppHeartbeatIntervalColumn(h.DB) {
		updates["heartbeat_interval_seconds"] = 60
	}
	if schema.HasAppOnlineEnabledColumn(h.DB) {
		updates["online_enabled"] = false
	}
	if len(updates) > 0 {
		if err := h.DB.Model(&models.App{}).Where("id = ?", app.ID).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initialize app settings"})
			return
		}
	}
	defaultChannel := models.Channel{
		AppID:     app.ID,
		Name:      "Stable",
		Code:      "stable",
		IsDefault: true,
	}
	_ = h.DB.Create(&defaultChannel).Error
	h.Audit(c, "app.create", "app", app.ID, nil, app)
	resp := gin.H{"app": app, "app_id": app.ID}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetApp(c *gin.Context) {
	orgID := c.GetString(middleware.ContextOrgID)
	appID := c.Param("id")
	var app models.App
	if err := h.DB.Where("id = ? AND org_id = ?", appID, orgID).First(&app).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	if !schema.HasAppFeedbackEnabledColumn(h.DB) {
		app.FeedbackEnabled = false
	}
	if !schema.HasAppHeartbeatIntervalColumn(h.DB) {
		app.HeartbeatIntervalSeconds = 60
	}
	if !schema.HasAppOnlineEnabledColumn(h.DB) {
		app.OnlineEnabled = false
	}
	if !schema.HasAppMaintenanceColumn(h.DB) {
		app.MaintenanceEnabled = false
		app.MaintenanceStartAt = nil
		app.MaintenanceMessage = ""
	}
	c.JSON(http.StatusOK, gin.H{"app": app})
}

func (h *Handler) UpdateApp(c *gin.Context) {
	appID := c.Param("id")
	userID := c.GetString(middleware.ContextUserID)
	if !h.HasPermission(c, "app.manage") && !h.HasAppPermission(userID, appID, "app.manage") {
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
	var req updateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{}
	if personal {
		status := strings.ToLower(strings.TrimSpace(app.Status))
		if status == "pending" {
			c.JSON(http.StatusForbidden, gin.H{"error": "app_pending_review", "status": status})
			return
		}
	}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Slug != nil {
		updates["slug"] = strings.ToLower(strings.TrimSpace(*req.Slug))
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.PublicKey != nil {
		if !schema.HasAppPublicKeyColumn(h.DB) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请先执行数据库迁移: 0034_app_public_key"})
			return
		}
		updates["public_key"] = strings.TrimSpace(*req.PublicKey)
	}
	if req.FeedbackEnabled != nil {
		if !schema.HasAppFeedbackEnabledColumn(h.DB) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请先执行数据库迁移: 0019_app_feedback_enabled"})
			return
		}
		updates["feedback_enabled"] = *req.FeedbackEnabled
	}
	if req.HeartbeatIntervalSeconds != nil {
		if !schema.HasAppHeartbeatIntervalColumn(h.DB) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请先执行数据库迁移: 0020_app_heartbeat_interval"})
			return
		}
		value := *req.HeartbeatIntervalSeconds
		if value < 10 || value > 3600 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "heartbeat_interval_seconds must be between 10 and 3600"})
			return
		}
		updates["heartbeat_interval_seconds"] = value
	}
	if req.OnlineEnabled != nil {
		if !schema.HasAppOnlineEnabledColumn(h.DB) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请先执行数据库迁移: 0021_app_online_enabled"})
			return
		}
		updates["online_enabled"] = *req.OnlineEnabled
	}
	maintenanceTouched := false
	if req.MaintenanceEnabled != nil || req.MaintenanceStartAt != nil || req.MaintenanceMessage != nil {
		if !schema.HasAppMaintenanceColumn(h.DB) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请先执行数据库迁移: 0005_app_maintenance"})
			return
		}
		maintenanceTouched = true
		effectiveStart := app.MaintenanceStartAt
		if req.MaintenanceStartAt != nil {
			raw := strings.TrimSpace(*req.MaintenanceStartAt)
			if raw == "" {
				updates["maintenance_start_at"] = nil
				effectiveStart = nil
			} else {
				parsed, perr := time.Parse(time.RFC3339, raw)
				if perr != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid maintenance_start_at"})
					return
				}
				parsedUTC := parsed.UTC()
				updates["maintenance_start_at"] = parsedUTC
				effectiveStart = &parsedUTC
			}
		}
		if req.MaintenanceMessage != nil {
			msg := strings.TrimSpace(*req.MaintenanceMessage)
			if len([]rune(msg)) > 500 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "maintenance_message too long"})
				return
			}
			updates["maintenance_message"] = msg
		}
		effectiveEnabled := app.MaintenanceEnabled
		if req.MaintenanceEnabled != nil {
			effectiveEnabled = *req.MaintenanceEnabled
			updates["maintenance_enabled"] = effectiveEnabled
		}
		if effectiveEnabled && effectiveStart == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "maintenance_start_at required"})
			return
		}
	}
	if personal && len(updates) > 0 {
		allowDirect := true
		for key := range updates {
			if key != "feedback_enabled" && key != "heartbeat_interval_seconds" && key != "online_enabled" &&
				key != "maintenance_enabled" && key != "maintenance_start_at" && key != "maintenance_message" {
				allowDirect = false
				break
			}
		}
		if !allowDirect {
			now := time.Now()
			updates["status"] = "pending"
			updates["submitted_at"] = now
			updates["approved_by"] = nil
			updates["approved_at"] = nil
			updates["rejected_by"] = nil
			updates["rejected_at"] = nil
			updates["rejection_reason"] = nil
		}
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no updates"})
		return
	}
	before := app
	if err := h.DB.Model(&models.App{}).Where("id = ?", appID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update app"})
		return
	}
	if err := h.DB.Where("id = ?", appID).First(&app).Error; err == nil {
		h.Audit(c, "app.update", "app", app.ID, before, app)
		if maintenanceTouched {
			if app.MaintenanceEnabled {
				h.EmitMaintenance(app, core.MaintenanceEventScheduled)
			} else {
				h.EmitMaintenance(app, core.MaintenanceEventCancelled)
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"app": app})
}

func (h *Handler) DeleteApp(c *gin.Context) {
	if !h.RequirePermission(c, "app.manage") {
		return
	}
	appID := c.Param("id")
	orgID := c.GetString(middleware.ContextOrgID)

	// 验证应用存在且属于当前组织，同时检查可操作状态
	app, ok := h.EnsureAppWritable(c, orgID, appID)
	if !ok {
		return
	}

	// 开启事务
	tx := h.DB.Begin()

	// 1. 删除关联 artifacts（通过 releases）
	var releaseIDs []string
	if err := tx.Model(&models.Release{}).Where("app_id = ?", appID).Pluck("id", &releaseIDs).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query releases"})
		return
	}
	if len(releaseIDs) > 0 {
		if err := tx.Where("release_id IN ?", releaseIDs).Delete(&models.Artifact{}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete artifacts"})
			return
		}
	}

	// 2. 删除 release_channels
	if len(releaseIDs) > 0 {
		if err := tx.Where("release_id IN ?", releaseIDs).Delete(&models.ReleaseChannel{}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete release channels"})
			return
		}
	}

	// 3. 删除 releases
	if err := tx.Where("app_id = ?", appID).Delete(&models.Release{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete releases"})
		return
	}

	// 4. 删除 channels
	if err := tx.Where("app_id = ?", appID).Delete(&models.Channel{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete channels"})
		return
	}

	// 5. 删除 app 成员
	if err := tx.Where("scope_id = ?", appID).Delete(&models.AppMember{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete app members"})
		return
	}

	// 6. 删除反馈与反馈附件
	if tx.Migrator().HasTable(&models.Feedback{}) {
		var feedbackIDs []string
		if err := tx.Model(&models.Feedback{}).Where("app_id = ?", appID).Pluck("id", &feedbackIDs).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query feedbacks"})
			return
		}
		if err := core.DeleteAttachmentsByOwners(tx, core.AttachmentOwnerFeedback, feedbackIDs); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete feedback attachments"})
			return
		}
		if err := tx.Where("app_id = ?", appID).Delete(&models.Feedback{}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete feedbacks"})
			return
		}
	}

	// 7. 删除 app_secrets
	if tx.Migrator().HasTable(&models.AppSecret{}) {
		if err := tx.Where("app_id = ?", appID).Delete(&models.AppSecret{}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete app secrets"})
			return
		}
	}

	// 8. 最后删除应用
	if err := tx.Where("id = ?", appID).Delete(&models.App{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete app"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
		return
	}
	h.Audit(c, "app.delete", "app", app.ID, app, nil)
	c.JSON(http.StatusOK, gin.H{"message": "app deleted"})
}

func (h *Handler) CreateChannel(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	if !h.HasPermission(c, "app.manage") && !h.HasAppPermission(userID, c.Param("id"), "app.manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}
	var req createChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	appID := c.Param("id")
	orgID := c.GetString(middleware.ContextOrgID)
	if _, ok := h.EnsureAppWritable(c, orgID, appID); !ok {
		return
	}
	appUUID, err := uuid.Parse(appID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid app id"})
		return
	}
	channel := models.Channel{
		AppID:               appUUID,
		Name:                req.Name,
		Code:                strings.ToLower(req.Code),
		IsDefault:           req.IsDefault,
		MinSupportedVersion: req.MinSupportedVersion,
	}
	if err := h.DB.Create(&channel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create channel"})
		return
	}
	h.Audit(c, "channel.create", "channel", channel.ID, nil, channel)
	c.JSON(http.StatusOK, gin.H{"channel": channel})
}

func (h *Handler) ListChannels(c *gin.Context) {
	appID := c.Param("id")
	orgID := c.GetString(middleware.ContextOrgID)
	if _, err := h.GetAppForOrg(orgID, appID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	var channels []models.Channel
	if err := h.DB.Where("app_id = ?", appID).Find(&channels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list channels"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": channels})
}

func (h *Handler) DeleteChannel(c *gin.Context) {
	appID := c.Param("id")
	channelID := c.Param("channel_id")
	userID := c.GetString(middleware.ContextUserID)
	if !h.HasPermission(c, "app.manage") && !h.HasAppPermission(userID, appID, "app.manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}
	orgID := c.GetString(middleware.ContextOrgID)
	if _, ok := h.EnsureAppWritable(c, orgID, appID); !ok {
		return
	}
	if _, err := h.GetAppForOrg(orgID, appID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}

	var channel models.Channel
	if err := h.DB.Where("id = ? AND app_id = ?", channelID, appID).First(&channel).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}
	if channel.IsDefault {
		c.JSON(http.StatusBadRequest, gin.H{"error": "default_channel_cannot_delete"})
		return
	}

	var channelCount int64
	if err := h.DB.Model(&models.Channel{}).Where("app_id = ?", appID).Count(&channelCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check channels"})
		return
	}
	if channelCount <= 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel_last_one_cannot_delete"})
		return
	}

	var releaseChannelCount int64
	if err := h.DB.Model(&models.ReleaseChannel{}).Where("channel_id = ?", channel.ID).Count(&releaseChannelCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check release channels"})
		return
	}
	if releaseChannelCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel_in_use"})
		return
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		// Keep delete explicit so old daily metrics won't block channel deletion by FK.
		if err := tx.Where("app_id = ? AND channel_id = ?", appID, channel.ID).Delete(&models.DailyMetric{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", channel.ID).Delete(&models.Channel{}).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete channel"})
		return
	}

	h.Audit(c, "channel.delete", "channel", channel.ID, channel, nil)
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func (h *Handler) ListAppMembers(c *gin.Context) {
	appID := c.Param("id")
	orgID := c.GetString(middleware.ContextOrgID)
	if _, err := h.GetAppForOrg(orgID, appID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	var members []models.AppMember
	if err := h.DB.Where("scope_id = ?", appID).Find(&members).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list members"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": members})
}

func (h *Handler) AddAppMember(c *gin.Context) {
	appID := c.Param("id")
	userID := c.GetString(middleware.ContextUserID)
	if !h.HasPermission(c, "app.manage") && !h.HasAppPermission(userID, appID, "app.manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}
	orgID := c.GetString(middleware.ContextOrgID)
	if _, ok := h.EnsureAppWritable(c, orgID, appID); !ok {
		return
	}
	var req addAppMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var user models.User
	if err := h.DB.Where("email = ?", strings.ToLower(strings.TrimSpace(req.UserEmail))).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	appUUID, err := uuid.Parse(appID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid app id"})
		return
	}
	member := models.AppMember{AppID: appUUID, UserID: user.ID, Role: req.Role}
	if err := h.DB.Create(&member).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add member"})
		return
	}
	h.Audit(c, "app_member.add", "app_member", appUUID, nil, member)
	c.JSON(http.StatusOK, gin.H{"member": member})
}
