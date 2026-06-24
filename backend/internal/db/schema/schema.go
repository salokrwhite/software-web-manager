// Package schema centralizes runtime schema-introspection probes. The product
// tolerates partially-migrated databases (older deployments may lack newer
// columns/tables), so handlers guard optional features behind these checks.
//
// Keeping them in one package gives a single source of truth and lets any layer
// (handlers, services) probe the schema without depending on the handlers core.
package schema

import (
	"software-web-manager/backend/internal/models"

	"gorm.io/gorm"
)

// HasOrgTypeColumn reports whether orgs.org_type exists.
func HasOrgTypeColumn(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	return db.Migrator().HasColumn(&models.Org{}, "org_type")
}

// HasAppFeedbackEnabledColumn reports whether apps.feedback_enabled exists.
func HasAppFeedbackEnabledColumn(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	return db.Migrator().HasColumn(&models.App{}, "feedback_enabled")
}

// HasAppHeartbeatIntervalColumn reports whether apps.heartbeat_interval_seconds exists.
func HasAppHeartbeatIntervalColumn(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	return db.Migrator().HasColumn(&models.App{}, "heartbeat_interval_seconds")
}

// HasAppOnlineEnabledColumn reports whether apps.online_enabled exists.
func HasAppOnlineEnabledColumn(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	return db.Migrator().HasColumn(&models.App{}, "online_enabled")
}

// HasAppMaintenanceColumn reports whether apps.maintenance_enabled exists.
func HasAppMaintenanceColumn(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	return db.Migrator().HasColumn(&models.App{}, "maintenance_enabled")
}

// HasAppPublicKeyColumn reports whether apps.public_key exists.
func HasAppPublicKeyColumn(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	return db.Migrator().HasColumn(&models.App{}, "public_key")
}

// HasReleaseExternalDownloadURLColumn reports whether releases.external_download_url exists.
func HasReleaseExternalDownloadURLColumn(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	return db.Migrator().HasColumn(&models.Release{}, "external_download_url")
}

// HasAppSecretCiphertextColumn reports whether apps.app_secret_ciphertext exists.
func HasAppSecretCiphertextColumn(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	return db.Migrator().HasColumn(&models.App{}, "app_secret_ciphertext")
}

// HasAppSecretScopesColumn reports whether apps.app_secret_scopes exists.
func HasAppSecretScopesColumn(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	return db.Migrator().HasColumn(&models.App{}, "app_secret_scopes")
}

// HasAppSecretExpiresAtColumn reports whether apps.app_secret_expires_at exists.
func HasAppSecretExpiresAtColumn(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	return db.Migrator().HasColumn(&models.App{}, "app_secret_expires_at")
}

// HasAppSecretNameColumn reports whether apps.app_secret_name exists.
func HasAppSecretNameColumn(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	return db.Migrator().HasColumn(&models.App{}, "app_secret_name")
}

// HasAppSecretsTable reports whether the app_secrets table exists.
func HasAppSecretsTable(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	return db.Migrator().HasTable(&models.AppSecret{})
}

// HasDeviceControlsTable reports whether the device_controls table exists.
func HasDeviceControlsTable(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	return db.Migrator().HasTable(&models.DeviceControl{})
}

// HasFeedbackTable reports whether the feedback table exists.
func HasFeedbackTable(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	return db.Migrator().HasTable(&models.Feedback{})
}

// HasFeedbackWorkflowColumns reports whether the feedback workflow columns all exist.
func HasFeedbackWorkflowColumns(db *gorm.DB) bool {
	if db == nil || !HasFeedbackTable(db) {
		return false
	}
	return db.Migrator().HasColumn(&models.Feedback{}, "status") &&
		db.Migrator().HasColumn(&models.Feedback{}, "internal_note") &&
		db.Migrator().HasColumn(&models.Feedback{}, "handled_by") &&
		db.Migrator().HasColumn(&models.Feedback{}, "handled_at") &&
		db.Migrator().HasColumn(&models.Feedback{}, "updated_at")
}

// HasSystemSettingsTable reports whether the system_settings table exists.
func HasSystemSettingsTable(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	return db.Migrator().HasTable(&models.SystemSetting{})
}

// HasEmailVerificationCodesTable reports whether the email_verification_codes table exists.
func HasEmailVerificationCodesTable(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	return db.Migrator().HasTable(&models.EmailVerificationCode{})
}
