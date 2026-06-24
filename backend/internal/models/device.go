package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Device struct {
	ID          uuid.UUID `gorm:"type:char(36);primaryKey"`
	AppID       uuid.UUID `gorm:"type:char(36);not null;index"`
	DeviceID    string    `gorm:"not null"`
	Platform    string    `gorm:"not null"`
	Arch        string    `gorm:"not null"`
	OSVersion   string    `gorm:""`
	Country     string    `gorm:""`
	AppVersion  string    `gorm:""`
	UserID      string    `gorm:""`
	LastIP      string    `gorm:""`
	FirstSeenAt time.Time `gorm:"autoCreateTime"`
	LastSeenAt  time.Time `gorm:"autoUpdateTime"`
}

func (Device) TableName() string { return "devices" }

func (d *Device) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&d.ID)
	return nil
}

type DeviceControl struct {
	ID          uuid.UUID `gorm:"type:char(36);primaryKey"`
	AppID       uuid.UUID `gorm:"type:char(36);not null;index"`
	DeviceID    string    `gorm:"type:varchar(255);not null;index"`
	Blocked     bool      `gorm:"not null;default:true"`
	Reason      *string   `gorm:"type:varchar(255)"`
	BlockedAt   *time.Time
	BlockedBy   *uuid.UUID `gorm:"type:char(36)"`
	UnblockedAt *time.Time
	UnblockedBy *uuid.UUID `gorm:"type:char(36)"`
	CreatedAt   time.Time  `gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime"`
}

func (DeviceControl) TableName() string { return "device_controls" }

func (d *DeviceControl) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&d.ID)
	return nil
}

type Event struct {
	ID          uuid.UUID      `gorm:"type:char(36);primaryKey"`
	OrgID       uuid.UUID      `gorm:"type:char(36);not null;index"`
	AppID       uuid.UUID      `gorm:"type:char(36);not null;index"`
	ReleaseID   *uuid.UUID     `gorm:"type:char(36);index"`
	DeviceID    string         `gorm:"not null"`
	EventName   string         `gorm:"not null"`
	EventTime   time.Time      `gorm:"not null;index"`
	ChannelCode string         `gorm:"not null"`
	Properties  datatypes.JSON `gorm:"column:properties_jsonb;type:json"`
}

func (Event) TableName() string { return "events" }

func (e *Event) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&e.ID)
	return nil
}

type DailyMetric struct {
	Date      time.Time `gorm:"type:date;primaryKey"`
	AppID     uuid.UUID `gorm:"type:char(36);primaryKey"`
	ChannelID uuid.UUID `gorm:"type:char(36);primaryKey"`
	EventName string    `gorm:"primaryKey"`
	Count     int64     `gorm:"not null"`
}

func (DailyMetric) TableName() string { return "daily_metrics" }
