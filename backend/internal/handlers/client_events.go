package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
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

func (h *Handler) IngestEvents(c *gin.Context) {
	app, org, ok := clientAppOrgFromContext(c)
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

	var batch eventBatchRequest
	if err := json.Unmarshal(rawBody, &batch); err == nil && len(batch.Events) > 0 {
		for _, ev := range batch.Events {
			if err := h.ingestEvent(app, org, ev, c.ClientIP()); err != nil {
				if errors.Is(err, errInsufficientScope) {
					c.JSON(http.StatusForbidden, gin.H{"error": "insufficient scope"})
					return
				}
				if errors.Is(err, errDeviceBlocked) {
					h.writeDeviceBlocked(c, nil)
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ingest events"})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	var req eventIngestRequest
	if err := json.Unmarshal(rawBody, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.ingestEvent(app, org, req, c.ClientIP()); err != nil {
		if errors.Is(err, errInsufficientScope) {
			c.JSON(http.StatusForbidden, gin.H{"error": "insufficient scope"})
			return
		}
		if errors.Is(err, errAppPending) {
			c.JSON(http.StatusForbidden, gin.H{"error": "app_pending_review"})
			return
		}
		if errors.Is(err, errAppRejected) {
			c.JSON(http.StatusForbidden, gin.H{"error": "app_rejected"})
			return
		}
		if errors.Is(err, errDeviceBlocked) {
			h.writeDeviceBlocked(c, nil)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ingest event"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) ingestEvent(app models.App, org models.Org, req eventIngestRequest, clientIP string) error {
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
	attrs := normalizeAttributes(req.Attributes)
	region := resolveRegion(h, attrs, clientIP)
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
		_ = h.upsertDevice(app.ID, req.DeviceID, "", "", attrs, appVersion, "")
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
			if id := parseReleaseID(toString(v)); id != nil {
				releaseID = id
			}
		} else if v, ok := req.Properties["releaseId"]; ok {
			if id := parseReleaseID(toString(v)); id != nil {
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
