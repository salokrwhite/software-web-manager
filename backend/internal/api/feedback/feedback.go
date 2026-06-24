package feedback

import (
	"encoding/json"
	"errors"
	"net/http"
	"software-web-manager/backend/internal/db/schema"
	appsvc "software-web-manager/backend/internal/services/app"
	feedbacksvc "software-web-manager/backend/internal/services/feedback"
	orgsvc "software-web-manager/backend/internal/services/org"
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

func (h *Handler) requireAppFeedbackPermission(c *gin.Context, appID string) bool {
	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	if common.HasPermission(c, "app.manage") || common.HasPermission(c, "release.manage") {
		return true
	}
	if orgsvc.NewService(h.DB).HasAppPermission(userID, appID, "app.manage") || orgsvc.NewService(h.DB).HasAppPermission(userID, appID, "release.manage") {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
	return false
}

type updateFeedbackRequest struct {
	Status       *string `json:"status"`
	InternalNote *string `json:"internal_note"`
}

func (h *Handler) ClientSubmitFeedback(c *gin.Context) {
	app, org, ok := middleware.ClientAppOrgFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if !schema.HasFeedbackTable(h.DB) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "feedback_not_ready"})
		return
	}
	if schema.HasAppFeedbackEnabledColumn(h.DB) && !app.FeedbackEnabled {
		c.JSON(http.StatusForbidden, gin.H{
			"error": gin.H{
				"code":    "feedback_disabled",
				"message": "feedback disabled",
			},
		})
		return
	}
	if !schema.HasFeedbackWorkflowColumns(h.DB) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "feedback_not_ready"})
		return
	}
	if err := h.EnsureStorage(); err != nil {
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

	svc := feedbacksvc.NewService(h.DB)
	channelCode := strings.TrimSpace(c.PostForm("channel_code"))
	if channelCode == "" {
		channelCode = svc.DefaultChannelCode(app.ID)
	}
	appVersion := strings.TrimSpace(c.PostForm("app_version"))
	contact := strings.TrimSpace(c.PostForm("contact"))
	ratingValue := strings.TrimSpace(c.PostForm("rating"))
	var rating *int
	if ratingValue != "" {
		if v, err := common.ParseInt(ratingValue); err == nil {
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

	if err := svc.CreateWithAttachments(&feedback, attachments); err != nil {
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
	if _, err := appsvc.NewService(h.DB).GetForOrg(orgID, appID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	if !h.requireAppFeedbackPermission(c, appID) {
		return
	}
	if !schema.HasFeedbackTable(h.DB) || !schema.HasFeedbackWorkflowColumns(h.DB) {
		c.JSON(http.StatusOK, gin.H{"items": []feedbacksvc.FeedbackListItem{}, "total": 0, "ready": false, "status_counts": gin.H{}})
		return
	}

	page := 1
	pageSize := 20
	if v := c.Query("page"); v != "" {
		if n, err := common.ParseInt(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := c.Query("page_size"); v != "" {
		if n, err := common.ParseInt(v); err == nil && n > 0 {
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
		if n, err := common.ParseInt(ratingValue); err == nil {
			rating = &n
		}
	}
	status := feedbacksvc.NormalizeStatus(c.Query("status"))
	if c.Query("status") != "" && !feedbacksvc.IsValidStatus(status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feedback status"})
		return
	}
	hasAttachment := strings.ToLower(strings.TrimSpace(c.Query("has_attachment")))

	var fromTime *time.Time
	if v := strings.TrimSpace(c.Query("from")); v != "" {
		if t, err := common.ParseTimeFlexible(v); err == nil {
			fromTime = &t
		}
	}
	var toTime *time.Time
	if v := strings.TrimSpace(c.Query("to")); v != "" {
		if t, err := common.ParseTimeFlexible(v); err == nil {
			end := t
			if len(v) == len("2006-01-02") {
				end = end.Add(24*time.Hour - time.Nanosecond)
			}
			toTime = &end
		}
	}

	items, total, statusCounts, err := feedbacksvc.NewService(h.DB).List(orgID, appID, feedbacksvc.ListFilter{
		Keyword:        keyword,
		Rating:         rating,
		Status:         status,
		StatusProvided: c.Query("status") != "",
		HasAttachment:  hasAttachment,
		FromTime:       fromTime,
		ToTime:         toTime,
		Limit:          pageSize,
		Offset:         offset,
	})
	if err != nil {
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
	if _, err := appsvc.NewService(h.DB).GetForOrg(orgID, appID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	if !h.requireAppFeedbackPermission(c, appID) {
		return
	}
	if !schema.HasFeedbackTable(h.DB) || !schema.HasFeedbackWorkflowColumns(h.DB) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "feedback_not_ready"})
		return
	}

	feedback, err := feedbacksvc.NewService(h.DB).GetForApp(orgID, appID, feedbackID)
	if err != nil {
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
	if _, err := appsvc.NewService(h.DB).GetForOrg(orgID, appID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}
	if !h.requireAppFeedbackPermission(c, appID) {
		return
	}
	if !schema.HasFeedbackTable(h.DB) || !schema.HasFeedbackWorkflowColumns(h.DB) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "feedback_not_ready"})
		return
	}

	var req updateFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	feedback, err := feedbacksvc.NewService(h.DB).Update(orgID, appID, feedbackID, feedbacksvc.UpdateInput{
		Status:       req.Status,
		InternalNote: req.InternalNote,
		UserID:       userID,
	})
	if err != nil {
		switch {
		case errors.Is(err, feedbacksvc.ErrInvalidStatus):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feedback status"})
		case errors.Is(err, feedbacksvc.ErrInternalNoteTooLong):
			c.JSON(http.StatusBadRequest, gin.H{"error": "internal_note too long"})
		case errors.Is(err, feedbacksvc.ErrNoUpdates):
			c.JSON(http.StatusBadRequest, gin.H{"error": "no updates"})
		case errors.Is(err, feedbacksvc.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "feedback not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update feedback"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"feedback": feedback})
}
