package handlers

import (
	"net/http"
	"strings"
	"time"

	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const deviceBlockedCode = "device_blocked"
const defaultBlockReason = "manual_remove"

type blockDeviceRequest struct {
	AppID  string `json:"app_id" binding:"required"`
	Reason string `json:"reason"`
}

type blockDeviceByIDRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
	Reason   string `json:"reason"`
}

type unblockDeviceRequest struct {
	AppID string `json:"app_id" binding:"required"`
}

type deviceControlResponse struct {
	ID          string  `json:"id"`
	AppID       string  `json:"app_id"`
	DeviceID    string  `json:"device_id"`
	Blocked     bool    `json:"blocked"`
	Reason      *string `json:"reason,omitempty"`
	BlockedAt   *string `json:"blocked_at,omitempty"`
	BlockedBy   *string `json:"blocked_by,omitempty"`
	UnblockedAt *string `json:"unblocked_at,omitempty"`
	UnblockedBy *string `json:"unblocked_by,omitempty"`
}

func (h *Handler) writeDeviceBlocked(c *gin.Context, reason *string) {
	message := "device is blocked"
	if reason != nil && strings.TrimSpace(*reason) != "" {
		message = "device is blocked: " + strings.TrimSpace(*reason)
	}
	c.JSON(http.StatusForbidden, gin.H{
		"error": gin.H{
			"code":    deviceBlockedCode,
			"message": message,
		},
	})
}

func (h *Handler) IsDeviceBlocked(appID uuid.UUID, deviceID string) (bool, *models.DeviceControl, error) {
	if h == nil || h.DB == nil || !h.hasDeviceControlsTable() {
		return false, nil, nil
	}
	deviceID = strings.TrimSpace(deviceID)
	if appID == uuid.Nil || deviceID == "" {
		return false, nil, nil
	}
	var control models.DeviceControl
	if err := h.DB.Where("app_id = ? AND device_id = ? AND blocked = 1", appID, deviceID).First(&control).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, &control, nil
}

func (h *Handler) checkDeviceBlocked(c *gin.Context, appID uuid.UUID, deviceID string) bool {
	blocked, control, err := h.IsDeviceBlocked(appID, deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check device status"})
		return true
	}
	if blocked {
		h.writeDeviceBlocked(c, control.Reason)
		return true
	}
	return false
}

func (h *Handler) setDeviceBlocked(appID uuid.UUID, deviceID, reason, actorID string) (models.DeviceControl, error) {
	now := time.Now()
	deviceID = strings.TrimSpace(deviceID)
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = defaultBlockReason
	}

	var reasonPtr *string
	if reason != "" {
		reasonCopy := reason
		reasonPtr = &reasonCopy
	}

	var actorUUID *uuid.UUID
	if parsed, err := uuid.Parse(strings.TrimSpace(actorID)); err == nil {
		actorUUID = &parsed
	}

	record := models.DeviceControl{
		AppID:       appID,
		DeviceID:    deviceID,
		Blocked:     true,
		Reason:      reasonPtr,
		BlockedAt:   &now,
		BlockedBy:   actorUUID,
		UnblockedAt: nil,
		UnblockedBy: nil,
	}
	err := h.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "app_id"},
			{Name: "device_id"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"blocked":      true,
			"reason":       reasonPtr,
			"blocked_at":   now,
			"blocked_by":   actorUUID,
			"unblocked_at": nil,
			"unblocked_by": nil,
			"updated_at":   now,
		}),
	}).Create(&record).Error
	if err != nil {
		return models.DeviceControl{}, err
	}
	var out models.DeviceControl
	if err := h.DB.Where("app_id = ? AND device_id = ?", appID, deviceID).First(&out).Error; err != nil {
		return models.DeviceControl{}, err
	}
	return out, nil
}

func (h *Handler) setDeviceUnblocked(appID uuid.UUID, deviceID, actorID string) (models.DeviceControl, error) {
	now := time.Now()
	deviceID = strings.TrimSpace(deviceID)

	var actorUUID *uuid.UUID
	if parsed, err := uuid.Parse(strings.TrimSpace(actorID)); err == nil {
		actorUUID = &parsed
	}

	record := models.DeviceControl{
		AppID:       appID,
		DeviceID:    deviceID,
		Blocked:     false,
		BlockedAt:   nil,
		BlockedBy:   nil,
		UnblockedAt: &now,
		UnblockedBy: actorUUID,
	}
	err := h.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "app_id"},
			{Name: "device_id"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"blocked":      false,
			"unblocked_at": now,
			"unblocked_by": actorUUID,
			"updated_at":   now,
		}),
	}).Create(&record).Error
	if err != nil {
		return models.DeviceControl{}, err
	}
	var out models.DeviceControl
	if err := h.DB.Where("app_id = ? AND device_id = ?", appID, deviceID).First(&out).Error; err != nil {
		return models.DeviceControl{}, err
	}
	return out, nil
}

func (h *Handler) checkAppManagePermission(c *gin.Context, appID string) bool {
	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	if h.hasPermission(c, "app.manage") || h.hasAppPermission(userID, appID, "app.manage") {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
	return false
}

func formatUUIDPtr(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	raw := id.String()
	return &raw
}

func formatTimePtr(v *time.Time) *string {
	if v == nil {
		return nil
	}
	raw := v.Format(time.RFC3339)
	return &raw
}

func toDeviceControlResponse(control models.DeviceControl) deviceControlResponse {
	return deviceControlResponse{
		ID:          control.ID.String(),
		AppID:       control.AppID.String(),
		DeviceID:    control.DeviceID,
		Blocked:     control.Blocked,
		Reason:      control.Reason,
		BlockedAt:   formatTimePtr(control.BlockedAt),
		BlockedBy:   formatUUIDPtr(control.BlockedBy),
		UnblockedAt: formatTimePtr(control.UnblockedAt),
		UnblockedBy: formatUUIDPtr(control.UnblockedBy),
	}
}

func (h *Handler) BlockDevice(c *gin.Context) {
	if !h.hasDeviceControlsTable() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先执行数据库迁移: 0028_device_controls"})
		return
	}
	orgID := strings.TrimSpace(c.GetString(middleware.ContextOrgID))
	if orgID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var req blockDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.AppID = strings.TrimSpace(req.AppID)
	deviceRecordID := strings.TrimSpace(c.Param("id"))
	if req.AppID == "" || deviceRecordID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id and id required"})
		return
	}
	if !h.checkAppManagePermission(c, req.AppID) {
		return
	}

	var app models.App
	if err := h.DB.Where("id = ? AND org_id = ?", req.AppID, orgID).First(&app).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}

	var device models.Device
	if err := h.DB.Where("id = ? AND app_id = ?", deviceRecordID, app.ID).First(&device).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}

	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = defaultBlockReason
	}
	control, err := h.setDeviceBlocked(app.ID, device.DeviceID, reason, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to block device"})
		return
	}

	h.emitDeviceShutdown(app.ID, device.DeviceID, reason)
	c.JSON(http.StatusOK, gin.H{"item": toDeviceControlResponse(control)})
}

func (h *Handler) BlockDeviceByDeviceID(c *gin.Context) {
	if !h.hasDeviceControlsTable() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先执行数据库迁移: 0028_device_controls"})
		return
	}
	orgID := strings.TrimSpace(c.GetString(middleware.ContextOrgID))
	appID := strings.TrimSpace(c.Param("id"))
	if orgID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id required"})
		return
	}
	if !h.checkAppManagePermission(c, appID) {
		return
	}

	var app models.App
	if err := h.DB.Where("id = ? AND org_id = ?", appID, orgID).First(&app).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}

	var req blockDeviceByIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.DeviceID = strings.TrimSpace(req.DeviceID)
	if req.DeviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id required"})
		return
	}

	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = defaultBlockReason
	}
	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	control, err := h.setDeviceBlocked(app.ID, req.DeviceID, reason, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to block device"})
		return
	}

	h.emitDeviceShutdown(app.ID, req.DeviceID, reason)
	c.JSON(http.StatusOK, gin.H{"item": toDeviceControlResponse(control)})
}

func (h *Handler) UnblockDevice(c *gin.Context) {
	if !h.hasDeviceControlsTable() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先执行数据库迁移: 0028_device_controls"})
		return
	}
	orgID := strings.TrimSpace(c.GetString(middleware.ContextOrgID))
	if orgID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var req unblockDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.AppID = strings.TrimSpace(req.AppID)
	controlID := strings.TrimSpace(c.Param("id"))
	if req.AppID == "" || controlID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id and id required"})
		return
	}
	if !h.checkAppManagePermission(c, req.AppID) {
		return
	}

	var app models.App
	if err := h.DB.Where("id = ? AND org_id = ?", req.AppID, orgID).First(&app).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}

	deviceID := ""
	var control models.DeviceControl
	if err := h.DB.Where("id = ? AND app_id = ?", controlID, app.ID).First(&control).Error; err == nil {
		deviceID = strings.TrimSpace(control.DeviceID)
	}
	if deviceID == "" {
		var device models.Device
		if err := h.DB.Where("id = ? AND app_id = ?", controlID, app.ID).First(&device).Error; err == nil {
			deviceID = strings.TrimSpace(device.DeviceID)
		}
	}
	if deviceID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "blocked device not found"})
		return
	}

	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	updated, err := h.setDeviceUnblocked(app.ID, deviceID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unblock device"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"item": toDeviceControlResponse(updated)})
}

func (h *Handler) ListBlockedDevices(c *gin.Context) {
	if !h.hasDeviceControlsTable() {
		c.JSON(http.StatusOK, gin.H{"items": []deviceControlResponse{}, "total": 0, "page": 1, "page_size": 20})
		return
	}
	orgID := strings.TrimSpace(c.GetString(middleware.ContextOrgID))
	appID := strings.TrimSpace(c.Param("id"))
	if orgID == "" || appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id required"})
		return
	}
	if !h.checkAppManagePermission(c, appID) {
		return
	}
	if _, err := h.getAppForOrg(orgID, appID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}

	page := 1
	pageSize := 20
	if v := strings.TrimSpace(c.Query("page")); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := strings.TrimSpace(c.Query("page_size")); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 {
			if n > 200 {
				n = 200
			}
			pageSize = n
		}
	}
	offset := (page - 1) * pageSize
	keyword := strings.TrimSpace(c.Query("q"))

	db := h.DB.Model(&models.DeviceControl{}).Where("app_id = ? AND blocked = 1", appID)
	if keyword != "" {
		db = db.Where("device_id LIKE ?", "%"+keyword+"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count blocked devices"})
		return
	}

	var rows []models.DeviceControl
	if err := db.Order("blocked_at desc").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list blocked devices"})
		return
	}

	actorEmailByID := map[uuid.UUID]string{}
	actorIDs := make([]uuid.UUID, 0, len(rows))
	actorIDSet := map[uuid.UUID]struct{}{}
	for _, row := range rows {
		if row.BlockedBy == nil {
			continue
		}
		if _, exists := actorIDSet[*row.BlockedBy]; exists {
			continue
		}
		actorIDSet[*row.BlockedBy] = struct{}{}
		actorIDs = append(actorIDs, *row.BlockedBy)
	}
	if len(actorIDs) > 0 {
		var users []models.User
		if err := h.DB.Model(&models.User{}).Select("id", "email").Where("id IN ?", actorIDs).Find(&users).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve blocked-by user emails"})
			return
		}
		for _, user := range users {
			actorEmailByID[user.ID] = user.Email
		}
	}

	items := make([]deviceControlResponse, 0, len(rows))
	for _, row := range rows {
		item := toDeviceControlResponse(row)
		if row.BlockedBy != nil {
			if email, ok := actorEmailByID[*row.BlockedBy]; ok && strings.TrimSpace(email) != "" {
				emailCopy := email
				item.BlockedBy = &emailCopy
			}
		}
		items = append(items, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
