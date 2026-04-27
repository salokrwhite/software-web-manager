package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
)

func (h *Handler) loadTicketMessages(c *gin.Context, ticketID string) ([]ticketMessageResponse, error) {
	type ticketMessageRow struct {
		ID         string    `json:"id"`
		TicketID   string    `json:"ticket_id"`
		SenderType string    `json:"sender_type"`
		UserID     string    `json:"user_id"`
		UserEmail  string    `json:"user_email"`
		Content    string    `json:"content"`
		CreatedAt  time.Time `json:"created_at"`
	}

	var rows []ticketMessageRow
	if err := h.DB.Table("ticket_messages tm").
		Select("tm.id, tm.ticket_id, tm.sender_type, tm.user_id, tm.content, tm.created_at, u.email as user_email").
		Joins("LEFT JOIN users u ON u.id = tm.user_id").
		Where("tm.ticket_id = ?", ticketID).
		Order("tm.created_at asc").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []ticketMessageResponse{}, nil
	}

	messageIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		messageIDs = append(messageIDs, row.ID)
	}

	attachmentItems, err := h.loadAttachmentResponseMap(c, attachmentOwnerTicketMessage, messageIDs)
	if err != nil {
		return nil, err
	}

	attachmentMap := make(map[string][]ticketMessageAttachmentResponse)
	for messageID, attachments := range attachmentItems {
		for _, attachment := range attachments {
			attachmentMap[messageID] = append(attachmentMap[messageID], ticketMessageAttachmentResponse{
				ID:          attachment.ID,
				FileName:    attachment.FileName,
				ContentType: attachment.ContentType,
				Size:        attachment.Size,
				DownloadURL: attachment.DownloadURL,
				CreatedAt:   attachment.CreatedAt,
			})
		}
	}

	out := make([]ticketMessageResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, ticketMessageResponse{
			ID:          row.ID,
			TicketID:    row.TicketID,
			SenderType:  row.SenderType,
			UserID:      row.UserID,
			UserEmail:   row.UserEmail,
			Content:     row.Content,
			CreatedAt:   row.CreatedAt,
			Attachments: attachmentMap[row.ID],
		})
	}
	return out, nil
}

func (h *Handler) loadTicketMessageByID(c *gin.Context, messageID string) (*ticketMessageResponse, error) {
	type ticketMessageRow struct {
		ID         string    `json:"id"`
		TicketID   string    `json:"ticket_id"`
		SenderType string    `json:"sender_type"`
		UserID     string    `json:"user_id"`
		UserEmail  string    `json:"user_email"`
		Content    string    `json:"content"`
		CreatedAt  time.Time `json:"created_at"`
	}

	var row ticketMessageRow
	if err := h.DB.Table("ticket_messages tm").
		Select("tm.id, tm.ticket_id, tm.sender_type, tm.user_id, tm.content, tm.created_at, u.email as user_email").
		Joins("LEFT JOIN users u ON u.id = tm.user_id").
		Where("tm.id = ?", messageID).
		Take(&row).Error; err != nil {
		return nil, err
	}

	attachments, err := h.loadAttachmentResponses(c, attachmentOwnerTicketMessage, messageID)
	if err != nil {
		return nil, err
	}

	outAttachments := make([]ticketMessageAttachmentResponse, 0, len(attachments))
	for _, attachment := range attachments {
		outAttachments = append(outAttachments, ticketMessageAttachmentResponse{
			ID:          attachment.ID,
			FileName:    attachment.FileName,
			ContentType: attachment.ContentType,
			Size:        attachment.Size,
			DownloadURL: attachment.DownloadURL,
			CreatedAt:   attachment.CreatedAt,
		})
	}

	return &ticketMessageResponse{
		ID:          row.ID,
		TicketID:    row.TicketID,
		SenderType:  row.SenderType,
		UserID:      row.UserID,
		UserEmail:   row.UserEmail,
		Content:     row.Content,
		CreatedAt:   row.CreatedAt,
		Attachments: outAttachments,
	}, nil
}
