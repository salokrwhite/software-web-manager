package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SystemSetting struct {
	ID           uuid.UUID  `gorm:"type:char(36);primaryKey"`
	SettingKey   string     `gorm:"type:varchar(100);uniqueIndex;not null"`
	SettingValue string     `gorm:"type:text;not null"`
	ValueType    string     `gorm:"type:varchar(32);not null;default:'string'"`
	Description  string     `gorm:"type:varchar(255)"`
	UpdatedBy    *uuid.UUID `gorm:"type:char(36)"`
	CreatedAt    time.Time  `gorm:"autoCreateTime"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime"`
}

func (SystemSetting) TableName() string { return "system_settings" }

func (s *SystemSetting) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&s.ID)
	return nil
}
