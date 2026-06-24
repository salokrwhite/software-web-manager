package auth

import (
	"errors"
	"strings"

	"software-web-manager/backend/internal/crypto"
	"software-web-manager/backend/internal/models"
	systemsvc "software-web-manager/backend/internal/services/system"

	"gorm.io/gorm"
)

// EnsureRegistrationAllowed verifies open user registration is enabled.
func (s *Service) EnsureRegistrationAllowed() error {
	allowRegister, err := systemsvc.NewService(s.DB).AllowUserRegister()
	if err != nil {
		return newError(500, "failed to load system settings")
	}
	if !allowRegister {
		return newError(403, "user_register_disabled")
	}
	return nil
}

// RegisterUser creates a new end-user account, consuming the register email
// verification code in the same transaction.
func (s *Service) RegisterUser(email, password, emailCode string) (models.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	registered, err := s.EmailRegistered(email)
	if err != nil {
		return models.User{}, newError(500, "failed to query user")
	}
	if registered {
		return models.User{}, newError(409, "email already registered")
	}
	if err := s.EnsureEmailVerificationTable(); err != nil {
		return models.User{}, newError(500, "failed to initialize email verification table")
	}

	hash, err := crypto.HashPassword(password)
	if err != nil {
		return models.User{}, newError(500, "failed to hash password")
	}

	var user models.User
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if err := s.consumeRegisterEmailCode(tx, email, emailCode); err != nil {
			return err
		}
		user = models.User{
			Email:        email,
			PasswordHash: hash,
			Status:       "active",
			SystemRole:   "none",
		}
		return tx.Create(&user).Error
	})
	if err != nil {
		switch {
		case errors.Is(err, errRegisterEmailCodeRequired):
			return models.User{}, newError(400, "email_code_required")
		case errors.Is(err, errRegisterEmailCodeInvalid):
			return models.User{}, newError(400, "email_code_invalid")
		case errors.Is(err, errRegisterEmailCodeExpired):
			return models.User{}, newError(400, "email_code_expired")
		}
		return models.User{}, newError(500, "failed to register")
	}
	return user, nil
}
