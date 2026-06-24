package handlers

import (
	"encoding/json"
	"strings"

	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

func (h *Handler) Audit(c *gin.Context, action, targetType string, targetID uuid.UUID, before any, after any) {
	orgIDStr := c.GetString(middleware.ContextOrgID)
	userIDStr := c.GetString(middleware.ContextUserID)
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return
	}
	var beforeJSON datatypes.JSON
	var afterJSON datatypes.JSON
	if before != nil {
		if b, err := json.Marshal(before); err == nil {
			beforeJSON = datatypes.JSON(b)
		}
	}
	if after != nil {
		if b, err := json.Marshal(after); err == nil {
			afterJSON = datatypes.JSON(b)
		}
	}
	log := models.AuditLog{
		OrgID:      orgID,
		UserID:     userID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		IPAddress:  strings.TrimSpace(c.ClientIP()),
		UserAgent:  strings.TrimSpace(c.GetHeader("User-Agent")),
		BeforeJSON: beforeJSON,
		AfterJSON:  afterJSON,
	}
	_ = h.DB.Create(&log).Error
}

func (h *Handler) AuditWithOrg(c *gin.Context, orgID uuid.UUID, action, targetType string, targetID uuid.UUID, before any, after any) {
	userIDStr := c.GetString(middleware.ContextUserID)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return
	}
	var beforeJSON datatypes.JSON
	var afterJSON datatypes.JSON
	if before != nil {
		if b, err := json.Marshal(before); err == nil {
			beforeJSON = datatypes.JSON(b)
		}
	}
	if after != nil {
		if b, err := json.Marshal(after); err == nil {
			afterJSON = datatypes.JSON(b)
		}
	}
	log := models.AuditLog{
		OrgID:      orgID,
		UserID:     userID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		IPAddress:  strings.TrimSpace(c.ClientIP()),
		UserAgent:  strings.TrimSpace(c.GetHeader("User-Agent")),
		BeforeJSON: beforeJSON,
		AfterJSON:  afterJSON,
	}
	_ = h.DB.Create(&log).Error
}

