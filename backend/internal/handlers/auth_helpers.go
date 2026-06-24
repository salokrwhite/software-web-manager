package handlers

import (
	"strings"

	orgsvc "software-web-manager/backend/internal/services/org"
)

// MaxEnterpriseMaterialSize bounds an uploaded enterprise-registration material file.
const MaxEnterpriseMaterialSize = 20 * 1024 * 1024

// NormalizeSystemRole lower-cases/trims a system role, defaulting to "none".
// The canonical implementation lives in services/org; this thin wrapper keeps
// existing handlers.NormalizeSystemRole call sites working.
func NormalizeSystemRole(role string) string {
	return orgsvc.NormalizeSystemRole(role)
}

// OrgStatusCode maps an org status to its client-facing error code.
func OrgStatusCode(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending":
		return "org_pending"
	case "disabled":
		return "org_disabled"
	default:
		return "org_inactive"
	}
}

// UserStatusCode maps a user status to its client-facing error code.
func UserStatusCode(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending":
		return "user_pending"
	case "disabled":
		return "user_disabled"
	default:
		return "user_inactive"
	}
}
