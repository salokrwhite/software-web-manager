package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"
	"time"

	"software-web-manager/backend/internal/db/schema"
	"software-web-manager/backend/internal/models"
	systemsvc "software-web-manager/backend/internal/services/system"

	"gorm.io/gorm"
)

const (
	registerEmailCodePurpose = "register"
	registerEmailCodeLength  = 6
	// RegisterEmailCodeExpiresMinutes is how long a register code stays valid.
	RegisterEmailCodeExpiresMinutes = 10
	// RegisterEmailCodeCooldownSeconds is the minimum gap between code sends.
	RegisterEmailCodeCooldownSeconds = 60
)

// Sentinel errors for register email codes. ErrEmailCodeTooFrequent and
// ErrEmailNotConfigured are mapped by the HTTP layer; the rest are mapped to
// 400 responses by RegisterUser.
var (
	errRegisterEmailCodeRequired = errors.New("email_code_required")
	errRegisterEmailCodeInvalid  = errors.New("email_code_invalid")
	errRegisterEmailCodeExpired  = errors.New("email_code_expired")
	// ErrEmailCodeTooFrequent indicates the send cooldown has not elapsed.
	ErrEmailCodeTooFrequent = errors.New("email_code_send_too_frequent")
	// ErrEmailNotConfigured indicates registration email is not set up.
	ErrEmailNotConfigured = errors.New("register_email_not_configured")
)

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

// IsSMTPConfigValidationError reports whether the error is an SMTP config
// validation failure (its message is suitable for direct display).
func IsSMTPConfigValidationError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.TrimSpace(err.Error())
	return strings.HasPrefix(msg, "smtp_")
}

// EmailRegistered reports whether a user with the (normalized) email exists.
func (s *Service) EmailRegistered(email string) (bool, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var existingCount int64
	if err := s.DB.Model(&models.User{}).Where("email = ?", email).Count(&existingCount).Error; err != nil {
		return false, err
	}
	return existingCount > 0, nil
}

// EnsureEmailVerificationTable creates the email verification codes table if it
// does not exist.
func (s *Service) EnsureEmailVerificationTable() error {
	if schema.HasEmailVerificationCodesTable(s.DB) {
		return nil
	}
	return s.DB.AutoMigrate(&models.EmailVerificationCode{})
}

// RegisterEmailContext loads and validates the SMTP config and template/site
// settings used to send a register code. It returns ErrEmailNotConfigured when
// system settings are unavailable.
func (s *Service) RegisterEmailContext() (systemsvc.SMTPConfig, string, string, error) {
	if !schema.HasSystemSettingsTable(s.DB) {
		return systemsvc.SMTPConfig{}, "", "", ErrEmailNotConfigured
	}
	items, err := systemsvc.NewService(s.DB).ListSettings()
	if err != nil {
		return systemsvc.SMTPConfig{}, "", "", err
	}
	cfg := systemsvc.NewService(s.DB).SMTPConfigFromSettings(items)
	password, configured, passwordErr := systemsvc.NewService(s.DB).SMTPPasswordFromSettings(items)
	if passwordErr != nil {
		return systemsvc.SMTPConfig{}, "", "", passwordErr
	}
	cfg.Password = password
	if err := systemsvc.ValidateSMTPConfig(cfg, !configured); err != nil {
		return systemsvc.SMTPConfig{}, "", "", err
	}
	siteName := systemsvc.GetString(items, systemsvc.SettingSiteNameKey, systemsvc.DefaultSiteName)
	template := systemsvc.GetString(items, systemsvc.SettingRegisterEmailCodeTemplateKey, systemsvc.DefaultRegisterEmailCodeTemplate)
	return cfg, siteName, template, nil
}

// CreateRegisterCode generates a register code, enforces the send cooldown,
// invalidates prior codes, and persists the new one. It returns the plaintext
// code and the new record id. On cooldown it returns ErrEmailCodeTooFrequent
// with retryAfterSeconds set.
func (s *Service) CreateRegisterCode(email, requestIP string) (code string, recordID string, retryAfterSeconds int, err error) {
	email = strings.ToLower(strings.TrimSpace(email))
	code, err = generateNumericCode(registerEmailCodeLength)
	if err != nil {
		return "", "", 0, err
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(RegisterEmailCodeExpiresMinutes) * time.Minute)
	codeHash := hashRegisterEmailCode(s.Cfg.JWTSecret, email, code)

	err = s.DB.Transaction(func(tx *gorm.DB) error {
		var latest models.EmailVerificationCode
		err := tx.
			Where("email = ? AND purpose = ?", email, registerEmailCodePurpose).
			Order("created_at desc").
			First(&latest).
			Error
		if err == nil {
			elapsed := now.Sub(latest.CreatedAt)
			cooldown := time.Duration(RegisterEmailCodeCooldownSeconds) * time.Second
			if elapsed < cooldown {
				retryAfterSeconds = int((cooldown - elapsed).Seconds())
				if retryAfterSeconds < 1 {
					retryAfterSeconds = 1
				}
				return ErrEmailCodeTooFrequent
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
		return "", "", retryAfterSeconds, err
	}
	return code, recordID, 0, nil
}

// InvalidateCode marks an email verification code record as used, best-effort.
func (s *Service) InvalidateCode(recordID string) {
	if strings.TrimSpace(recordID) == "" {
		return
	}
	_ = s.DB.Model(&models.EmailVerificationCode{}).
		Where("id = ? AND used_at IS NULL", recordID).
		Update("used_at", time.Now()).
		Error
}

func (s *Service) consumeRegisterEmailCode(tx *gorm.DB, email, code string) error {
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

	expectedHash := hashRegisterEmailCode(s.Cfg.JWTSecret, normalizedEmail, normalizedCode)
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
