package handlers

import "strings"

// MaxEnterpriseMaterialSize bounds an uploaded enterprise-registration material file.
const MaxEnterpriseMaterialSize = 20 * 1024 * 1024

// NormalizeSystemRole lower-cases/trims a system role, defaulting to "none".
func NormalizeSystemRole(role string) string {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		return "none"
	}
	return role
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
