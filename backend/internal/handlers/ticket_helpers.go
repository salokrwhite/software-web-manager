package handlers

import (
	"errors"
	"strings"

	"github.com/google/uuid"
)

func normalizeTicketStatus(raw string) string {
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

func isValidTicketStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "submitted", "in_progress", "resolved":
		return true
	default:
		return false
	}
}

func canTransitionTicketStatus(current, next string) bool {
	current = normalizeTicketStatus(current)
	next = normalizeTicketStatus(next)
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

func normalizeTicketIDs(rawIDs []string) ([]string, error) {
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

