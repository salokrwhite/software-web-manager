package app

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"software-web-manager/backend/internal/db/schema"
	"strings"
	"time"

	"software-web-manager/backend/internal/crypto"
	"software-web-manager/backend/internal/core"
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

func (h *Handler) EnsureAppSecret(appID string) (string, error) {
	var app models.App
	if err := h.DB.Where("id = ?", appID).First(&app).Error; err != nil {
		if IsAppSecretColumnMissingErr(err) {
			return "", fmt.Errorf("missing app_secret_ciphertext column, run migration 0029_app_secret_and_signature")
		}
		return "", err
	}

	if schema.HasAppSecretsTable(h.DB) {
		var current models.AppSecret
		err := h.DB.
			Where("app_id = ? AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > ?)", app.ID, time.Now()).
			Order("created_at asc").
			First(&current).Error
		if err == nil {
			return crypto.DecryptAppSecret(h.Cfg.AppSecretMasterKey, current.SecretCiphertext)
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
				row.ScopesJSON = core.AppSecretScopesJSON(nil)
			}
			if err := h.DB.Create(&row).Error; err != nil {
				return "", err
			}
			return crypto.DecryptAppSecret(h.Cfg.AppSecretMasterKey, row.SecretCiphertext)
		}

		secret, err := GenerateAppSecret()
		if err != nil {
			return "", err
		}
		secretCipher, err := crypto.EncryptAppSecret(h.Cfg.AppSecretMasterKey, secret)
		if err != nil {
			return "", err
		}
		row := models.AppSecret{
			AppID:            app.ID,
			Name:             "app_secret",
			SecretCiphertext: secretCipher,
			ScopesJSON:       core.AppSecretScopesJSON(nil),
		}
		if err := h.DB.Create(&row).Error; err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				var created models.AppSecret
				if loadErr := h.DB.
					Where("app_id = ? AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > ?)", app.ID, time.Now()).
					Order("created_at asc").
					First(&created).Error; loadErr == nil {
					return crypto.DecryptAppSecret(h.Cfg.AppSecretMasterKey, created.SecretCiphertext)
				}
			}
			return "", err
		}
		return secret, nil
	}

	if strings.TrimSpace(app.AppSecretCiphertext) != "" {
		return crypto.DecryptAppSecret(h.Cfg.AppSecretMasterKey, app.AppSecretCiphertext)
	}

	secret, err := GenerateAppSecret()
	if err != nil {
		return "", err
	}
	secretCipher, err := crypto.EncryptAppSecret(h.Cfg.AppSecretMasterKey, secret)
	if err != nil {
		return "", err
	}
	now := time.Now()
	txErr := h.DB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"app_secret_ciphertext": secretCipher,
			"app_secret_updated_at": now,
		}
		if schema.HasAppSecretScopesColumn(h.DB) {
			updates["app_secret_scopes"] = core.AppSecretScopesJSON(nil)
		}
		if schema.HasAppSecretExpiresAtColumn(h.DB) {
			updates["app_secret_expires_at"] = nil
		}
		if schema.HasAppSecretNameColumn(h.DB) {
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
		secret, err = crypto.DecryptAppSecret(h.Cfg.AppSecretMasterKey, current.AppSecretCiphertext)
		return err
	})
	if txErr != nil {
		return "", txErr
	}
	return secret, nil
}

func IsAppSecretColumnMissingErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unknown column") && strings.Contains(msg, "app_secret_ciphertext")
}
