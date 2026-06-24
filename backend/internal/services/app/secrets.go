package app

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"software-web-manager/backend/internal/crypto"
	"software-web-manager/backend/internal/db/schema"
	"software-web-manager/backend/internal/models"

	"gorm.io/gorm"
)

// GenerateAppSecret returns a new random hex app secret.
func GenerateAppSecret() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

// IsAppSecretColumnMissingErr reports whether the error indicates the
// app_secret_ciphertext column is absent (migration not yet applied).
func IsAppSecretColumnMissingErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unknown column") && strings.Contains(msg, "app_secret_ciphertext")
}

// EnsureAppSecret returns the active plaintext app secret for the app,
// generating and persisting one if necessary. It transparently handles both the
// app_secrets table and the legacy app_secret_ciphertext column layout.
func (s *Service) EnsureAppSecret(appID string, masterKey string) (string, error) {
	var app models.App
	if err := s.DB.Where("id = ?", appID).First(&app).Error; err != nil {
		if IsAppSecretColumnMissingErr(err) {
			return "", fmt.Errorf("missing app_secret_ciphertext column, run migration 0029_app_secret_and_signature")
		}
		return "", err
	}

	if schema.HasAppSecretsTable(s.DB) {
		var current models.AppSecret
		err := s.DB.
			Where("app_id = ? AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > ?)", app.ID, time.Now()).
			Order("created_at asc").
			First(&current).Error
		if err == nil {
			return crypto.DecryptAppSecret(masterKey, current.SecretCiphertext)
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}

		if strings.TrimSpace(app.AppSecretCiphertext) != "" {
			row := models.AppSecret{
				AppID:            app.ID,
				Name:             app.AppSecretName,
				SecretCiphertext: app.AppSecretCiphertext,
				ScopesJSON:       app.AppSecretScopesJSON,
				ExpiresAt:        app.AppSecretExpiresAt,
			}
			if strings.TrimSpace(row.Name) == "" {
				row.Name = "app_secret"
			}
			if len(row.ScopesJSON) == 0 {
				row.ScopesJSON = AppSecretScopesJSON(nil)
			}
			if err := s.DB.Create(&row).Error; err != nil {
				return "", err
			}
			return crypto.DecryptAppSecret(masterKey, row.SecretCiphertext)
		}

		secret, err := GenerateAppSecret()
		if err != nil {
			return "", err
		}
		secretCipher, err := crypto.EncryptAppSecret(masterKey, secret)
		if err != nil {
			return "", err
		}
		row := models.AppSecret{
			AppID:            app.ID,
			Name:             "app_secret",
			SecretCiphertext: secretCipher,
			ScopesJSON:       AppSecretScopesJSON(nil),
		}
		if err := s.DB.Create(&row).Error; err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				var created models.AppSecret
				if loadErr := s.DB.
					Where("app_id = ? AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > ?)", app.ID, time.Now()).
					Order("created_at asc").
					First(&created).Error; loadErr == nil {
					return crypto.DecryptAppSecret(masterKey, created.SecretCiphertext)
				}
			}
			return "", err
		}
		return secret, nil
	}

	if strings.TrimSpace(app.AppSecretCiphertext) != "" {
		return crypto.DecryptAppSecret(masterKey, app.AppSecretCiphertext)
	}

	secret, err := GenerateAppSecret()
	if err != nil {
		return "", err
	}
	secretCipher, err := crypto.EncryptAppSecret(masterKey, secret)
	if err != nil {
		return "", err
	}
	now := time.Now()
	txErr := s.DB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"app_secret_ciphertext": secretCipher,
			"app_secret_updated_at": now,
		}
		if schema.HasAppSecretScopesColumn(s.DB) {
			updates["app_secret_scopes"] = AppSecretScopesJSON(nil)
		}
		if schema.HasAppSecretExpiresAtColumn(s.DB) {
			updates["app_secret_expires_at"] = nil
		}
		if schema.HasAppSecretNameColumn(s.DB) {
			updates["app_secret_name"] = "app_secret"
		}
		res := tx.Model(&models.App{}).
			Where("id = ? AND (app_secret_ciphertext = '' OR app_secret_ciphertext IS NULL)", app.ID).
			Updates(updates)
		if res.Error != nil {
			if IsAppSecretColumnMissingErr(res.Error) {
				return fmt.Errorf("missing app_secret_ciphertext column, run migration 0029_app_secret_and_signature")
			}
			return res.Error
		}
		if res.RowsAffected > 0 {
			return nil
		}
		var current models.App
		if err := tx.Where("id = ?", app.ID).First(&current).Error; err != nil {
			return err
		}
		if strings.TrimSpace(current.AppSecretCiphertext) == "" {
			return fmt.Errorf("app secret still empty")
		}
		secret, err = crypto.DecryptAppSecret(masterKey, current.AppSecretCiphertext)
		return err
	})
	if txErr != nil {
		return "", txErr
	}
	return secret, nil
}
