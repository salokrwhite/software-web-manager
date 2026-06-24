package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AuditLog struct {
	ID         uuid.UUID      `gorm:"type:char(36);primaryKey"`
	OrgID      uuid.UUID      `gorm:"type:char(36);not null;index"`
	UserID     uuid.UUID      `gorm:"type:char(36);not null;index"`
	Action     string         `gorm:"not null"`
	TargetType string         `gorm:"not null"`
	TargetID   uuid.UUID      `gorm:"type:char(36)"`
	IPAddress  string         `gorm:""`
	UserAgent  string         `gorm:""`
	BeforeJSON datatypes.JSON `gorm:"type:json"`
	AfterJSON  datatypes.JSON `gorm:"type:json"`
	CreatedAt  time.Time      `gorm:"autoCreateTime"`
}

func (AuditLog) TableName() string { return "audit_logs" }

func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&a.ID)
	return nil
}
