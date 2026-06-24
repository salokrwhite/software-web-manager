package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type App struct {
	ID                       uuid.UUID      `gorm:"type:char(36);primaryKey"`
	OrgID                    uuid.UUID      `gorm:"type:char(36);not null;index"`
	Name                     string         `gorm:"not null"`
	Slug                     string         `gorm:"not null"`
	Description              string         `gorm:""`
	PublicKey                string         `gorm:"type:text"`
	AppSecretCiphertext      string         `gorm:"column:app_secret_ciphertext;type:text;not null" json:"-"`
	AppSecretUpdatedAt       *time.Time     `gorm:"column:app_secret_updated_at" json:"-"`
	AppSecretScopesJSON      datatypes.JSON `gorm:"column:app_secret_scopes;type:json" json:"-"`
	AppSecretExpiresAt       *time.Time     `gorm:"column:app_secret_expires_at" json:"-"`
	AppSecretName            string         `gorm:"column:app_secret_name;type:varchar(128);not null;default:'app_secret'" json:"-"`
	RegionRulesJSON          datatypes.JSON `gorm:"type:json"`
	FeedbackEnabled          bool           `gorm:"not null;default:true"`
	HeartbeatIntervalSeconds int            `gorm:"not null;default:60"`
	OnlineEnabled            bool           `gorm:"not null;default:true"`
	MaintenanceEnabled       bool           `gorm:"not null;default:false"`
	MaintenanceStartAt       *time.Time     `gorm:"column:maintenance_start_at"`
	MaintenanceMessage       string         `gorm:"type:varchar(500);not null;default:''"`
	Status                   string         `gorm:"type:varchar(32);not null;default:'active'"`
	SubmittedAt              *time.Time
	ApprovedBy               *uuid.UUID `gorm:"type:char(36)"`
	ApprovedAt               *time.Time
	RejectedBy               *uuid.UUID `gorm:"type:char(36)"`
	RejectedAt               *time.Time
	RejectionReason          *string   `gorm:"type:text"`
	CreatedAt                time.Time `gorm:"autoCreateTime"`
}

func (App) TableName() string { return "apps" }

func (a *App) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&a.ID)
	return nil
}

type AppSecret struct {
	ID               uuid.UUID      `gorm:"type:char(36);primaryKey"`
	AppID            uuid.UUID      `gorm:"type:char(36);not null;index"`
	Name             string         `gorm:"type:varchar(128);not null;default:'app_secret'"`
	SecretCiphertext string         `gorm:"column:secret_ciphertext;type:text;not null"`
	ScopesJSON       datatypes.JSON `gorm:"column:scopes_json;type:json"`
	ExpiresAt        *time.Time
	LastUsedAt       *time.Time
	RevokedAt        *time.Time
	CreatedAt        time.Time `gorm:"autoCreateTime"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime"`
}

func (AppSecret) TableName() string { return "app_secrets" }

func (s *AppSecret) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&s.ID)
	return nil
}

type Attachment struct {
	ID            uuid.UUID  `gorm:"type:char(36);primaryKey"`
	OwnerType     string     `gorm:"type:varchar(64);not null;index:idx_attachments_owner,priority:1"`
	OwnerID       uuid.UUID  `gorm:"type:char(36);not null;index:idx_attachments_owner,priority:2"`
	OrgID         *uuid.UUID `gorm:"type:char(36);index"`
	FileName      string     `gorm:"type:varchar(255);not null"`
	ContentType   string     `gorm:"type:varchar(255);not null"`
	Size          int64      `gorm:"not null"`
	StorageDriver string     `gorm:"type:varchar(32);not null"`
	StoragePath   string     `gorm:"type:varchar(1024);not null"`
	CreatedBy     *uuid.UUID `gorm:"type:char(36)"`
	CreatedAt     time.Time  `gorm:"autoCreateTime"`
}

func (Attachment) TableName() string { return "attachments" }

func (a *Attachment) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&a.ID)
	return nil
}
