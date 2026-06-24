// Package app provides app-domain data access that is independent of the HTTP
// layer (no gin, no response writing).
package app

import (
	"software-web-manager/backend/internal/models"

	"gorm.io/gorm"
)

// Service exposes app queries over a single database handle.
type Service struct {
	DB *gorm.DB
}

// NewService builds an app service from a database handle.
func NewService(db *gorm.DB) *Service {
	return &Service{DB: db}
}

// GetForOrg loads an app scoped to the given org, returning the underlying
// gorm error (e.g. ErrRecordNotFound) when it is missing.
func (s *Service) GetForOrg(orgID, appID string) (models.App, error) {
	var app models.App
	if err := s.DB.Where("id = ? AND org_id = ?", appID, orgID).First(&app).Error; err != nil {
		return app, err
	}
	return app, nil
}
