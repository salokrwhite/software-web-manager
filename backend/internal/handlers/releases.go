package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type createReleaseRequest struct {
	Version             string  `json:"version" binding:"required"`
	VersionCode         *int    `json:"version_code"`
	Notes               string  `json:"notes"`
	NotesURL            *string `json:"notes_url"`
	ExternalDownloadURL string  `json:"external_download_url"`
	ReleaseTemplateID   string  `json:"release_template_id"`
}

type updateReleaseRequest struct {
	Version             *string `json:"version"`
	VersionCode         *int    `json:"version_code"`
	ExternalDownloadURL *string `json:"external_download_url"`
}

type setReleaseTemplateRequest struct {
	TemplateID *string `json:"template_id"`
}

type publishReleaseRequest struct {
	ChannelCode    string                 `json:"channel_code" binding:"required"`
	RolloutPercent int                    `json:"rollout_percent"`
	Mandatory      bool                   `json:"mandatory"`
	Whitelist      []string               `json:"whitelist"`
	TargetingRules map[string]interface{} `json:"targeting_rules"`
	RegionRules    map[string]interface{} `json:"region_rules"`
	RolloutStartAt *time.Time             `json:"rollout_start_at"`
	RolloutEndAt   *time.Time             `json:"rollout_end_at"`
	Paused         bool                   `json:"paused"`
}

func cloneTimePtr(v *time.Time) *time.Time {
	if v == nil {
		return nil
	}
	out := *v
	return &out
}

type reviewReleaseRequest struct {
	Note string `json:"note"`
}

type rollbackReleaseRequest struct {
	ChannelCode string `json:"channel_code" binding:"required"`
	ReleaseID   string `json:"release_id"`
}

type releaseListItem struct {
	models.Release
	ArtifactCount       int64  `json:"artifact_count" gorm:"column:artifact_count"`
	ExternalDownloadURL string `json:"external_download_url" gorm:"column:external_download_url"`
}

func (h *Handler) CreateRelease(c *gin.Context) {
	appID := c.Param("id")
	userID := c.GetString(middleware.ContextUserID)
	if !h.hasPermission(c, "release.manage") && !h.hasAppPermission(userID, appID, "release.manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}
	orgID := c.GetString(middleware.ContextOrgID)
	if _, ok := h.ensureAppWritable(c, orgID, appID); !ok {
		return
	}
	appUUID, err := uuid.Parse(appID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid app id"})
		return
	}
	var req createReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.NotesURL != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "notes_url_not_supported"})
		return
	}
	personal, err := h.isPersonalOrg(orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load org"})
		return
	}
	status := "draft"
	var approvedAt *time.Time
	var approvedBy *uuid.UUID
	if !personal {
		status = "approved"
		now := time.Now()
		approvedAt = &now
		if uid, parseErr := uuid.Parse(userID); parseErr == nil {
			approvedBy = &uid
		}
	}
	var templateID *uuid.UUID
	if strings.TrimSpace(req.ReleaseTemplateID) != "" {
		parsed, err := uuid.Parse(req.ReleaseTemplateID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid release_template_id"})
			return
		}
		var tpl models.ReleaseTemplate
		if err := h.DB.Where("id = ? AND org_id = ?", parsed, orgID).First(&tpl).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "release template not found"})
			return
		}
		templateID = &parsed
	}
	release := models.Release{
		AppID:               appUUID,
		Version:             req.Version,
		VersionCode:         req.VersionCode,
		Notes:               req.Notes,
		ExternalDownloadURL: strings.TrimSpace(req.ExternalDownloadURL),
		ReleaseTemplateID:   templateID,
		Status:              status,
		ApprovedAt:          approvedAt,
		ApprovedBy:          approvedBy,
	}
	db := h.DB
	if !h.hasReleaseExternalDownloadURLColumn() {
		release.ExternalDownloadURL = ""
		db = db.Omit("external_download_url")
	}
	if err := db.Create(&release).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create release"})
		return
	}
	h.audit(c, "release.create", "release", release.ID, nil, release)
	c.JSON(http.StatusOK, gin.H{"release": release})
}

func (h *Handler) ListReleases(c *gin.Context) {
	if !h.requirePermission(c, PermissionRoleViewer) {
		return
	}
	appID := c.Param("id")
	orgID := c.GetString(middleware.ContextOrgID)
	if _, err := h.getAppForOrg(orgID, appID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	var releases []releaseListItem
	selectExternal := "r.external_download_url as external_download_url"
	if !h.hasReleaseExternalDownloadURLColumn() {
		selectExternal = "'' as external_download_url"
	}
	if err := h.DB.Raw(`
		SELECT r.*,
		       `+selectExternal+`,
		       (SELECT COUNT(*) FROM artifacts a WHERE a.release_id = r.id) AS artifact_count
		FROM releases r
		WHERE r.app_id = ?
		ORDER BY r.created_at DESC
	`, appID).Scan(&releases).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list releases"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": releases})
}

func (h *Handler) UpdateRelease(c *gin.Context) {
	releaseID := c.Param("id")
	userID := c.GetString(middleware.ContextUserID)
	orgID := c.GetString(middleware.ContextOrgID)
	release, err := h.getReleaseForOrg(orgID, releaseID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "release not found"})
		return
	}
	if !h.hasPermission(c, "release.manage") && !h.hasAppPermission(userID, release.AppID.String(), "release.manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}
	var req updateReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{}
	if req.Version != nil {
		value := strings.TrimSpace(*req.Version)
		if value != "" {
			updates["version"] = value
		}
	}
	if req.VersionCode != nil {
		updates["version_code"] = req.VersionCode
	}
	if req.ExternalDownloadURL != nil {
		if h.hasReleaseExternalDownloadURLColumn() {
			updates["external_download_url"] = strings.TrimSpace(*req.ExternalDownloadURL)
		}
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no updates"})
		return
	}
	before := release
	if err := h.DB.Model(&models.Release{}).Where("id = ?", releaseID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update release"})
		return
	}
	if err := h.DB.Where("id = ?", releaseID).First(&release).Error; err == nil {
		h.audit(c, "release.update", "release", release.ID, before, release)
	}
	c.JSON(http.StatusOK, gin.H{"release": release})
}

func (h *Handler) SetReleaseTemplate(c *gin.Context) {
	releaseID := c.Param("id")
	userID := c.GetString(middleware.ContextUserID)
	orgID := c.GetString(middleware.ContextOrgID)
	release, err := h.getReleaseForOrg(orgID, releaseID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "release not found"})
		return
	}
	if !h.hasPermission(c, "release.manage") && !h.hasAppPermission(userID, release.AppID.String(), "release.manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}
	var req setReleaseTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var templateID *uuid.UUID
	if req.TemplateID != nil && strings.TrimSpace(*req.TemplateID) != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(*req.TemplateID))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template_id"})
			return
		}
		var tpl models.ReleaseTemplate
		if err := h.DB.Where("id = ? AND org_id = ?", parsed, orgID).First(&tpl).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "release template not found"})
			return
		}
		templateID = &parsed
	}
	before := release
	if err := h.DB.Model(&models.Release{}).Where("id = ?", releaseID).Update("release_template_id", templateID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update release template"})
		return
	}
	_ = h.DB.Where("id = ?", releaseID).First(&release).Error
	h.audit(c, "release.template.update", "release", release.ID, before, release)
	c.JSON(http.StatusOK, gin.H{"release": release})
}

func (h *Handler) PublishRelease(c *gin.Context) {
	releaseID := c.Param("id")
	userID := c.GetString(middleware.ContextUserID)
	var req publishReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ChannelCode = strings.ToLower(req.ChannelCode)
	if req.RolloutPercent <= 0 || req.RolloutPercent > 100 {
		req.RolloutPercent = 100
	}

	orgID := c.GetString(middleware.ContextOrgID)
	var release models.Release
	rel, err := h.getReleaseForOrg(orgID, releaseID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "release not found"})
		return
	}
	release = rel
	if _, ok := h.ensureAppWritable(c, orgID, release.AppID.String()); !ok {
		return
	}
	if !h.hasPermission(c, "release.manage") && !h.hasAppPermission(userID, release.AppID.String(), "release.manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}
	if release.Status != "approved" && release.Status != "published" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "release not approved"})
		return
	}

	var channel models.Channel
	if err := h.DB.Where("app_id = ? AND code = ?", release.AppID, req.ChannelCode).First(&channel).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	// Template-driven publish policy:
	// - emergency=true: immediate publish, ignore template schedule/window.
	// - emergency=false + schedule_at in future: create scheduled channel and defer activation.
	now := time.Now()
	rolloutStartAt := cloneTimePtr(req.RolloutStartAt)
	rolloutEndAt := cloneTimePtr(req.RolloutEndAt)
	channelStatus := "active"
	publishedAt := now
	shouldUpdateReleasePublished := true
	publishReason := "manual_publish"
	var template models.ReleaseTemplate
	templateLoaded := false
	if release.ReleaseTemplateID != nil && *release.ReleaseTemplateID != uuid.Nil {
		if err := h.DB.Where("id = ? AND org_id = ?", *release.ReleaseTemplateID, orgID).First(&template).Error; err == nil {
			templateLoaded = true
		}
	}
	if templateLoaded {
		if template.Emergency {
			rolloutStartAt = nil
			rolloutEndAt = nil
			publishReason = "template_emergency_publish"
		} else {
			if template.WindowStart != nil {
				rolloutStartAt = cloneTimePtr(template.WindowStart)
			}
			if template.WindowEnd != nil {
				rolloutEndAt = cloneTimePtr(template.WindowEnd)
			}
			if template.ScheduleAt != nil && template.ScheduleAt.After(now) {
				channelStatus = "scheduled"
				publishedAt = *template.ScheduleAt
				shouldUpdateReleasePublished = false
				publishReason = "template_scheduled"
			} else {
				publishReason = "template_manual_publish"
			}
		}
	}

	whitelistBytes, _ := json.Marshal(req.Whitelist)
	targetingBytes, _ := json.Marshal(req.TargetingRules)
	regionBytes := normalizeRegionRulesValue(req.RegionRules)
	relChannel := models.ReleaseChannel{
		ReleaseID:          release.ID,
		ChannelID:          channel.ID,
		RolloutPercent:     req.RolloutPercent,
		Mandatory:          req.Mandatory,
		WhitelistJSON:      whitelistBytes,
		RegionRulesJSON:    regionBytes,
		TargetingRulesJSON: targetingBytes,
		RolloutStartAt:     rolloutStartAt,
		RolloutEndAt:       rolloutEndAt,
		Paused:             req.Paused,
		Status:             channelStatus,
		PublishedAt:        &publishedAt,
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		// (release_id, channel_id) 有唯一约束：已存在则更新，避免重复发布/双击撞约束报错。
		var existing models.ReleaseChannel
		err := tx.Where("release_id = ? AND channel_id = ?", release.ID, channel.ID).First(&existing).Error
		if err == nil {
			updates := map[string]interface{}{
				"rollout_percent":      req.RolloutPercent,
				"mandatory":            req.Mandatory,
				"whitelist_json":       whitelistBytes,
				"region_rules_json":    regionBytes,
				"targeting_rules_json": targetingBytes,
				"rollout_start_at":     rolloutStartAt,
				"rollout_end_at":       rolloutEndAt,
				"paused":               req.Paused,
				"status":               channelStatus,
				"published_at":         publishedAt,
			}
			if err := tx.Model(&models.ReleaseChannel{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
				return err
			}
			if err := tx.Where("id = ?", existing.ID).First(&relChannel).Error; err != nil {
				return err
			}
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := tx.Create(&relChannel).Error; err != nil {
				return err
			}
		} else {
			return err
		}
		if channelStatus == "active" {
			if err := tx.Model(&models.ReleaseChannel{}).Where("channel_id = ? AND id <> ?", channel.ID, relChannel.ID).
				Update("status", "inactive").Error; err != nil {
				return err
			}
		}
		if shouldUpdateReleasePublished {
			return tx.Model(&release).Updates(map[string]interface{}{
				"status":       "published",
				"published_at": publishedAt,
			}).Error
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to publish release"})
		return
	}
	if channelStatus == "active" && h.shouldEmitImmediateReleaseChannel(relChannel) {
		h.emitReleaseClientUpdate(
			"release_published",
			publishReason,
			release.AppID,
			release.ID,
			req.ChannelCode,
			publishedAt,
		)
	}

	h.audit(c, "release.publish", "release_channel", relChannel.ID, nil, relChannel)
	c.JSON(http.StatusOK, gin.H{"release_channel": relChannel})
}

func (h *Handler) RevokeRelease(c *gin.Context) {
	releaseID := c.Param("id")
	if !h.requirePermission(c, "release.manage") {
		return
	}
	orgID := c.GetString(middleware.ContextOrgID)
	release, err := h.getReleaseForOrg(orgID, releaseID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "release not found"})
		return
	}
	if _, ok := h.ensureAppWritable(c, orgID, release.AppID.String()); !ok {
		return
	}
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Release{}).Where("id = ?", releaseID).Update("status", "revoked").Error; err != nil {
			return err
		}
		// 同步停用该版本下的所有通道：避免撤销后通道仍显示为运行中，
		// 也防止预约中的通道被后台 watcher 误激活而「复活」已撤销的版本。
		return tx.Model(&models.ReleaseChannel{}).
			Where("release_id = ? AND status IN ?", release.ID, []string{"active", "scheduled"}).
			Update("status", "inactive").Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke release"})
		return
	}
	if rel, err := h.getReleaseForOrg(orgID, releaseID); err == nil {
		h.audit(c, "release.revoke", "release", rel.ID, rel, nil)
	}
	c.JSON(http.StatusOK, gin.H{"status": "revoked"})
}

func (h *Handler) DeleteRelease(c *gin.Context) {
	releaseID := c.Param("id")
	userID := c.GetString(middleware.ContextUserID)
	orgID := c.GetString(middleware.ContextOrgID)
	release, err := h.getReleaseForOrg(orgID, releaseID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "release not found"})
		return
	}
	if _, ok := h.ensureAppWritable(c, orgID, release.AppID.String()); !ok {
		return
	}
	if !h.hasPermission(c, "release.manage") && !h.hasAppPermission(userID, release.AppID.String(), "release.manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("release_id = ?", release.ID).Delete(&models.Artifact{}).Error; err != nil {
			return err
		}
		if err := tx.Where("release_id = ?", release.ID).Delete(&models.ReleaseChannel{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", release.ID).Delete(&models.Release{}).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete release"})
		return
	}
	h.audit(c, "release.delete", "release", release.ID, release, nil)
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func (h *Handler) SubmitRelease(c *gin.Context) {
	releaseID := c.Param("id")
	var req reviewReleaseRequest
	_ = c.ShouldBindJSON(&req)
	orgID := c.GetString(middleware.ContextOrgID)
	release, err := h.getReleaseForOrg(orgID, releaseID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "release not found"})
		return
	}
	if _, ok := h.ensureAppWritable(c, orgID, release.AppID.String()); !ok {
		return
	}
	userID := c.GetString(middleware.ContextUserID)
	if !h.hasPermission(c, "release.manage") && !h.hasAppPermission(userID, release.AppID.String(), "release.manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}
	if release.Status != "draft" && release.Status != "rejected" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "release not in draft"})
		return
	}
	personal, err := h.isPersonalOrg(orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load org"})
		return
	}
	if personal {
		var count int64
		if err := h.DB.Model(&models.Artifact{}).Where("release_id = ?", release.ID).Count(&count).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check artifacts"})
			return
		}
		extURL := strings.TrimSpace(release.ExternalDownloadURL)
		if !h.hasReleaseExternalDownloadURLColumn() {
			extURL = ""
		}
		if count == 0 && extURL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "artifact_required"})
			return
		}
	}
	now := time.Now()
	if personal {
		if err := h.DB.Model(&models.Release{}).Where("id = ?", releaseID).
			Updates(map[string]interface{}{"status": "in_review", "submitted_at": now}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit release"})
			return
		}
		h.audit(c, "release.submit", "release", release.ID, release, gin.H{"note": req.Note})
		c.JSON(http.StatusOK, gin.H{"status": "in_review"})
		return
	}

	updates := map[string]interface{}{
		"status":       "approved",
		"submitted_at": now,
		"approved_at":  now,
	}
	if uid, parseErr := uuid.Parse(userID); parseErr == nil {
		updates["approved_by"] = uid
	}
	if err := h.DB.Model(&models.Release{}).Where("id = ?", releaseID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to auto approve release"})
		return
	}
	h.audit(c, "release.submit", "release", release.ID, release, gin.H{"note": req.Note, "auto_approved": true})
	c.JSON(http.StatusOK, gin.H{"status": "approved", "auto_approved": true})
}

func (h *Handler) ApproveRelease(c *gin.Context) {
	releaseID := c.Param("id")
	var req reviewReleaseRequest
	_ = c.ShouldBindJSON(&req)
	orgID := c.GetString(middleware.ContextOrgID)
	release, err := h.getReleaseForOrg(orgID, releaseID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "release not found"})
		return
	}
	if _, ok := h.ensureAppWritable(c, orgID, release.AppID.String()); !ok {
		return
	}
	if personal, err := h.isPersonalOrg(orgID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load org"})
		return
	} else if personal {
		systemRole := strings.ToLower(strings.TrimSpace(c.GetString(middleware.ContextSystemRole)))
		if systemRole != "system_admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "system_admin_required"})
			return
		}
	}
	userID := c.GetString(middleware.ContextUserID)
	if !h.hasPermission(c, "release.manage") && !h.hasAppPermission(userID, release.AppID.String(), "release.manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}
	if release.Status != "in_review" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "release not in review"})
		return
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	now := time.Now()
	if err := h.DB.Model(&models.Release{}).Where("id = ?", releaseID).
		Updates(map[string]interface{}{"status": "approved", "approved_at": now, "approved_by": uid}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to approve release"})
		return
	}
	h.audit(c, "release.approve", "release", release.ID, release, gin.H{"note": req.Note})
	c.JSON(http.StatusOK, gin.H{"status": "approved"})
}

func (h *Handler) RejectRelease(c *gin.Context) {
	releaseID := c.Param("id")
	var req reviewReleaseRequest
	_ = c.ShouldBindJSON(&req)
	orgID := c.GetString(middleware.ContextOrgID)
	release, err := h.getReleaseForOrg(orgID, releaseID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "release not found"})
		return
	}
	if _, ok := h.ensureAppWritable(c, orgID, release.AppID.String()); !ok {
		return
	}
	if personal, err := h.isPersonalOrg(orgID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load org"})
		return
	} else if personal {
		systemRole := strings.ToLower(strings.TrimSpace(c.GetString(middleware.ContextSystemRole)))
		if systemRole != "system_admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "system_admin_required"})
			return
		}
	}
	userID := c.GetString(middleware.ContextUserID)
	if !h.hasPermission(c, "release.manage") && !h.hasAppPermission(userID, release.AppID.String(), "release.manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}
	if release.Status != "in_review" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "release not in review"})
		return
	}
	if err := h.DB.Model(&models.Release{}).Where("id = ?", releaseID).Update("status", "rejected").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reject release"})
		return
	}
	h.audit(c, "release.reject", "release", release.ID, release, gin.H{"note": req.Note})
	c.JSON(http.StatusOK, gin.H{"status": "rejected"})
}

func (h *Handler) RollbackRelease(c *gin.Context) {
	releaseID := c.Param("id")
	var req rollbackReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	orgID := c.GetString(middleware.ContextOrgID)
	current, err := h.getReleaseForOrg(orgID, releaseID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "release not found"})
		return
	}
	if _, ok := h.ensureAppWritable(c, orgID, current.AppID.String()); !ok {
		return
	}
	userID := c.GetString(middleware.ContextUserID)
	if !h.hasPermission(c, "release.manage") && !h.hasAppPermission(userID, current.AppID.String(), "release.manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}
	var channel models.Channel
	if err := h.DB.Where("app_id = ? AND code = ?", current.AppID, req.ChannelCode).First(&channel).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}
	targetReleaseID := req.ReleaseID
	if targetReleaseID == "" {
		var target models.Release
		if err := h.DB.Raw(`
			SELECT r.* FROM releases r
			JOIN release_channels rc ON rc.release_id = r.id
			WHERE rc.channel_id = ? AND r.status = 'published' AND r.id <> ?
			ORDER BY rc.published_at DESC
			LIMIT 1
		`, channel.ID, releaseID).Scan(&target).Error; err != nil || target.ID == (uuid.UUID{}) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no rollback target"})
			return
		}
		targetReleaseID = target.ID.String()
	}
	var targetRelease models.Release
	if err := h.DB.Where("id = ? AND app_id = ?", targetReleaseID, current.AppID).First(&targetRelease).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "target release not found"})
		return
	}
	// 显式指定的回滚目标也必须是已发布版本，否则会「回滚成功」但实际不下发（update_check 过滤 published）。
	if strings.ToLower(strings.TrimSpace(targetRelease.Status)) != "published" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rollback target not published"})
		return
	}
	var relChannel models.ReleaseChannel
	if err := h.DB.Where("release_id = ? AND channel_id = ?", targetRelease.ID, channel.ID).First(&relChannel).Error; err != nil {
		publishedAt := time.Now()
		relChannel = models.ReleaseChannel{
			ReleaseID:      targetRelease.ID,
			ChannelID:      channel.ID,
			RolloutPercent: 100,
			Mandatory:      false,
			Status:         "active",
			PublishedAt:    &publishedAt,
		}
	}
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if relChannel.ID == (uuid.UUID{}) {
			if err := tx.Create(&relChannel).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Model(&relChannel).Updates(map[string]interface{}{"status": "active", "paused": false}).Error; err != nil {
				return err
			}
		}
		return tx.Model(&models.ReleaseChannel{}).Where("channel_id = ? AND id <> ?", channel.ID, relChannel.ID).
			Update("status", "inactive").Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to rollback release"})
		return
	}
	if h.shouldEmitImmediateReleaseChannel(relChannel) {
		published := time.Now()
		if relChannel.PublishedAt != nil {
			published = *relChannel.PublishedAt
		}
		h.emitReleaseClientUpdate(
			"release_rolled_back",
			"rollback",
			current.AppID,
			relChannel.ReleaseID,
			req.ChannelCode,
			published,
		)
	}
	h.audit(c, "release.rollback", "release_channel", relChannel.ID, nil, relChannel)
	c.JSON(http.StatusOK, gin.H{"release_channel": relChannel})
}
