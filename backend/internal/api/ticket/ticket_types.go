package ticket

import (
	"time"
)

type ticketListItem struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Status          string    `json:"status"`
	AssigneeType    string    `json:"assignee_type"`
	AssigneeUserID  *string   `json:"assignee_user_id"`
	AssigneeEmail   string    `json:"assignee_email"`
	CreatedByEmail  string    `json:"created_by_email"`
	CreatedAt       time.Time `json:"created_at"`
	AttachmentCount int64     `json:"attachment_count"`
}

type systemTicketListItem struct {
	ID              string    `json:"id"`
	OrgID           string    `json:"org_id"`
	OrgName         string    `json:"org_name"`
	Title           string    `json:"title"`
	Status          string    `json:"status"`
	AssigneeType    string    `json:"assignee_type"`
	AssigneeUserID  *string   `json:"assignee_user_id"`
	AssigneeEmail   string    `json:"assignee_email"`
	CreatedByEmail  string    `json:"created_by_email"`
	CreatedAt       time.Time `json:"created_at"`
	AttachmentCount int64     `json:"attachment_count"`
}

type ticketDetailRow struct {
	ID             string     `json:"id"`
	OrgID          string     `json:"org_id"`
	OrgName        string     `json:"org_name,omitempty"`
	CreatedBy      string     `json:"created_by"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	Status         string     `json:"status"`
	AssigneeType   string     `json:"assignee_type"`
	AssigneeUserID *string    `json:"assignee_user_id"`
	CreatedByEmail string     `json:"created_by_email"`
	AssigneeEmail  string     `json:"assignee_email"`
	InProgressAt   *time.Time `json:"in_progress_at"`
	ResolvedAt     *time.Time `json:"resolved_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type ticketAttachmentResponse struct {
	ID          string    `json:"id"`
	FileName    string    `json:"file_name"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	DownloadURL string    `json:"download_url"`
	CreatedAt   time.Time `json:"created_at"`
}

type ticketDetailResponse struct {
	ticketDetailRow
	Attachments []ticketAttachmentResponse `json:"attachments"`
}

type ticketMessageAttachmentResponse struct {
	ID          string    `json:"id"`
	FileName    string    `json:"file_name"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	DownloadURL string    `json:"download_url"`
	CreatedAt   time.Time `json:"created_at"`
}

type ticketMessageResponse struct {
	ID          string                            `json:"id"`
	TicketID    string                            `json:"ticket_id"`
	SenderType  string                            `json:"sender_type"`
	UserID      string                            `json:"user_id"`
	UserEmail   string                            `json:"user_email"`
	Content     string                            `json:"content"`
	CreatedAt   time.Time                         `json:"created_at"`
	Attachments []ticketMessageAttachmentResponse `json:"attachments"`
}

type updateTicketStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type batchDeleteTicketsRequest struct {
	IDs []string `json:"ids" binding:"required"`
}
