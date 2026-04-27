package handlers

import (
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"
	"software-web-manager/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const maxOrgAvatarSize = 2 * 1024 * 1024

type profileResponse struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	AvatarURL  string `json:"avatar_url"`
	OTPEnabled bool   `json:"otp_enabled"`
	OTPBound   bool   `json:"otp_bound"`
}

type updateProfilePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

type confirmProfile2FARequest struct {
	OTPCode string `json:"otp_code" binding:"required"`
}

type toggleProfile2FARequest struct {
	Enable  *bool  `json:"enable" binding:"required"`
	OTPCode string `json:"otp_code"`
}

type disableProfile2FARequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	OTPCode         string `json:"otp_code" binding:"required"`
}

func requireOrgAdmin(c *gin.Context) bool {
	systemRole := strings.ToLower(strings.TrimSpace(c.GetString(middleware.ContextSystemRole)))
	if systemRole != "org_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return false
	}
	return true
}

func (h *Handler) requireProfileAccess(c *gin.Context) bool {
	return true
}

func (h *Handler) GetProfile(c *gin.Context) {
	if !h.requireProfileAccess(c) {
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

	avatarURL := ""
	if strings.TrimSpace(user.AvatarPath) != "" {
		if err := h.ensureStorage(c); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "storage not configured"})
			return
		}
		if strings.EqualFold(h.Cfg.StorageDriver, "local") {
			avatarURL = h.buildLocalFileURL(c, user.AvatarPath, 7*24*time.Hour)
		} else {
			url, err := h.Storage.GetDownloadURL(c.Request.Context(), user.AvatarPath, 7*24*time.Hour)
			if err == nil {
				avatarURL = url
			}
		}
	}
	otpBound := false
	if user.OTPSecret != nil && strings.TrimSpace(*user.OTPSecret) != "" {
		otpBound = true
	}

	c.JSON(http.StatusOK, profileResponse{
		ID:         user.ID.String(),
		Email:      user.Email,
		AvatarURL:  avatarURL,
		OTPEnabled: user.OTPEnabled,
		OTPBound:   otpBound,
	})
}

func (h *Handler) UpdateProfilePassword(c *gin.Context) {
	if !h.requireProfileAccess(c) {
		return
	}
	var req updateProfilePasswordRequest
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

func (h *Handler) UpdateProfileAvatar(c *gin.Context) {
	if !h.requireProfileAccess(c) {
		return
	}
	if err := h.ensureStorage(c); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage not configured"})
		return
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "avatar required"})
		return
	}
	if file.Size <= 0 || file.Size > maxOrgAvatarSize {
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

	orgID := strings.TrimSpace(c.GetString(middleware.ContextOrgID))
	basePath := filepath.Join("users", userID, "avatar")
	if orgID != "" {
		basePath = filepath.Join("orgs", orgID, "users", userID, "avatar")
	}
	key := filepath.ToSlash(filepath.Join(basePath, uuid.New().String()+ext))
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
		url = h.buildLocalFileURL(c, storagePath, 7*24*time.Hour)
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

func (h *Handler) SetupProfile2FA(c *gin.Context) {
	if !requireOrgAdmin(c) {
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
	if user.OTPEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "otp already enabled"})
		return
	}
	secret, err := generateOTPSecret()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate otp"})
		return
	}
	otpauthURL := buildOTPAuthURL(user.Email, secret)
	if err := h.DB.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]any{
		"otp_secret":  secret,
		"otp_enabled": false,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store otp"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"secret": secret, "otpauth_url": otpauthURL})
}

func (h *Handler) ConfirmProfile2FA(c *gin.Context) {
	if !requireOrgAdmin(c) {
		return
	}
	var req confirmProfile2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	if user.OTPEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "otp already enabled"})
		return
	}
	secret := ""
	if user.OTPSecret != nil {
		secret = strings.TrimSpace(*user.OTPSecret)
	}
	if secret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "otp not setup"})
		return
	}
	if !validateTOTP(secret, req.OTPCode) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid otp"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) ToggleProfile2FA(c *gin.Context) {
	if !requireOrgAdmin(c) {
		return
	}
	var req toggleProfile2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Enable == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "enable required"})
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
	secret := ""
	if user.OTPSecret != nil {
		secret = strings.TrimSpace(*user.OTPSecret)
	}
	if secret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "otp not setup"})
		return
	}

	enable := *req.Enable
	if enable {
		if strings.TrimSpace(req.OTPCode) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "otp required"})
			return
		}
		if !validateTOTP(secret, req.OTPCode) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid otp"})
			return
		}
	}
	if err := h.DB.Model(&models.User{}).Where("id = ?", userID).Update("otp_enabled", enable).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update otp"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) DisableProfile2FA(c *gin.Context) {
	if !requireOrgAdmin(c) {
		return
	}
	var req disableProfile2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	secret := ""
	if user.OTPSecret != nil {
		secret = strings.TrimSpace(*user.OTPSecret)
	}
	if secret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "otp not setup"})
		return
	}
	if !validateTOTP(secret, req.OTPCode) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid otp"})
		return
	}
	if err := h.DB.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]any{
		"otp_enabled": false,
		"otp_secret":  nil,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable otp"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
