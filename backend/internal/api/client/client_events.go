package client

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"io"
	"net/http"
	"software-web-manager/backend/internal/api/common"
	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"
	"strings"
	"time"
)

type eventIngestRequest struct {
	DeviceID    string                 `json:"device_id"`
	EventName   string                 `json:"event_name"`
	EventTime   *time.Time             `json:"event_time"`
	ChannelCode string                 `json:"channel_code"`
	Properties  map[string]interface{} `json:"properties"`
	Attributes  map[string]interface{} `json:"attributes"`
}

type eventBatchRequest struct {
	Events []eventIngestRequest `json:"events"`
}

var errDeviceBlocked = errors.New("device_blocked")

// eventsOKResponse builds the success body, attaching a signed authz verdict
// (bound to the reporting device) so clients that re-verify per call can fail
// closed if they get revoked while running.
func (h *Handler) eventsOKResponse(c *gin.Context, app models.App, deviceID string) gin.H {
	resp := gin.H{"status": "ok"}
	if env := h.SignAuthzForRequest(c, app, deviceID); env != nil {
		resp["authz"] = env
	}
	return resp
}

func firstEventDeviceID(events []eventIngestRequest) string {
	for i := range events {
		if d := strings.TrimSpace(events[i].DeviceID); d != "" {
			return d
		}
	}
	return ""
}

func (h *Handler) IngestEvents(c *gin.Context) {
	app, org, ok := middleware.ClientAppOrgFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if len(strings.TrimSpace(string(rawBody))) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "EOF"})
		return
	}

	esa := h.readESAGeo(c)
	clientIP := esa.realIPOr(c.ClientIP())

	var batch eventBatchRequest
	if err := json.Unmarshal(rawBody, &batch); err == nil && len(batch.Events) > 0 {
		for _, ev := range batch.Events {
			if err := h.ingestEvent(app, org, ev, esa, clientIP); err != nil {
				if errors.Is(err, ErrInsufficientScope) {
					c.JSON(http.StatusForbidden, gin.H{"error": "insufficient scope"})
					return
				}
				if errors.Is(err, errDeviceBlocked) {
					h.WriteDeviceBlocked(c, nil)
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ingest events"})
				return
			}
		}
		c.JSON(http.StatusOK, h.eventsOKResponse(c, app, firstEventDeviceID(batch.Events)))
		return
	}

	var req eventIngestRequest
	if err := json.Unmarshal(rawBody, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.ingestEvent(app, org, req, esa, clientIP); err != nil {
		if errors.Is(err, ErrInsufficientScope) {
			c.JSON(http.StatusForbidden, gin.H{"error": "insufficient scope"})
			return
		}
		if errors.Is(err, ErrAppPending) {
			c.JSON(http.StatusForbidden, gin.H{"error": "app_pending_review"})
			return
		}
		if errors.Is(err, ErrAppRejected) {
			c.JSON(http.StatusForbidden, gin.H{"error": "app_rejected"})
			return
		}
		if errors.Is(err, errDeviceBlocked) {
			h.WriteDeviceBlocked(c, nil)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ingest event"})
		return
	}
	c.JSON(http.StatusOK, h.eventsOKResponse(c, app, strings.TrimSpace(req.DeviceID)))
}

func (h *Handler) ingestEvent(app models.App, org models.Org, req eventIngestRequest, esa esaGeo, clientIP string) error {
	if strings.TrimSpace(req.DeviceID) != "" {
		blocked, _, err := h.IsDeviceBlocked(app.ID, req.DeviceID)
		if err != nil {
			return err
		}
		if blocked {
			return errDeviceBlocked
		}
	}
	if req.EventName == "" {
		return errors.New("event_name required")
	}
	attrs := NormalizeAttributes(req.Attributes)
	region := h.ResolveRegion(esa, attrs, clientIP)
	if region.ISO != "" && attrs["country_iso"] == "" {
		attrs["country_iso"] = region.ISO
	}
	if attrs["country"] == "" {
		if region.Country != "" {
			attrs["country"] = region.Country
		} else if region.ISO != "" {
			attrs["country"] = region.ISO
		}
	}
	appVersion := ""
	if v, ok := req.Properties["version"].(string); ok {
		appVersion = v
	}
	if req.DeviceID != "" {
		_ = h.UpsertDevice(app.ID, req.DeviceID, "", "", attrs, appVersion, "")
	}
	if req.ChannelCode == "" {
		var channel models.Channel
		if err := h.DB.Where("app_id = ? AND is_default = true", app.ID).First(&channel).Error; err == nil {
			req.ChannelCode = channel.Code
		}
	}
	eventTime := time.Now()
	if req.EventTime != nil {
		eventTime = *req.EventTime
	}
	payload, _ := json.Marshal(req.Properties)
	var releaseID *uuid.UUID
	if req.Properties != nil {
		if v, ok := req.Properties["release_id"]; ok {
			if id := parseReleaseID(common.ToString(v)); id != nil {
				releaseID = id
			}
		} else if v, ok := req.Properties["releaseId"]; ok {
			if id := parseReleaseID(common.ToString(v)); id != nil {
				releaseID = id
			}
		}
	}
	event := models.Event{
		OrgID:       org.ID,
		AppID:       app.ID,
		ReleaseID:   releaseID,
		DeviceID:    req.DeviceID,
		EventName:   req.EventName,
		EventTime:   eventTime,
		ChannelCode: req.ChannelCode,
		Properties:  datatypes.JSON(payload),
	}
	return h.DB.Create(&event).Error
}

func parseReleaseID(raw string) *uuid.UUID {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil
	}
	return &id
}
