package core

import (
	"software-web-manager/backend/internal/middleware"
	orgsvc "software-web-manager/backend/internal/services/org"
)

// MaxEnterpriseMaterialSize bounds an uploaded enterprise-registration material file.
const MaxEnterpriseMaterialSize = 20 * 1024 * 1024

// NormalizeSystemRole lower-cases/trims a system role, defaulting to "none".
// The canonical implementation lives in services/org; this thin wrapper keeps
// existing core.NormalizeSystemRole call sites working.
func NormalizeSystemRole(role string) string {
	return orgsvc.NormalizeSystemRole(role)
}

// OrgStatusCode re-exports middleware.OrgStatusCode (the canonical home) so
// existing core.OrgStatusCode call sites keep working.
func OrgStatusCode(status string) string {
	return middleware.OrgStatusCode(status)
}

// UserStatusCode re-exports middleware.UserStatusCode so existing
// core.UserStatusCode call sites keep working.
func UserStatusCode(status string) string {
	return middleware.UserStatusCode(status)
}
