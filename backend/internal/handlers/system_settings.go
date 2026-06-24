package handlers

import (
	"software-web-manager/backend/internal/models"
	"software-web-manager/backend/internal/services/system"
)

// systemSvc builds a system-settings service from the handler's dependencies.
func (h *Handler) systemSvc() *system.Service {
	return system.NewService(h.DB)
}

// The following thin methods preserve the handler-level API used by sibling handlers
// (auth, registration, org management). They delegate to the system service.

func (h *Handler) ListSystemSettings() (map[string]models.SystemSetting, error) {
	return h.systemSvc().ListSettings()
}

func (h *Handler) AllowUserRegisterEnabled() (bool, error) {
	return h.systemSvc().AllowUserRegister()
}

func (h *Handler) AllowEnterpriseRegisterEnabled() (bool, error) {
	return h.systemSvc().AllowEnterpriseRegister()
}

func (h *Handler) GetOrgPlanTypes() ([]string, error) {
	return h.systemSvc().OrgPlanTypes()
}

func (h *Handler) GetSMTPConfigFromSettings(items map[string]models.SystemSetting) system.SMTPConfig {
	return h.systemSvc().SMTPConfigFromSettings(items)
}

func (h *Handler) GetSMTPPasswordFromSettings(items map[string]models.SystemSetting) (string, bool, error) {
	return h.systemSvc().SMTPPasswordFromSettings(items)
}
