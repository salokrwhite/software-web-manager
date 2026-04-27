package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Ticket struct {
	ID             uuid.UUID  `gorm:"type:char(36);primaryKey"`
	OrgID          uuid.UUID  `gorm:"type:char(36);not null;index"`
	CreatedBy      uuid.UUID  `gorm:"type:char(36);not null;index"`
	Title          string     `gorm:"type:varchar(255);not null"`
	Description    string     `gorm:"type:text"`
	Status         string     `gorm:"type:varchar(32);not null;default:'submitted'"`
	AssigneeType   string     `gorm:"type:varchar(32);not null;default:'system'"`
	AssigneeUserID *uuid.UUID `gorm:"type:char(36);index"`
	InProgressAt   *time.Time
	ResolvedAt     *time.Time
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
}

func (Ticket) TableName() string { return "tickets" }

func (t *Ticket) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&t.ID)
	return nil
}
