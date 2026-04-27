package handlers

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"
	"software-web-manager/backend/internal/storage"
	"software-web-manager/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const maxSystemAvatarSize = 2 * 1024 * 1024

type updateSystemPasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

type systemProfileResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

func (h *Handler) ensureStorage(c *gin.Context) error {
	if h.Storage != nil {
		return nil
	}
	store, err := storage.New(context.Background(), h.Cfg)
	if err != nil && h.Cfg.StorageDriver != "local" {
		fallbackCfg := h.Cfg
		fallbackCfg.StorageDriver = "local"
		store, err = storage.New(context.Background(), fallbackCfg)
	}
	if err != nil {
		return err
	}
	h.Storage = store
	return nil
}

func localFileURL(c *gin.Context, storagePath string) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			scheme = strings.TrimSpace(parts[0])
		}
	}
	host := strings.TrimSpace(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(c.Request.Host)
	}
	if host == "" {
		host = strings.TrimSpace(c.Request.URL.Host)
	}
	return fmt.Sprintf("%s://%s/files/%s", scheme, host, strings.TrimLeft(storagePath, "/"))
}

func (h *Handler) GetSystemProfile(c *gin.Context) {
	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	var user models.User
	if err := h.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	avatarURL := ""
	if strings.TrimSpace(user.AvatarPath) != "" {
		if err := h.ensureStorage(c); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "storage not configured"})
			return
		}
		if strings.EqualFold(h.Cfg.StorageDriver, "local") {
			avatarURL = localFileURL(c, user.AvatarPath)
		} else {
			url, err := h.Storage.GetDownloadURL(c.Request.Context(), user.AvatarPath, 7*24*time.Hour)
			if err == nil {
				avatarURL = url
			}
		}
	}

	c.JSON(http.StatusOK, systemProfileResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		AvatarURL: avatarURL,
	})
}

func (h *Handler) UpdateSystemPassword(c *gin.Context) {
	var req updateSystemPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.NewPassword = strings.TrimSpace(req.NewPassword)
	if len(req.NewPassword) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password too short"})
		return
	}

	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	var user models.User
	if err := h.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if !utils.CheckPassword(user.PasswordHash, req.CurrentPassword) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "current password incorrect"})
		return
	}

	hash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	if err := h.DB.Model(&models.User{}).Where("id = ?", user.ID).Update("password_hash", hash).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) UpdateSystemAvatar(c *gin.Context) {
	if err := h.ensureStorage(c); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage not configured"})
		return
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "avatar required"})
		return
	}
	if file.Size <= 0 || file.Size > maxSystemAvatarSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "avatar too large"})
		return
	}

	contentType := strings.TrimSpace(file.Header.Get("Content-Type"))
	if idx := strings.Index(contentType, ";"); idx >= 0 {
		contentType = strings.TrimSpace(contentType[:idx])
	}
	if contentType == "" {
		contentType = mime.TypeByExtension(strings.ToLower(filepath.Ext(file.Filename)))
	}

	ext := ""
	switch strings.ToLower(contentType) {
	case "image/jpeg", "image/jpg":
		contentType = "image/jpeg"
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported image type"})
		return
	}

	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	if _, err := uuid.Parse(userID); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	handle, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read avatar"})
		return
	}
	defer handle.Close()

	key := filepath.ToSlash(filepath.Join("system", "users", userID, "avatar", uuid.New().String()+ext))
	storagePath, err := h.Storage.Save(c.Request.Context(), handle, file.Size, key, contentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store avatar"})
		return
	}

	if err := h.DB.Model(&models.User{}).Where("id = ?", userID).Update("avatar_path", storagePath).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update avatar"})
		return
	}

	url := ""
	if strings.EqualFold(h.Cfg.StorageDriver, "local") {
		url = localFileURL(c, storagePath)
	} else {
		downloadURL, err := h.Storage.GetDownloadURL(c.Request.Context(), storagePath, 7*24*time.Hour)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create avatar url"})
			return
		}
		url = downloadURL
	}

	c.JSON(http.StatusOK, gin.H{"avatar_url": url})
}

