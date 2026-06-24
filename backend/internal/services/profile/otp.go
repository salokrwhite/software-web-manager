package profile

import (
	"strings"

	"software-web-manager/backend/internal/crypto"
	"software-web-manager/backend/internal/models"
)

// SetupOTP generates and persists a new (disabled) OTP secret for the user,
// returning the secret and its otpauth provisioning URL.
func (s *Service) SetupOTP(userID string) (string, string, error) {
	var user models.User
	if err := s.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return "", "", ErrUserNotFound
	}
	if user.OTPEnabled {
		return "", "", ErrOTPAlreadyEnabled
	}
	secret, err := crypto.GenerateOTPSecret()
	if err != nil {
		return "", "", err
	}
	otpauthURL := crypto.BuildOTPAuthURL(user.Email, secret)
	if err := s.DB.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]any{
		"otp_secret":  secret,
		"otp_enabled": false,
	}).Error; err != nil {
		return "", "", err
	}
	return secret, otpauthURL, nil
}

// ConfirmOTP validates a TOTP code against the user's pending secret without
// enabling it.
func (s *Service) ConfirmOTP(userID, code string) error {
	var user models.User
	if err := s.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return ErrUserNotFound
	}
	if user.OTPEnabled {
		return ErrOTPAlreadyEnabled
	}
	secret := otpSecret(user)
	if secret == "" {
		return ErrOTPNotSetup
	}
	if !crypto.ValidateTOTP(secret, code) {
		return ErrInvalidOTP
	}
	return nil
}

// ToggleOTP enables or disables OTP enforcement for the user. Enabling requires
// a valid TOTP code.
func (s *Service) ToggleOTP(userID string, enable bool, code string) error {
	var user models.User
	if err := s.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return ErrUserNotFound
	}
	secret := otpSecret(user)
	if secret == "" {
		return ErrOTPNotSetup
	}
	if enable {
		if strings.TrimSpace(code) == "" {
			return ErrOTPRequired
		}
		if !crypto.ValidateTOTP(secret, code) {
			return ErrInvalidOTP
		}
	}
	return s.DB.Model(&models.User{}).Where("id = ?", userID).Update("otp_enabled", enable).Error
}

// DisableOTP verifies the user's password and TOTP code, then clears the OTP
// secret and disables OTP.
func (s *Service) DisableOTP(userID, currentPassword, code string) error {
	var user models.User
	if err := s.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return ErrUserNotFound
	}
	if !crypto.CheckPassword(user.PasswordHash, currentPassword) {
		return ErrCurrentPasswordIncorrect
	}
	secret := otpSecret(user)
	if secret == "" {
		return ErrOTPNotSetup
	}
	if !crypto.ValidateTOTP(secret, code) {
		return ErrInvalidOTP
	}
	return s.DB.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]any{
		"otp_enabled": false,
		"otp_secret":  nil,
	}).Error
}

func otpSecret(user models.User) string {
	if user.OTPSecret == nil {
		return ""
	}
	return strings.TrimSpace(*user.OTPSecret)
}
