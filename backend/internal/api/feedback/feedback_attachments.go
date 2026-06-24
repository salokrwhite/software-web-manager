package feedback

import (
	"time"

	"software-web-manager/backend/internal/handlers"
	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type feedbackAttachmentResponse struct {
	ID          string    `json:"id"`
	FileName    string    `json:"file_name"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	DownloadURL string    `json:"download_url"`
	CreatedAt   time.Time `json:"created_at"`
}

func (h *Handler) storeFeedbackAttachments(c *gin.Context, feedbackID uuid.UUID, orgID uuid.UUID) ([]models.Attachment, int, error) {
	return h.StoreAttachments(c, handlers.AttachmentOwnerFeedback, feedbackID, &orgID, nil, "attachments", "feedbacks", maxFeedbackAttachments, maxFeedbackAttachmentSize)
}

func (h *Handler) loadFeedbackAttachments(c *gin.Context, feedbackID string) ([]feedbackAttachmentResponse, error) {
	items, err := h.LoadAttachmentResponses(c, handlers.AttachmentOwnerFeedback, feedbackID)
	if err != nil {
		return nil, err
	}
	out := make([]feedbackAttachmentResponse, 0, len(items))
	for _, attachment := range items {
		out = append(out, feedbackAttachmentResponse{
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
