package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
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
	CreatedAt    time.Time `gorm:"autoCreateTime"`
}

func (User) TableName() string { return "users" }

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

// Scope discriminators for the unified memberships table.
const (
	ScopeOrg = "org"
	ScopeApp = "app"
)

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

type DailyMetric struct {
	Date      time.Time `gorm:"type:date;primaryKey"`
	AppID     uuid.UUID `gorm:"type:char(36);primaryKey"`
	ChannelID uuid.UUID `gorm:"type:char(36);primaryKey"`
	EventName string    `gorm:"primaryKey"`
	Count     int64     `gorm:"not null"`
}

func (DailyMetric) TableName() string { return "daily_metrics" }

type AuditLog struct {
	ID         uuid.UUID      `gorm:"type:char(36);primaryKey"`
	OrgID      uuid.UUID      `gorm:"type:char(36);not null;index"`
	UserID     uuid.UUID      `gorm:"type:char(36);not null;index"`
	Action     string         `gorm:"not null"`
	TargetType string         `gorm:"not null"`
	TargetID   uuid.UUID      `gorm:"type:char(36)"`
	IPAddress  string         `gorm:""`
	UserAgent  string         `gorm:""`
	BeforeJSON datatypes.JSON `gorm:"type:json"`
	AfterJSON  datatypes.JSON `gorm:"type:json"`
	CreatedAt  time.Time      `gorm:"autoCreateTime"`
}

func (AuditLog) TableName() string { return "audit_logs" }

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

func ensureUUID(id *uuid.UUID) {
	if id == nil || *id != uuid.Nil {
		return
	}
	*id = uuid.New()
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&u.ID)
	return nil
}

func (o *Org) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&o.ID)
	return nil
}

func (a *App) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&a.ID)
	return nil
}

func (c *Channel) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&c.ID)
	return nil
}

func (r *Release) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&r.ID)
	return nil
}

func (t *ReleaseTemplate) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&t.ID)
	return nil
}

func (rc *ReleaseChannel) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&rc.ID)
	return nil
}

func (r *OrgRole) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&r.ID)
	return nil
}

func (a *Artifact) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&a.ID)
	return nil
}

func (s *AppSecret) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&s.ID)
	return nil
}

func (a *Attachment) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&a.ID)
	return nil
}

func (d *Device) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&d.ID)
	return nil
}

func (d *DeviceControl) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&d.ID)
	return nil
}

func (e *Event) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&e.ID)
	return nil
}

func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&a.ID)
	return nil
}

func (i *OrgInvite) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&i.ID)
	return nil
}

func (p *OrgRolePermission) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&p.ID)
	return nil
}

func (r *OrgJoinRequest) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&r.ID)
	return nil
}

func (s *SystemSetting) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&s.ID)
	return nil
}

func (v *EmailVerificationCode) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&v.ID)
	return nil
}
