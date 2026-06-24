package audit

import (
	"encoding/json"

	"software-web-manager/backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Service writes audit records over a single database handle.
type Service struct {
	DB *gorm.DB
}

// NewService builds an audit service from a database handle.
func NewService(db *gorm.DB) *Service {
	return &Service{DB: db}
}

// Record inserts an audit-log entry. before/after are JSON-encoded when non-nil.
// The insert error is returned but is typically ignored by callers (audit is
// best-effort).
func (s *Service) Record(orgID, userID uuid.UUID, action, targetType string, targetID uuid.UUID, ipAddress, userAgent string, before any, after any) error {
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
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		BeforeJSON: beforeJSON,
		AfterJSON:  afterJSON,
	}
	return s.DB.Create(&log).Error
}
