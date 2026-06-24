// Package device encapsulates device-control persistence: checking whether a
// device is blocked and toggling its blocked state. It owns only data access and
// never touches gin or writes HTTP responses.
package device

import (
	"strings"
	"time"

	"software-web-manager/backend/internal/db/schema"
	"software-web-manager/backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DefaultBlockReason is recorded when a block request carries no explicit reason.
const DefaultBlockReason = "manual_remove"

// Service provides device-control data access over a single database handle.
type Service struct {
	DB *gorm.DB
}

// NewService builds a device service from a database handle.
func NewService(db *gorm.DB) *Service {
	return &Service{DB: db}
}

func (s *Service) hasControlsTable() bool {
	if s == nil {
		return false
	}
	return schema.HasDeviceControlsTable(s.DB)
}

// IsBlocked reports whether the given device is currently blocked for an app.
// When the device_controls table is absent it conservatively returns false.
func (s *Service) IsBlocked(appID uuid.UUID, deviceID string) (bool, *models.DeviceControl, error) {
	if s == nil || s.DB == nil || !s.hasControlsTable() {
		return false, nil, nil
	}
	deviceID = strings.TrimSpace(deviceID)
	if appID == uuid.Nil || deviceID == "" {
		return false, nil, nil
	}
	var control models.DeviceControl
	if err := s.DB.Where("app_id = ? AND device_id = ? AND blocked = 1", appID, deviceID).First(&control).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, &control, nil
}

// SetBlocked blocks a device, upserting the device-control row and returning the
// persisted record.
func (s *Service) SetBlocked(appID uuid.UUID, deviceID, reason, actorID string) (models.DeviceControl, error) {
	now := time.Now()
	deviceID = strings.TrimSpace(deviceID)
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = DefaultBlockReason
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
	err := s.DB.Clauses(clause.OnConflict{
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
	if err := s.DB.Where("app_id = ? AND device_id = ?", appID, deviceID).First(&out).Error; err != nil {
		return models.DeviceControl{}, err
	}
	return out, nil
}

// SetUnblocked clears the blocked flag for a device and returns the persisted
// record.
func (s *Service) SetUnblocked(appID uuid.UUID, deviceID, actorID string) (models.DeviceControl, error) {
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
	err := s.DB.Clauses(clause.OnConflict{
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
	if err := s.DB.Where("app_id = ? AND device_id = ?", appID, deviceID).First(&out).Error; err != nil {
		return models.DeviceControl{}, err
	}
	return out, nil
}
