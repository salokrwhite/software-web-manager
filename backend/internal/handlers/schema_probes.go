package handlers

import (
	"software-web-manager/backend/internal/db/schema"
)

// These schema probes delegate to the internal/db/schema package, which holds
// the single source of truth. They are kept as thin methods so existing call
// sites (h.HasXxx()) remain unchanged.

func (h *Handler) HasOrgTypeColumn() bool {
	if h == nil {
		return false
	}
	return schema.HasOrgTypeColumn(h.DB)
}

func (h *Handler) HasAppFeedbackEnabledColumn() bool {
	if h == nil {
		return false
	}
	return schema.HasAppFeedbackEnabledColumn(h.DB)
}

func (h *Handler) HasAppHeartbeatIntervalColumn() bool {
	if h == nil {
		return false
	}
	return schema.HasAppHeartbeatIntervalColumn(h.DB)
}

func (h *Handler) HasAppOnlineEnabledColumn() bool {
	if h == nil {
		return false
	}
	return schema.HasAppOnlineEnabledColumn(h.DB)
}

func (h *Handler) HasAppMaintenanceColumn() bool {
	if h == nil {
		return false
	}
	return schema.HasAppMaintenanceColumn(h.DB)
}

func (h *Handler) HasAppPublicKeyColumn() bool {
	if h == nil {
		return false
	}
	return schema.HasAppPublicKeyColumn(h.DB)
}

func (h *Handler) HasReleaseExternalDownloadURLColumn() bool {
	if h == nil {
		return false
	}
	return schema.HasReleaseExternalDownloadURLColumn(h.DB)
}

func (h *Handler) HasAppSecretCiphertextColumn() bool {
	if h == nil {
		return false
	}
	return schema.HasAppSecretCiphertextColumn(h.DB)
}

func (h *Handler) HasAppSecretScopesColumn() bool {
	if h == nil {
		return false
	}
	return schema.HasAppSecretScopesColumn(h.DB)
}

func (h *Handler) HasAppSecretExpiresAtColumn() bool {
	if h == nil {
		return false
	}
	return schema.HasAppSecretExpiresAtColumn(h.DB)
}

func (h *Handler) HasAppSecretNameColumn() bool {
	if h == nil {
		return false
	}
	return schema.HasAppSecretNameColumn(h.DB)
}

func (h *Handler) HasAppSecretsTable() bool {
	if h == nil {
		return false
	}
	return schema.HasAppSecretsTable(h.DB)
}

func (h *Handler) HasDeviceControlsTable() bool {
	if h == nil {
		return false
	}
	return schema.HasDeviceControlsTable(h.DB)
}

func (h *Handler) HasFeedbackTable() bool {
	if h == nil {
		return false
	}
	return schema.HasFeedbackTable(h.DB)
}

func (h *Handler) HasFeedbackWorkflowColumns() bool {
	if h == nil {
		return false
	}
	return schema.HasFeedbackWorkflowColumns(h.DB)
}

func (h *Handler) HasSystemSettingsTable() bool {
	if h == nil {
		return false
	}
	return schema.HasSystemSettingsTable(h.DB)
}

func (h *Handler) HasEmailVerificationCodesTable() bool {
	if h == nil {
		return false
	}
	return schema.HasEmailVerificationCodesTable(h.DB)
}
