package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TicketMessage struct {
	ID         uuid.UUID `gorm:"type:char(36);primaryKey"`
	TicketID   uuid.UUID `gorm:"type:char(36);not null;index"`
	OrgID      uuid.UUID `gorm:"type:char(36);not null;index"`
	UserID     uuid.UUID `gorm:"type:char(36);not null;index"`
	SenderType string    `gorm:"type:varchar(32);not null"`
	Content    string    `gorm:"type:text"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}

func (TicketMessage) TableName() string { return "ticket_messages" }

func (m *TicketMessage) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&m.ID)
	return nil
}
