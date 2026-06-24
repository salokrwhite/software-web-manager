// Package feedback provides feedback-domain data access and status logic that is
// independent of the HTTP layer (no gin, no response writing).
package feedback

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// Sentinel errors returned by feedback updates so the HTTP layer can map them to
// the right status code.
var (
	ErrInvalidStatus       = errors.New("invalid feedback status")
	ErrInternalNoteTooLong = errors.New("internal_note too long")
	ErrNoUpdates           = errors.New("no updates")
	ErrNotFound            = errors.New("feedback not found")
)

// Service exposes feedback queries and commands over a single database handle.
type Service struct {
	DB *gorm.DB
}

// NewService builds a feedback service from a database handle.
func NewService(db *gorm.DB) *Service {
	return &Service{DB: db}
}

// NormalizeStatus lower-cases/trims a status, defaulting to "open".
func NormalizeStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return "open"
	}
	return status
}

// IsValidStatus reports whether the status is a known feedback workflow status.
func IsValidStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "open", "processing", "resolved", "closed":
		return true
	default:
		return false
	}
}
