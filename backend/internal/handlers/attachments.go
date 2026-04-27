package handlers

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	attachmentOwnerTicket                  = "ticket"
	attachmentOwnerTicketMessage           = "ticket_message"
	attachmentOwnerFeedback                = "feedback"
	attachmentOwnerOrgRegistrationMaterial = "org_registration_material"
)

type attachmentResponse struct {
	ID          string    `json:"id"`
	FileName    string    `json:"file_name"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	DownloadURL string    `json:"download_url"`
	CreatedAt   time.Time `json:"created_at"`
}

func (h *Handler) storeAttachments(
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

func (h *Handler) loadAttachmentResponses(c *gin.Context, ownerType string, ownerID string) ([]attachmentResponse, error) {
	var attachments []models.Attachment
	if err := h.DB.Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).Order("created_at asc").Find(&attachments).Error; err != nil {
		return nil, err
	}
	return h.buildAttachmentResponses(c, attachments)
}

func (h *Handler) loadAttachmentResponseMap(c *gin.Context, ownerType string, ownerIDs []string) (map[string][]attachmentResponse, error) {
	out := make(map[string][]attachmentResponse, len(ownerIDs))
	if len(ownerIDs) == 0 {
		return out, nil
	}
	var attachments []models.Attachment
	if err := h.DB.Where("owner_type = ? AND owner_id IN ?", ownerType, ownerIDs).Order("created_at asc").Find(&attachments).Error; err != nil {
		return nil, err
	}
	responses, err := h.buildAttachmentResponses(c, attachments)
	if err != nil {
		return nil, err
	}
	for i := range attachments {
		out[attachments[i].OwnerID.String()] = append(out[attachments[i].OwnerID.String()], responses[i])
	}
	return out, nil
}

func (h *Handler) buildAttachmentResponses(c *gin.Context, attachments []models.Attachment) ([]attachmentResponse, error) {
	if len(attachments) == 0 {
		return []attachmentResponse{}, nil
	}
	if err := h.ensureStorage(c); err != nil {
		return nil, err
	}
	out := make([]attachmentResponse, 0, len(attachments))
	for _, attachment := range attachments {
		url := ""
		if strings.EqualFold(h.Cfg.StorageDriver, "local") {
			url = h.buildLocalFileURL(c, attachment.StoragePath, 24*time.Hour)
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

func loadAttachmentStoragePaths(tx *gorm.DB, ownerType string, ownerIDs []string) ([]string, error) {
	if len(ownerIDs) == 0 {
		return nil, nil
	}
	var paths []string
	err := tx.Model(&models.Attachment{}).
		Where("owner_type = ? AND owner_id IN ?", ownerType, ownerIDs).
		Pluck("storage_path", &paths).Error
	return paths, err
}

func deleteAttachmentsByOwners(tx *gorm.DB, ownerType string, ownerIDs []string) error {
	if len(ownerIDs) == 0 {
		return nil
	}
	return tx.Where("owner_type = ? AND owner_id IN ?", ownerType, ownerIDs).Delete(&models.Attachment{}).Error
}
