package handlers

import (
	"os"
	"path/filepath"
	"strings"

	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) storeTicketAttachments(c *gin.Context, ticketID uuid.UUID, orgID uuid.UUID, createdBy uuid.UUID) ([]models.Attachment, int, error) {
	return h.storeAttachments(c, attachmentOwnerTicket, ticketID, &orgID, &createdBy, "attachments", "tickets", maxTicketAttachments, maxTicketAttachmentSize)
}

func (h *Handler) loadTicketAttachments(c *gin.Context, ticketID string) ([]ticketAttachmentResponse, error) {
	items, err := h.loadAttachmentResponses(c, attachmentOwnerTicket, ticketID)
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
	return h.storeAttachments(c, attachmentOwnerTicketMessage, messageID, &orgID, &createdBy, "attachments", filepath.ToSlash(filepath.Join("tickets", ticketID, "messages")), maxTicketAttachments, maxTicketAttachmentSize)
}

func (h *Handler) deleteStoragePaths(c *gin.Context, paths []string) {
	if len(paths) == 0 {
		return
	}
	if err := h.ensureStorage(c); err != nil {
		return
	}
	if h.Storage == nil {
		return
	}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		_ = h.Storage.Delete(c.Request.Context(), path)
	}
}

func (h *Handler) deleteLocalTicketDir(ticketID string) {
	if !strings.EqualFold(h.Cfg.StorageDriver, "local") {
		return
	}
	root := strings.TrimSpace(h.Cfg.LocalStoragePath)
	if root == "" || ticketID == "" {
		return
	}
	dir := filepath.Join(root, "tickets", ticketID)
	_ = os.RemoveAll(dir)
}
