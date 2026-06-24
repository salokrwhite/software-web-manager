package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"software-web-manager/backend/internal/db/schema"
	"strconv"
	"strings"
	"time"

	"software-web-manager/backend/internal/models"
	"software-web-manager/backend/internal/services/system"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	registerEmailCodePurpose         = "register"
	registerEmailCodeLength          = 6
	registerEmailCodeExpiresMinutes  = 10
	registerEmailCodeCooldownSeconds = 60
)

var (
	errRegisterEmailCodeRequired    = errors.New("email_code_required")
	errRegisterEmailCodeInvalid     = errors.New("email_code_invalid")
	errRegisterEmailCodeExpired     = errors.New("email_code_expired")
	errRegisterEmailCodeTooFrequent = errors.New("email_code_send_too_frequent")
	errRegisterEmailNotConfigured   = errors.New("register_email_not_configured")
)

type sendRegisterEmailCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func generateNumericCode(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("invalid code length")
	}
	digits := make([]byte, length)
	max := big.NewInt(10)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		digits[i] = byte('0' + n.Int64())
	}
	return string(digits), nil
}

func hashRegisterEmailCode(secret, email, code string) string {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	normalizedCode := strings.TrimSpace(code)
	base := normalizedEmail + "|" + registerEmailCodePurpose + "|" + normalizedCode + "|" + strings.TrimSpace(secret)
	sum := sha256.Sum256([]byte(base))
	return hex.EncodeToString(sum[:])
}

func renderRegisterEmailCodeTemplate(content, siteName, code string, expiresMinutes int) string {
	template := strings.TrimSpace(content)
	if template == "" {
		template = system.DefaultRegisterEmailCodeTemplate
	}
	if strings.TrimSpace(siteName) == "" {
		siteName = system.DefaultSiteName
	}
	replacer := strings.NewReplacer(
		"{{code}}", code,
		"{{minutes}}", strconv.Itoa(expiresMinutes),
		"{{expire_minutes}}", strconv.Itoa(expiresMinutes),
		"{{site_name}}", siteName,
	)
	return replacer.Replace(template)
}

func (h *Handler) ensureEmailVerificationCodesTable() error {
	if schema.HasEmailVerificationCodesTable(h.DB) {
		return nil
	}
	return h.DB.AutoMigrate(&models.EmailVerificationCode{})
}

func (h *Handler) getRegisterEmailContext() (system.SMTPConfig, string, string, error) {
	if !schema.HasSystemSettingsTable(h.DB) {
		return system.SMTPConfig{}, "", "", errRegisterEmailNotConfigured
	}
	items, err := system.NewService(h.DB).ListSettings()
	if err != nil {
		return system.SMTPConfig{}, "", "", err
	}
	cfg := system.NewService(h.DB).SMTPConfigFromSettings(items)
	password, configured, passwordErr := system.NewService(h.DB).SMTPPasswordFromSettings(items)
	if passwordErr != nil {
		return system.SMTPConfig{}, "", "", passwordErr
	}
	cfg.Password = password
	if err := system.ValidateSMTPConfig(cfg, !configured); err != nil {
		return system.SMTPConfig{}, "", "", err
	}
	siteName := system.GetString(items, system.SettingSiteNameKey, system.DefaultSiteName)
	template := system.GetString(items, system.SettingRegisterEmailCodeTemplateKey, system.DefaultRegisterEmailCodeTemplate)
	return cfg, siteName, template, nil
}

func isSMTPConfigValidationError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.TrimSpace(err.Error())
	return strings.HasPrefix(msg, "smtp_")
}

func (h *Handler) SendRegisterEmailCode(c *gin.Context) {
	allowRegister, err := system.NewService(h.DB).AllowUserRegister()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load system settings"})
		return
	}
	if !allowRegister {
		c.JSON(http.StatusForbidden, gin.H{"error": "user_register_disabled"})
		return
	}

	var req sendRegisterEmailCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))

	var existingCount int64
	if err := h.DB.Model(&models.User{}).Where("email = ?", email).Count(&existingCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query user"})
		return
	}
	if existingCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	if err := h.ensureEmailVerificationCodesTable(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initialize email verification table"})
		return
	}

	cfg, siteName, template, err := h.getRegisterEmailContext()
	if err != nil {
		if errors.Is(err, errRegisterEmailNotConfigured) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "register_email_not_configured"})
			return
		}
		if isSMTPConfigValidationError(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load system settings"})
		return
	}

	code, err := generateNumericCode(registerEmailCodeLength)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate email code"})
		return
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(registerEmailCodeExpiresMinutes) * time.Minute)
	codeHash := hashRegisterEmailCode(h.Cfg.JWTSecret, email, code)
	requestIP := strings.TrimSpace(c.ClientIP())
	retryAfterSeconds := 0
	recordID := ""

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		var latest models.EmailVerificationCode
		err := tx.
			Where("email = ? AND purpose = ?", email, registerEmailCodePurpose).
			Order("created_at desc").
			First(&latest).
			Error
		if err == nil {
			elapsed := now.Sub(latest.CreatedAt)
			cooldown := time.Duration(registerEmailCodeCooldownSeconds) * time.Second
			if elapsed < cooldown {
				retryAfterSeconds = int((cooldown - elapsed).Seconds())
				if retryAfterSeconds < 1 {
					retryAfterSeconds = 1
				}
				return errRegisterEmailCodeTooFrequent
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := tx.Model(&models.EmailVerificationCode{}).
			Where("email = ? AND purpose = ? AND used_at IS NULL", email, registerEmailCodePurpose).
			Update("used_at", now).
			Error; err != nil {
			return err
		}

		item := models.EmailVerificationCode{
			Email:     email,
			Purpose:   registerEmailCodePurpose,
			CodeHash:  codeHash,
			ExpiresAt: expiresAt,
			RequestIP: requestIP,
		}
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		recordID = item.ID.String()
		return nil
	})
	if err != nil {
		if errors.Is(err, errRegisterEmailCodeTooFrequent) {
			if retryAfterSeconds < 1 {
				retryAfterSeconds = registerEmailCodeCooldownSeconds
			}
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":               "email_code_send_too_frequent",
				"retry_after_seconds": retryAfterSeconds,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create email verification code"})
		return
	}

	subjectSiteName := strings.TrimSpace(siteName)
	if subjectSiteName == "" {
		subjectSiteName = system.DefaultSiteName
	}
	subject := fmt.Sprintf("%s 注册验证码", subjectSiteName)
	body := renderRegisterEmailCodeTemplate(template, siteName, code, registerEmailCodeExpiresMinutes)
	if err := system.SendMail(cfg, email, subject, body); err != nil {
		if strings.TrimSpace(recordID) != "" {
			_ = h.DB.Model(&models.EmailVerificationCode{}).
				Where("id = ? AND used_at IS NULL", recordID).
				Update("used_at", time.Now()).
				Error
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "register_email_send_failed",
			"detail": system.SanitizeSMTPError(err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":                 true,
		"expires_in_seconds": registerEmailCodeExpiresMinutes * 60,
	})
}

func (h *Handler) consumeRegisterEmailCode(tx *gorm.DB, email, code string) error {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	normalizedCode := strings.TrimSpace(code)
	if normalizedCode == "" {
		return errRegisterEmailCodeRequired
	}

	var latest models.EmailVerificationCode
	err := tx.
		Where("email = ? AND purpose = ?", normalizedEmail, registerEmailCodePurpose).
		Order("created_at desc").
		First(&latest).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errRegisterEmailCodeExpired
		}
		return err
	}

	now := time.Now()
	if latest.UsedAt != nil || latest.ExpiresAt.Before(now) {
		return errRegisterEmailCodeExpired
	}

	expectedHash := hashRegisterEmailCode(h.Cfg.JWTSecret, normalizedEmail, normalizedCode)
	if subtle.ConstantTimeCompare([]byte(latest.CodeHash), []byte(expectedHash)) != 1 {
		return errRegisterEmailCodeInvalid
	}

	update := tx.Model(&models.EmailVerificationCode{}).
		Where("id = ? AND used_at IS NULL", latest.ID).
		Update("used_at", now)
	if update.Error != nil {
		return update.Error
	}
	if update.RowsAffected == 0 {
		return errRegisterEmailCodeExpired
	}
	return nil
}
