package system

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type systemOrgListItem struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Plan        string     `json:"plan"`
	Status      string     `json:"status"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	ApprovedBy  *uuid.UUID `json:"approved_by"`
	ApprovedAt  *time.Time `json:"approved_at"`
	OwnerEmail  string     `json:"owner_email"`
	MemberCount int64      `json:"member_count"`
	AppCount    int64      `json:"app_count"`
}

type createSystemOrgRequest struct {
	OrgName    string  `json:"org_name" binding:"required"`
	OwnerEmail string  `json:"owner_email" binding:"required,email"`
	Password   string  `json:"password"`
	Plan       *string `json:"plan"`
}

type rejectSystemOrgRequest struct {
	Reason        *string `json:"reason"`
	AllowResubmit bool    `json:"allow_resubmit"`
}

type impersonateRequest struct {
	OrgID string `json:"org_id" binding:"required"`
	Role  string `json:"role"`
}

type systemOverviewResponse struct {
	Orgs    map[string]int64 `json:"orgs"`
	Users   map[string]int64 `json:"users"`
	Apps    map[string]int64 `json:"apps"`
	Devices map[string]int64 `json:"devices"`
	Events  map[string]int64 `json:"events"`
	Daily   []gin.H          `json:"daily"`
}

type systemAppItem struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	Slug            string     `json:"slug"`
	OrgID           uuid.UUID  `json:"org_id"`
	OrgName         string     `json:"org_name"`
	OrgStatus       string     `json:"org_status"`
	OrgType         string     `json:"org_type"`
	OwnerEmail      string     `json:"owner_email"`
	CreatedAt       time.Time  `json:"created_at"`
	Status          string     `json:"status"`
	SubmittedAt     *time.Time `json:"submitted_at"`
	RejectionReason *string    `json:"rejection_reason"`
	ReleaseCount    int64      `json:"release_count"`
	MemberCount     int64      `json:"member_count"`
	DeviceCount     int64      `json:"device_count"`
	SubmitNote      string     `json:"submit_note"`
}

type systemReleaseItem struct {
	ID          uuid.UUID  `json:"id"`
	Version     string     `json:"version"`
	Status      string     `json:"status"`
	SubmittedAt *time.Time `json:"submitted_at"`
	CreatedAt   time.Time  `json:"created_at"`
	AppID       uuid.UUID  `json:"app_id"`
	AppName     string     `json:"app_name"`
	AppSlug     string     `json:"app_slug"`
	OrgID       uuid.UUID  `json:"org_id"`
	OrgName     string     `json:"org_name"`
	OrgType     string     `json:"org_type"`
	OwnerEmail  string     `json:"owner_email"`
	SubmitNote  string     `json:"submit_note"`
}

type systemAuditItem struct {
	ID         uuid.UUID `json:"id"`
	OrgID      uuid.UUID `json:"org_id"`
	OrgName    string    `json:"org_name"`
	UserID     uuid.UUID `json:"user_id"`
	UserEmail  string    `json:"user_email"`
	Action     string    `json:"action"`
	TargetType string    `json:"target_type"`
	TargetID   string    `json:"target_id"`
	IPAddress  string    `json:"ip_address"`
	CreatedAt  time.Time `json:"created_at"`
}

type deleteSystemAuditLogsRequest struct {
	IDs []string `json:"ids" binding:"required"`
}

type batchDeleteSystemOrgsRequest struct {
	IDs []string `json:"ids" binding:"required"`
}

type batchDeleteSystemAppsRequest struct {
	IDs []string `json:"ids" binding:"required"`
}

type batchDeleteSystemOrgApprovalLogsRequest struct {
	IDs []string `json:"ids" binding:"required"`
}

type batchDeleteSystemAppApprovalLogsRequest struct {
	IDs []string `json:"ids" binding:"required"`
}

type batchDeleteSystemReleaseApprovalLogsRequest struct {
	IDs []string `json:"ids" binding:"required"`
}

type rejectSystemAppRequest struct {
	Reason *string `json:"reason"`
}

type orgRegistrationMaterialItem struct {
	ID          uuid.UUID `json:"id"`
	FileName    string    `json:"file_name"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	DownloadURL string    `json:"download_url"`
}
