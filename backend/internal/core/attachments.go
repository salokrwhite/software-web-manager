package core

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"software-web-manager/backend/internal/models"
	"software-web-manager/backend/internal/services/attachment"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Attachment owner-type aliases; the canonical values live in services/attachment.
const (
	AttachmentOwnerTicket                  = attachment.OwnerTicket
	AttachmentOwnerTicketMessage           = attachment.OwnerTicketMessage
	AttachmentOwnerFeedback                = attachment.OwnerFeedback
	AttachmentOwnerOrgRegistrationMaterial = attachment.OwnerOrgRegistrationMaterial
)

type attachmentResponse struct {
	ID          string    `json:"id"`
	FileName    string    `json:"file_name"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	DownloadURL string    `json:"download_url"`
	CreatedAt   time.Time `json:"created_at"`
}

func (h *Handler) StoreAttachments(
	c *gin.Context,
	ownerType string,
	ownerID uuid.UUID,
	orgID *uuid.UUID,
	createdBy *uuid.UUID,
	formField string,
	storagePrefix string,
	maxFiles int,
	maxSize int64,
) ([]models.Attachment, int, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("failed to parse form")
	}
	files := form.File[formField]
	if len(files) == 0 {
		return nil, 0, nil
	}
	if len(files) > maxFiles {
		return nil, http.StatusBadRequest, errors.New("too many attachments")
	}

	attachments := make([]models.Attachment, 0, len(files))
	for _, file := range files {
		if file.Size > maxSize {
			return nil, http.StatusBadRequest, errors.New("attachment too large")
		}
		handle, err := file.Open()
		if err != nil {
			return nil, http.StatusBadRequest, errors.New("failed to read attachment")
		}
		func() {
			defer handle.Close()
			attachmentID := uuid.New()
			fileName := strings.TrimSpace(filepath.Base(file.Filename))
			if fileName == "" || fileName == "." {
				fileName = attachmentID.String()
			}
			key := filepath.ToSlash(filepath.Join(storagePrefix, ownerID.String(), attachmentID.String(), fileName))
			contentType := file.Header.Get("Content-Type")
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			storagePath, err := h.Storage.Save(c.Request.Context(), handle, file.Size, key, contentType)
			if err != nil {
				attachments = nil
				return
			}
			attachments = append(attachments, models.Attachment{
				ID:            attachmentID,
				OwnerType:     ownerType,
				OwnerID:       ownerID,
				OrgID:         orgID,
				FileName:      fileName,
				ContentType:   contentType,
				Size:          file.Size,
				StorageDriver: h.Cfg.StorageDriver,
				StoragePath:   storagePath,
				CreatedBy:     createdBy,
			})
		}()
		if attachments == nil {
			return nil, http.StatusInternalServerError, errors.New("failed to store attachment")
		}
	}
	return attachments, 0, nil
}

func (h *Handler) LoadAttachmentResponses(c *gin.Context, ownerType string, ownerID string) ([]attachmentResponse, error) {
	var attachments []models.Attachment
	if err := h.DB.Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).Order("created_at asc").Find(&attachments).Error; err != nil {
		return nil, err
	}
	return h.BuildAttachmentResponses(c, attachments)
}

func (h *Handler) LoadAttachmentResponseMap(c *gin.Context, ownerType string, ownerIDs []string) (map[string][]attachmentResponse, error) {
	out := make(map[string][]attachmentResponse, len(ownerIDs))
	if len(ownerIDs) == 0 {
		return out, nil
	}
	var attachments []models.Attachment
	if err := h.DB.Where("owner_type = ? AND owner_id IN ?", ownerType, ownerIDs).Order("created_at asc").Find(&attachments).Error; err != nil {
		return nil, err
	}
	responses, err := h.BuildAttachmentResponses(c, attachments)
	if err != nil {
		return nil, err
	}
	for i := range attachments {
		out[attachments[i].OwnerID.String()] = append(out[attachments[i].OwnerID.String()], responses[i])
	}
	return out, nil
}

func (h *Handler) BuildAttachmentResponses(c *gin.Context, attachments []models.Attachment) ([]attachmentResponse, error) {
	if len(attachments) == 0 {
		return []attachmentResponse{}, nil
	}
	if err := h.EnsureStorage(c); err != nil {
		return nil, err
	}
	out := make([]attachmentResponse, 0, len(attachments))
	for _, attachment := range attachments {
		url := ""
		if strings.EqualFold(h.Cfg.StorageDriver, "local") {
			url = h.BuildLocalFileURL(c, attachment.StoragePath, 24*time.Hour)
		} else if h.Storage != nil {
			if downloadURL, err := h.Storage.GetDownloadURL(c.Request.Context(), attachment.StoragePath, 24*time.Hour); err == nil {
				url = downloadURL
			}
		}
		out = append(out, attachmentResponse{
			ID:          attachment.ID.String(),
			FileName:    attachment.FileName,
			ContentType: attachment.ContentType,
			Size:        attachment.Size,
			DownloadURL: url,
			CreatedAt:   attachment.CreatedAt,
		})
	}
	return out, nil
}

// LoadAttachmentStoragePaths delegates to services/attachment.
func LoadAttachmentStoragePaths(tx *gorm.DB, ownerType string, ownerIDs []string) ([]string, error) {
	return attachment.LoadStoragePaths(tx, ownerType, ownerIDs)
}

// DeleteAttachmentsByOwners delegates to services/attachment.
func DeleteAttachmentsByOwners(tx *gorm.DB, ownerType string, ownerIDs []string) error {
	return attachment.DeleteByOwners(tx, ownerType, ownerIDs)
}
