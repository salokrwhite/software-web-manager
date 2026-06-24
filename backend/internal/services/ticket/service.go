// Package ticket provides ticket-domain data access and status-flow logic that
// is independent of the HTTP layer (no gin, no response writing).
package ticket

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrInvalidStatusTransition is returned when a requested status change is not
// permitted by the ticket status flow.
var ErrInvalidStatusTransition = errors.New("invalid status transition")

// Service exposes ticket queries and commands over a single database handle.
type Service struct {
	DB *gorm.DB
}

// NewService builds a ticket service from a database handle.
func NewService(db *gorm.DB) *Service {
	return &Service{DB: db}
}

// NormalizeStatus lower-cases/trims a status and maps aliases to canonical form.
func NormalizeStatus(raw string) string {
	status := strings.ToLower(strings.TrimSpace(raw))
	switch status {
	case "submitted", "in_progress", "resolved":
		return status
	case "in-progress", "in progress":
		return "in_progress"
	default:
		return status
	}
}

// IsValidStatus reports whether the status is a known canonical ticket status.
func IsValidStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "submitted", "in_progress", "resolved":
		return true
	default:
		return false
	}
}

// CanTransition reports whether a ticket may move from current to next status.
func CanTransition(current, next string) bool {
	current = NormalizeStatus(current)
	next = NormalizeStatus(next)
	if current == next {
		return true
	}
	switch current {
	case "submitted":
		return next == "in_progress" || next == "resolved"
	case "in_progress":
		return next == "resolved"
	default:
		return false
	}
}

// NormalizeIDs validates and de-duplicates a set of ticket ids.
func NormalizeIDs(rawIDs []string) ([]string, error) {
	if len(rawIDs) == 0 {
		return nil, errors.New("ids required")
	}
	seen := make(map[string]struct{}, len(rawIDs))
	ids := make([]string, 0, len(rawIDs))
	for _, raw := range rawIDs {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, err := uuid.Parse(value); err != nil {
			return nil, errors.New("invalid id")
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		ids = append(ids, value)
	}
	if len(ids) == 0 {
		return nil, errors.New("ids required")
	}
	return ids, nil
}
