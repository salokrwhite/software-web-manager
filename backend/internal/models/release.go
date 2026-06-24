package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Release struct {
	ID                  uuid.UUID  `gorm:"type:char(36);primaryKey"`
	AppID               uuid.UUID  `gorm:"type:char(36);not null;index"`
	Version             string     `gorm:"not null"`
	VersionCode         *int       `gorm:""`
	Notes               string     `gorm:""`
	ExternalDownloadURL string     `gorm:"column:external_download_url;type:varchar(2048)"`
	ReleaseTemplateID   *uuid.UUID `gorm:"type:char(36)"`
	Status              string     `gorm:"not null;default:'draft'"`
	SubmittedAt         *time.Time
	ApprovedAt          *time.Time
	ApprovedBy          *uuid.UUID `gorm:"type:char(36)"`
	PublishedAt         *time.Time
	CreatedAt           time.Time `gorm:"autoCreateTime"`
}

func (Release) TableName() string { return "releases" }

func (r *Release) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&r.ID)
	return nil
}

type ReleaseTemplate struct {
	ID          uuid.UUID `gorm:"type:char(36);primaryKey"`
	OrgID       uuid.UUID `gorm:"type:char(36);not null;index"`
	Name        string    `gorm:"not null"`
	ScheduleAt  *time.Time
	WindowStart *time.Time
	WindowEnd   *time.Time
	Emergency   bool      `gorm:"not null;default:false"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

func (ReleaseTemplate) TableName() string { return "release_templates" }

func (t *ReleaseTemplate) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&t.ID)
	return nil
}

type Artifact struct {
	ID             uuid.UUID `gorm:"type:char(36);primaryKey"`
	ReleaseID      uuid.UUID `gorm:"type:char(36);not null;index"`
	Platform       string    `gorm:"not null"`
	Arch           string    `gorm:"not null"`
	FileType       string    `gorm:"not null"`
	Size           int64     `gorm:"not null"`
	ChecksumSHA256 string    `gorm:"not null"`
	Signature      string    `gorm:"type:text"`
	StorageDriver  string    `gorm:"not null"`
	StoragePath    string    `gorm:"not null"`
	DownloadURL    string    `gorm:""`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
}

func (Artifact) TableName() string { return "artifacts" }

func (a *Artifact) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&a.ID)
	return nil
}
