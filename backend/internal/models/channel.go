package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Channel struct {
	ID                  uuid.UUID `gorm:"type:char(36);primaryKey"`
	AppID               uuid.UUID `gorm:"type:char(36);not null;index"`
	Name                string    `gorm:"not null"`
	Code                string    `gorm:"not null"`
	IsDefault           bool      `gorm:"not null;default:false"`
	MinSupportedVersion string    `gorm:""`
	PreviewToken        string    `gorm:""`
	CreatedAt           time.Time `gorm:"autoCreateTime"`
}

func (Channel) TableName() string { return "channels" }

func (c *Channel) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&c.ID)
	return nil
}

type ReleaseChannel struct {
	ID                 uuid.UUID      `gorm:"type:char(36);primaryKey"`
	ReleaseID          uuid.UUID      `gorm:"type:char(36);not null;index"`
	ChannelID          uuid.UUID      `gorm:"type:char(36);not null;index"`
	RolloutPercent     int            `gorm:"not null;default:100"`
	Mandatory          bool           `gorm:"not null;default:false"`
	WhitelistJSON      datatypes.JSON `gorm:"type:json"`
	RegionRulesJSON    datatypes.JSON `gorm:"type:json"`
	Status             string         `gorm:"not null;default:'inactive'"`
	Paused             bool           `gorm:"not null;default:false"`
	TargetingRulesJSON datatypes.JSON `gorm:"type:json"`
	RolloutStartAt     *time.Time
	RolloutEndAt       *time.Time
	PublishedAt        *time.Time `gorm:""`
}

func (ReleaseChannel) TableName() string { return "release_channels" }

func (rc *ReleaseChannel) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&rc.ID)
	return nil
}
