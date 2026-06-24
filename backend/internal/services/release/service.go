// Package release provides release-domain data access that is independent of the
// HTTP layer (no gin, no response writing).
package release

import (
	"software-web-manager/backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Service exposes release queries over a single database handle.
type Service struct {
	DB *gorm.DB
}

// NewService builds a release service from a database handle.
func NewService(db *gorm.DB) *Service {
	return &Service{DB: db}
}

// GetForOrg loads a release scoped to the given org (joined through its app),
// returning gorm.ErrRecordNotFound when nothing matches.
func (s *Service) GetForOrg(orgID, releaseID string) (models.Release, error) {
	var release models.Release
	if err := s.DB.Raw(`
		SELECT r.* FROM releases r
		JOIN apps a ON a.id = r.app_id
		WHERE r.id = ? AND a.org_id = ?
	`, releaseID, orgID).Scan(&release).Error; err != nil {
		return release, err
	}
	if release.ID == (uuid.UUID{}) {
		return release, gorm.ErrRecordNotFound
	}
	return release, nil
}
