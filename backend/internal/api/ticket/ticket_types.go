package ticket

import (
	"time"

	ticketsvc "software-web-manager/backend/internal/services/ticket"
)

type ticketAttachmentResponse struct {
	ID          string    `json:"id"`
	FileName    string    `json:"file_name"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	DownloadURL string    `json:"download_url"`
	CreatedAt   time.Time `json:"created_at"`
}

type ticketDetailResponse struct {
	ticketsvc.TicketDetailRow
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
