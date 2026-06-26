// Package profile provides profile-domain data access (user lookup, password and
// 2FA management) that is independent of the HTTP layer (no gin, no response
// writing).
package profile

import (
	"errors"

	"software-web-manager/backend/internal/crypto"
	"software-web-manager/backend/internal/models"

	"gorm.io/gorm"
)

// Sentinel errors returned by profile operations so the HTTP layer can map them
// to the right status code.
var (
	ErrUserNotFound             = errors.New("user not found")
	ErrCurrentPasswordIncorrect = errors.New("current password incorrect")
	ErrOTPAlreadyEnabled        = errors.New("otp already enabled")
	ErrOTPNotSetup              = errors.New("otp not setup")
	ErrOTPRequired              = errors.New("otp required")
	ErrInvalidOTP               = errors.New("invalid otp")
)

// Service exposes profile queries and commands over a single database handle.
type Service struct {
	DB *gorm.DB
}

// NewService builds a profile service from a database handle.
func NewService(db *gorm.DB) *Service {
	return &Service{DB: db}
}

// GetUser loads a user by id, returning the underlying gorm error when missing.
func (s *Service) GetUser(userID string) (models.User, error) {
	var user models.User
	if err := s.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return user, err
	}
	return user, nil
}

// ChangePassword verifies the current password and stores a new hashed password.
func (s *Service) ChangePassword(userID, currentPassword, newPassword string) error {
	var user models.User
	if err := s.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return ErrUserNotFound
	}
	if !crypto.CheckPassword(user.PasswordHash, currentPassword) {
		return ErrCurrentPasswordIncorrect
	}
	hash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return err
	}
	// Bump token_version so all previously issued tokens are revoked on password change.
	return s.DB.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"password_hash": hash,
		"token_version": user.TokenVersion + 1,
	}).Error
}

// SetAvatarPath stores the user's avatar storage path.
func (s *Service) SetAvatarPath(userID, storagePath string) error {
	return s.DB.Model(&models.User{}).Where("id = ?", userID).Update("avatar_path", storagePath).Error
}
