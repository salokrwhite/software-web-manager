package ticket

import (
	"path/filepath"

	"software-web-manager/backend/internal/handlers"
	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) storeTicketAttachments(c *gin.Context, ticketID uuid.UUID, orgID uuid.UUID, createdBy uuid.UUID) ([]models.Attachment, int, error) {
	return h.StoreAttachments(c, handlers.AttachmentOwnerTicket, ticketID, &orgID, &createdBy, "attachments", "tickets", maxTicketAttachments, maxTicketAttachmentSize)
}

func (h *Handler) loadTicketAttachments(c *gin.Context, ticketID string) ([]ticketAttachmentResponse, error) {
	items, err := h.LoadAttachmentResponses(c, handlers.AttachmentOwnerTicket, ticketID)
	if err != nil {
		return nil, err
	}
	out := make([]ticketAttachmentResponse, 0, len(items))
	for _, attachment := range items {
		out = append(out, ticketAttachmentResponse{
			ID:          attachment.ID,
			FileName:    attachment.FileName,
			ContentType: attachment.ContentType,
			Size:        attachment.Size,
			DownloadURL: attachment.DownloadURL,
			CreatedAt:   attachment.CreatedAt,
		})
	}
	return out, nil
}

func (h *Handler) storeTicketMessageAttachments(c *gin.Context, ticketID string, messageID uuid.UUID, orgID uuid.UUID, createdBy uuid.UUID) ([]models.Attachment, int, error) {
	return h.StoreAttachments(c, handlers.AttachmentOwnerTicketMessage, messageID, &orgID, &createdBy, "attachments", filepath.ToSlash(filepath.Join("tickets", ticketID, "messages")), maxTicketAttachments, maxTicketAttachmentSize)
}
