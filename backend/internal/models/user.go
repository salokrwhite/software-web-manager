package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID           uuid.UUID `gorm:"type:char(36);primaryKey"`
	Email        string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	PasswordHash string    `gorm:"not null"`
	AvatarPath   string    `gorm:"type:varchar(512)"`
	Status       string    `gorm:"type:varchar(32);not null;default:'active'"`
	SystemRole   string    `gorm:"type:varchar(32);not null;default:'none'"`
	OTPSecret    *string   `gorm:"type:varchar(128)"`
	OTPEnabled   bool      `gorm:"not null;default:false"`
	SSOSub       *string   `gorm:"column:sso_sub;type:varchar(255);uniqueIndex:uniq_users_sso_sub"`
	// TokenVersion is the session epoch embedded in issued JWTs (claim "tv"); a
	// mismatch revokes the token. Bumped on password change / credential reset.
	TokenVersion int       `gorm:"column:token_version;not null;default:0" json:"-"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
}

func (User) TableName() string { return "users" }

func (u *User) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&u.ID)
	return nil
}

type EmailVerificationCode struct {
	ID        uuid.UUID `gorm:"type:char(36);primaryKey"`
	Email     string    `gorm:"type:varchar(255);not null;index:idx_email_verification_codes_lookup,priority:1"`
	Purpose   string    `gorm:"type:varchar(64);not null;index:idx_email_verification_codes_lookup,priority:2"`
	CodeHash  string    `gorm:"type:varchar(128);not null"`
	ExpiresAt time.Time `gorm:"not null;index"`
	UsedAt    *time.Time
	RequestIP string    `gorm:"type:varchar(64)"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (EmailVerificationCode) TableName() string { return "email_verification_codes" }

func (v *EmailVerificationCode) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&v.ID)
	return nil
}
