package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Scope discriminators for the unified memberships table.
const (
	ScopeOrg = "org"
	ScopeApp = "app"
)

type Org struct {
	ID              uuid.UUID  `gorm:"type:char(36);primaryKey"`
	Name            string     `gorm:"not null"`
	Plan            string     `gorm:"not null;default:'free'"`
	OrgType         string     `gorm:"type:varchar(32);not null;default:'enterprise'"`
	Status          string     `gorm:"type:varchar(32);not null;default:'active'"`
	CreatedBy       uuid.UUID  `gorm:"type:char(36);not null"`
	ApprovedBy      *uuid.UUID `gorm:"type:char(36)"`
	ApprovedAt      *time.Time
	RejectionReason *string    `gorm:"type:text"`
	AllowResubmit   bool       `gorm:"not null;default:false"`
	ResubmitToken   *string    `gorm:"type:char(36)"`
	RejectedBy      *uuid.UUID `gorm:"type:char(36)"`
	RejectedAt      *time.Time
	CreatedAt       time.Time `gorm:"autoCreateTime"`
}

func (Org) TableName() string { return "orgs" }

func (o *Org) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&o.ID)
	return nil
}

// OrgMember and AppMember are both persisted in the unified `memberships`
// table, discriminated by the scope_type column (scope_id holds the org/app
// id). Each keeps its own struct/field names for readability. All queries
// (GORM and raw SQL) target the memberships table directly with a scope_type
// filter; no compatibility views are used.
type OrgMember struct {
	ScopeType string    `gorm:"column:scope_type;type:varchar(16);not null;default:'org'" json:"-"`
	OrgID     uuid.UUID `gorm:"column:scope_id;type:char(36);primaryKey"`
	UserID    uuid.UUID `gorm:"type:char(36);primaryKey"`
	Role      string    `gorm:"type:varchar(32);not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (OrgMember) TableName() string { return "memberships" }

func (m *OrgMember) BeforeCreate(tx *gorm.DB) error {
	if m.ScopeType == "" {
		m.ScopeType = ScopeOrg
	}
	return nil
}

type AppMember struct {
	ScopeType string    `gorm:"column:scope_type;type:varchar(16);not null;default:'app'" json:"-"`
	AppID     uuid.UUID `gorm:"column:scope_id;type:char(36);primaryKey"`
	UserID    uuid.UUID `gorm:"type:char(36);primaryKey"`
	Role      string    `gorm:"type:varchar(32);not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (AppMember) TableName() string { return "memberships" }

func (m *AppMember) BeforeCreate(tx *gorm.DB) error {
	if m.ScopeType == "" {
		m.ScopeType = ScopeApp
	}
	return nil
}

type OrgRole struct {
	ID          uuid.UUID  `gorm:"type:char(36);primaryKey"`
	OrgID       uuid.UUID  `gorm:"type:char(36);not null;index"`
	RoleName    string     `gorm:"type:varchar(64);not null"`
	IsBuiltin   bool       `gorm:"not null;default:false"`
	Description *string    `gorm:"type:varchar(255)"`
	Status      string     `gorm:"type:varchar(32);not null;default:'active'"`
	CreatedBy   *uuid.UUID `gorm:"type:char(36)"`
	CreatedAt   time.Time  `gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime"`
}

func (OrgRole) TableName() string { return "org_roles" }

func (r *OrgRole) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&r.ID)
	return nil
}

type PermissionCatalog struct {
	PermissionCode string    `gorm:"column:permission_code;type:varchar(128);primaryKey"`
	Module         string    `gorm:"type:varchar(64);not null;index"`
	Name           string    `gorm:"type:varchar(128);not null"`
	Description    *string   `gorm:"type:varchar(255)"`
	Status         string    `gorm:"type:varchar(32);not null;default:'active'"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
}

func (PermissionCatalog) TableName() string { return "permission_catalog" }

type OrgRolePermission struct {
	ID             uuid.UUID `gorm:"type:char(36);primaryKey"`
	OrgID          uuid.UUID `gorm:"type:char(36);not null;index"`
	RoleName       string    `gorm:"column:role_name;type:varchar(64);not null;index"`
	PermissionCode string    `gorm:"column:permission_code;type:varchar(128);not null;index"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
}

func (OrgRolePermission) TableName() string { return "org_role_permissions" }

func (p *OrgRolePermission) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&p.ID)
	return nil
}

type OrgInvite struct {
	ID        uuid.UUID `gorm:"type:char(36);primaryKey"`
	OrgID     uuid.UUID `gorm:"type:char(36);not null;index"`
	Email     string    `gorm:"type:varchar(255);not null"`
	Role      string    `gorm:"type:varchar(32);not null"`
	TokenHash string    `gorm:"type:varchar(255);not null;uniqueIndex"`
	ExpiresAt *time.Time
	CreatedBy uuid.UUID `gorm:"type:char(36);not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UsedAt    *time.Time
	RevokedAt *time.Time
}

func (OrgInvite) TableName() string { return "org_invites" }

func (i *OrgInvite) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&i.ID)
	return nil
}

type OrgJoinRequest struct {
	ID           uuid.UUID  `gorm:"type:char(36);primaryKey"`
	OrgID        uuid.UUID  `gorm:"type:char(36);not null;index"`
	UserID       uuid.UUID  `gorm:"type:char(36);not null;index"`
	Reason       string     `gorm:"type:text"`
	Status       string     `gorm:"type:varchar(32);not null;default:'pending'"`
	ReviewReason *string    `gorm:"type:text"`
	ReviewedBy   *uuid.UUID `gorm:"type:char(36);index"`
	ReviewedAt   *time.Time
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

func (OrgJoinRequest) TableName() string { return "org_join_requests" }

func (r *OrgJoinRequest) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&r.ID)
	return nil
}
