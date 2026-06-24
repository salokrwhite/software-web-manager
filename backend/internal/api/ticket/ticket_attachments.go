package ticket

import (
	"path/filepath"
	"software-web-manager/backend/internal/api/common"
	attachment "software-web-manager/backend/internal/services/attachment"

	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) storeTicketAttachments(c *gin.Context, ticketID uuid.UUID, orgID uuid.UUID, createdBy uuid.UUID) ([]models.Attachment, int, error) {
	return common.StoreAttachments(h.Storage, h.Cfg.StorageDriver, c, attachment.OwnerTicket, ticketID, &orgID, &createdBy, "attachments", "tickets", maxTicketAttachments, maxTicketAttachmentSize)
}

func (h *Handler) loadTicketAttachments(c *gin.Context, ticketID string) ([]ticketAttachmentResponse, error) {
	if err := h.EnsureStorage(); err != nil {
		return nil, err
	}
	items, err := common.LoadAttachmentResponses(h.DB, h.Storage, h.Cfg, c, attachment.OwnerTicket, ticketID)
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
	return common.StoreAttachments(h.Storage, h.Cfg.StorageDriver, c, attachment.OwnerTicketMessage, messageID, &orgID, &createdBy, "attachments", filepath.ToSlash(filepath.Join("tickets", ticketID, "messages")), maxTicketAttachments, maxTicketAttachmentSize)
}
