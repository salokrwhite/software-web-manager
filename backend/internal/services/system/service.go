package system

import (
	"software-web-manager/backend/internal/models"

	"gorm.io/gorm"
)

// Service provides read/write access to system-level settings. It carries only the
// dependencies it needs and is free of any HTTP/gin coupling.
type Service struct {
	DB *gorm.DB
}

// NewService builds a system settings service bound to the given database handle.
func NewService(db *gorm.DB) *Service {
	return &Service{DB: db}
}

// HasSettingsTable reports whether the system_settings table exists.
func (s *Service) HasSettingsTable() bool {
	if s == nil || s.DB == nil {
		return false
	}
	return s.DB.Migrator().HasTable(&models.SystemSetting{})
}

// ListSettings loads every system setting keyed by its setting key.
func (s *Service) ListSettings() (map[string]models.SystemSetting, error) {
	var items []models.SystemSetting
	if err := s.DB.Order("created_at asc").Find(&items).Error; err != nil {
		return nil, err
	}
	out := make(map[string]models.SystemSetting, len(items))
	for i := range items {
		item := items[i]
		out[item.SettingKey] = item
	}
	return out, nil
}

// AllowUserRegister reports whether public user registration is enabled.
func (s *Service) AllowUserRegister() (bool, error) {
	if !s.HasSettingsTable() {
		return DefaultAllowUserRegister, nil
	}
	items, err := s.ListSettings()
	if err != nil {
		return false, err
	}
	return GetBool(items, SettingAllowUserRegisterKey, DefaultAllowUserRegister), nil
}

// AllowEnterpriseRegister reports whether enterprise registration is enabled.
func (s *Service) AllowEnterpriseRegister() (bool, error) {
	if !s.HasSettingsTable() {
		return DefaultAllowEnterpriseRegister, nil
	}
	items, err := s.ListSettings()
	if err != nil {
		return false, err
	}
	return GetBool(items, SettingAllowEnterpriseRegisterKey, DefaultAllowEnterpriseRegister), nil
}

// OrgPlanTypes returns the configured set of selectable org plan types.
func (s *Service) OrgPlanTypes() ([]string, error) {
	if !s.HasSettingsTable() {
		return NormalizeOrgPlanTypes(nil), nil
	}
	items, err := s.ListSettings()
	if err != nil {
		return nil, err
	}
	return getOrgPlanTypesSetting(items), nil
}
