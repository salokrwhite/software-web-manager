package common

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"software-web-manager/backend/internal/config"
	"software-web-manager/backend/internal/models"
	"software-web-manager/backend/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AttachmentResponse is the API representation of a stored attachment.
type AttachmentResponse struct {
	ID          string    `json:"id"`
	FileName    string    `json:"file_name"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	DownloadURL string    `json:"download_url"`
	CreatedAt   time.Time `json:"created_at"`
}

// StoreAttachments saves multipart-uploaded files to storage and returns the
// attachment records (not yet persisted) plus an HTTP status on error. The
// caller is responsible for ensuring storage is initialized before calling.
func StoreAttachments(
	store storage.Driver,
	storageDriver string,
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
			storagePath, err := store.Save(c.Request.Context(), handle, file.Size, key, contentType)
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
				StorageDriver: storageDriver,
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

// LoadAttachmentResponses loads and renders all attachments for an owner.
func LoadAttachmentResponses(db *gorm.DB, store storage.Driver, cfg config.Config, c *gin.Context, ownerType string, ownerID string) ([]AttachmentResponse, error) {
	var attachments []models.Attachment
	if err := db.Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).Order("created_at asc").Find(&attachments).Error; err != nil {
		return nil, err
	}
	return BuildAttachmentResponses(store, cfg, c, attachments)
}

// LoadAttachmentResponseMap loads attachments for many owners, keyed by owner id.
func LoadAttachmentResponseMap(db *gorm.DB, store storage.Driver, cfg config.Config, c *gin.Context, ownerType string, ownerIDs []string) (map[string][]AttachmentResponse, error) {
	out := make(map[string][]AttachmentResponse, len(ownerIDs))
	if len(ownerIDs) == 0 {
		return out, nil
	}
	var attachments []models.Attachment
	if err := db.Where("owner_type = ? AND owner_id IN ?", ownerType, ownerIDs).Order("created_at asc").Find(&attachments).Error; err != nil {
		return nil, err
	}
	responses, err := BuildAttachmentResponses(store, cfg, c, attachments)
	if err != nil {
		return nil, err
	}
	for i := range attachments {
		out[attachments[i].OwnerID.String()] = append(out[attachments[i].OwnerID.String()], responses[i])
	}
	return out, nil
}

// BuildAttachmentResponses renders attachment records into API responses with
// signed download URLs. The caller is responsible for ensuring storage is
// initialized before calling.
func BuildAttachmentResponses(store storage.Driver, cfg config.Config, c *gin.Context, attachments []models.Attachment) ([]AttachmentResponse, error) {
	if len(attachments) == 0 {
		return []AttachmentResponse{}, nil
	}
	out := make([]AttachmentResponse, 0, len(attachments))
	for _, attachment := range attachments {
		url := ""
		if strings.EqualFold(cfg.StorageDriver, "local") {
			url = BuildLocalFileURL(cfg, c, attachment.StoragePath, 24*time.Hour)
		} else if store != nil {
			if downloadURL, err := store.GetDownloadURL(c.Request.Context(), attachment.StoragePath, 24*time.Hour); err == nil {
				url = downloadURL
			}
		}
		out = append(out, AttachmentResponse{
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
