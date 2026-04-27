package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Feedback struct {
	ID           uuid.UUID      `gorm:"type:char(36);primaryKey"`
	OrgID        uuid.UUID      `gorm:"type:char(36);not null;index"`
	AppID        uuid.UUID      `gorm:"type:char(36);not null;index"`
	DeviceID     string         `gorm:"type:varchar(255);not null"`
	ChannelCode  string         `gorm:"type:varchar(64)"`
	AppVersion   string         `gorm:"type:varchar(128)"`
	Rating       *int           `gorm:""`
	Content      string         `gorm:"type:text"`
	Contact      string         `gorm:"type:varchar(255)"`
	Metadata     datatypes.JSON `gorm:"column:metadata_json;type:json"`
	Status       string         `gorm:"type:varchar(32);not null;default:'open'"`
	InternalNote string         `gorm:"type:text"`
	HandledBy    *uuid.UUID     `gorm:"type:char(36)"`
	HandledAt    *time.Time
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

func (Feedback) TableName() string { return "feedbacks" }

func (f *Feedback) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&f.ID)
	return nil
}
