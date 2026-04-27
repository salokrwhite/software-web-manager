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
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type feedbackListItem struct {
	ID              string     `json:"id"`
	Content         string     `json:"content"`
	Rating          *int       `json:"rating"`
	Contact         string     `json:"contact"`
	DeviceID        string     `json:"device_id"`
	AppVersion      string     `json:"app_version"`
	ChannelCode     string     `json:"channel_code"`
	Status          string     `json:"status"`
	InternalNote    string     `json:"internal_note"`
	HandledBy       *string    `json:"handled_by"`
	HandledAt       *time.Time `json:"handled_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	AttachmentCount int64      `json:"attachment_count"`
}

type updateFeedbackRequest struct {
	Status       *string `json:"status"`
	InternalNote *string `json:"internal_note"`
}

var validFeedbackStatuses = map[string]bool{
	"open":       true,
	"processing": true,
	"resolved":   true,
	"closed":     true,
}

func normalizeFeedbackStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return "open"
	}
	return status
}

func (h *Handler) ClientSubmitFeedback(c *gin.Context) {
	app, org, ok := clientAppOrgFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if !h.hasFeedbackTable() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "feedback_not_ready"})
		return
	}
	if h.hasAppFeedbackEnabledColumn() && !app.FeedbackEnabled {
		c.JSON(http.StatusForbidden, gin.H{
			"error": gin.H{
				"code":    "feedback_disabled",
				"message": "feedback disabled",
			},
		})
		return
	}
	if !h.hasFeedbackWorkflowColumns() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "feedback_not_ready"})
		return
	}
	if err := h.ensureStorage(c); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage not configured"})
		return
	}
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
		return
	}

	deviceID := strings.TrimSpace(c.PostForm("device_id"))
	content := strings.TrimSpace(c.PostForm("content"))
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id required"})
		return
	}
	if content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content required"})
		return
	}

	channelCode := strings.TrimSpace(c.PostForm("channel_code"))
	if channelCode == "" {
		var channel models.Channel
		if err := h.DB.Where("app_id = ? AND is_default = true", app.ID).First(&channel).Error; err == nil {
			channelCode = channel.Code
		}
	}
	appVersion := strings.TrimSpace(c.PostForm("app_version"))
	contact := strings.TrimSpace(c.PostForm("contact"))
	ratingValue := strings.TrimSpace(c.PostForm("rating"))
	var rating *int
	if ratingValue != "" {
		if v, err := parseInt(ratingValue); err == nil {
			if v < 1 || v > 5 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "rating must be between 1 and 5"})
				return
			}
			rating = &v
		}
	}

	var metadata datatypes.JSON
	if raw := strings.TrimSpace(c.PostForm("metadata")); raw != "" {
		var payload any
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid metadata"})
			return
		}
		metadata = datatypes.JSON([]byte(raw))
	}

	feedback := models.Feedback{
		ID:          uuid.New(),
		OrgID:       org.ID,
		AppID:       app.ID,
		DeviceID:    deviceID,
		ChannelCode: channelCode,
		AppVersion:  appVersion,
		Rating:      rating,
		Content:     content,
		Contact:     contact,
		Metadata:    metadata,
		Status:      "open",
	}

	attachments, statusCode, err := h.storeFeedbackAttachments(c, feedback.ID, app.OrgID)
	if err != nil {
		if statusCode == 0 {
			statusCode = http.StatusInternalServerError
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&feedback).Error; err != nil {
			return err
		}
		if len(attachments) > 0 {
			return tx.Create(&attachments).Error
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create feedback"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "id": feedback.ID})
}

func (h *Handler) ListAppFeedback(c *gin.Context) {
	orgID := strings.TrimSpace(c.GetString(middleware.ContextOrgID))
	if orgID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	appID := strings.TrimSpace(c.Param("id"))
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id required"})
		return
	}
	if _, err := h.getAppForOrg(orgID, appID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	if !h.hasFeedbackTable() || !h.hasFeedbackWorkflowColumns() {
		c.JSON(http.StatusOK, gin.H{"items": []feedbackListItem{}, "total": 0, "ready": false, "status_counts": gin.H{}})
		return
	}

	page := 1
	pageSize := 20
	if v := c.Query("page"); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := c.Query("page_size"); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 {
			if n > 200 {
				n = 200
			}
			pageSize = n
		}
	}
	offset := (page - 1) * pageSize

	keyword := strings.TrimSpace(c.Query("keyword"))
	ratingValue := strings.TrimSpace(c.Query("rating"))
	var rating *int
	if ratingValue != "" {
		if n, err := parseInt(ratingValue); err == nil {
			rating = &n
		}
	}
	status := normalizeFeedbackStatus(c.Query("status"))
	if c.Query("status") != "" && !validFeedbackStatuses[status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feedback status"})
		return
	}
	hasAttachment := strings.ToLower(strings.TrimSpace(c.Query("has_attachment")))

	var fromTime *time.Time
	if v := strings.TrimSpace(c.Query("from")); v != "" {
		if t, err := parseTimeFlexible(v); err == nil {
			fromTime = &t
		}
	}
	var toTime *time.Time
	if v := strings.TrimSpace(c.Query("to")); v != "" {
		if t, err := parseTimeFlexible(v); err == nil {
			end := t
			if len(v) == len("2006-01-02") {
				end = end.Add(24*time.Hour - time.Nanosecond)
			}
			toTime = &end
		}
	}

	applyFeedbackFilters := func(db *gorm.DB, includeStatus bool) *gorm.DB {
		db = db.Where("org_id = ? AND app_id = ?", orgID, appID)
		if keyword != "" {
			db = db.Where("(content LIKE ? OR contact LIKE ?)", "%"+keyword+"%", "%"+keyword+"%")
		}
		if rating != nil {
			db = db.Where("rating = ?", *rating)
		}
		if fromTime != nil {
			db = db.Where("created_at >= ?", *fromTime)
		}
		if toTime != nil {
			db = db.Where("created_at <= ?", *toTime)
		}
		switch hasAttachment {
		case "true", "1", "yes":
			db = db.Where("EXISTS (SELECT 1 FROM attachments fa WHERE fa.owner_type = 'feedback' AND fa.owner_id = feedbacks.id)")
		case "false", "0", "no":
			db = db.Where("NOT EXISTS (SELECT 1 FROM attachments fa WHERE fa.owner_type = 'feedback' AND fa.owner_id = feedbacks.id)")
		}
		if includeStatus && c.Query("status") != "" {
			db = db.Where("status = ?", status)
		}
		return db
	}

	countDB := applyFeedbackFilters(h.DB.Model(&models.Feedback{}), true)
	var total int64
	if err := countDB.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count feedback"})
		return
	}

	type statusCount struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	var statusRows []statusCount
	statusDB := applyFeedbackFilters(h.DB.Model(&models.Feedback{}), false)
	if err := statusDB.Select("status, COUNT(*) as count").Group("status").Scan(&statusRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count feedback statuses"})
		return
	}
	statusCounts := gin.H{"open": 0, "processing": 0, "resolved": 0, "closed": 0}
	for _, row := range statusRows {
		statusCounts[normalizeFeedbackStatus(row.Status)] = row.Count
	}

	db := h.DB.Table("feedbacks f").
		Select(`f.id, f.content, f.rating, f.contact, f.device_id, f.app_version, f.channel_code,
		        f.status, f.internal_note, CAST(f.handled_by AS CHAR) as handled_by, f.handled_at, f.created_at, f.updated_at,
		        (SELECT COUNT(*) FROM attachments fa WHERE fa.owner_type = 'feedback' AND fa.owner_id = f.id) as attachment_count`).
		Where("f.org_id = ? AND f.app_id = ?", orgID, appID)
	if keyword != "" {
		db = db.Where("(f.content LIKE ? OR f.contact LIKE ?)", "%"+keyword+"%", "%"+keyword+"%")
	}
	if rating != nil {
		db = db.Where("f.rating = ?", *rating)
	}
	if fromTime != nil {
		db = db.Where("f.created_at >= ?", *fromTime)
	}
	if toTime != nil {
		db = db.Where("f.created_at <= ?", *toTime)
	}
	if c.Query("status") != "" {
		db = db.Where("f.status = ?", status)
	}
	switch hasAttachment {
	case "true", "1", "yes":
		db = db.Where("EXISTS (SELECT 1 FROM attachments fa WHERE fa.owner_type = 'feedback' AND fa.owner_id = f.id)")
	case "false", "0", "no":
		db = db.Where("NOT EXISTS (SELECT 1 FROM attachments fa WHERE fa.owner_type = 'feedback' AND fa.owner_id = f.id)")
	}

	var items []feedbackListItem
	if err := db.Order("f.updated_at desc, f.created_at desc").Limit(pageSize).Offset(offset).Scan(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list feedback"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total, "ready": true, "status_counts": statusCounts})
}

func (h *Handler) GetAppFeedbackDetail(c *gin.Context) {
	orgID := strings.TrimSpace(c.GetString(middleware.ContextOrgID))
	if orgID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	appID := strings.TrimSpace(c.Param("id"))
	feedbackID := strings.TrimSpace(c.Param("fid"))
	if appID == "" || feedbackID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id and feedback_id required"})
		return
	}
	if _, err := h.getAppForOrg(orgID, appID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	if !h.hasFeedbackTable() || !h.hasFeedbackWorkflowColumns() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "feedback_not_ready"})
		return
	}

	var feedback models.Feedback
	if err := h.DB.Where("id = ? AND org_id = ? AND app_id = ?", feedbackID, orgID, appID).First(&feedback).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "feedback not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load feedback"})
		return
	}

	attachments, err := h.loadFeedbackAttachments(c, feedbackID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load attachments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"feedback":    feedback,
		"attachments": attachments,
	})
}

func (h *Handler) UpdateAppFeedback(c *gin.Context) {
	orgID := strings.TrimSpace(c.GetString(middleware.ContextOrgID))
	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	if orgID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	appID := strings.TrimSpace(c.Param("id"))
	feedbackID := strings.TrimSpace(c.Param("fid"))
	if appID == "" || feedbackID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id and feedback_id required"})
		return
	}
	if _, err := h.getAppForOrg(orgID, appID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	if !h.hasFeedbackTable() || !h.hasFeedbackWorkflowColumns() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "feedback_not_ready"})
		return
	}

	var req updateFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	updates := map[string]interface{}{}
	statusChanged := false
	if req.Status != nil {
		status := normalizeFeedbackStatus(*req.Status)
		if !validFeedbackStatuses[status] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feedback status"})
			return
		}
		updates["status"] = status
		statusChanged = true
	}
	if req.InternalNote != nil {
		note := strings.TrimSpace(*req.InternalNote)
		if len(note) > 5000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "internal_note too long"})
			return
		}
		updates["internal_note"] = note
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no updates"})
		return
	}
	now := time.Now()
	if statusChanged {
		updates["handled_at"] = &now
		if parsedUserID, err := uuid.Parse(userID); err == nil {
			updates["handled_by"] = &parsedUserID
		}
	}
	updates["updated_at"] = now

	result := h.DB.Model(&models.Feedback{}).
		Where("id = ? AND org_id = ? AND app_id = ?", feedbackID, orgID, appID).
		Updates(updates)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update feedback"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "feedback not found"})
		return
	}

	var feedback models.Feedback
	if err := h.DB.Where("id = ? AND org_id = ? AND app_id = ?", feedbackID, orgID, appID).First(&feedback).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "feedback not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load feedback"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"feedback": feedback})
}
