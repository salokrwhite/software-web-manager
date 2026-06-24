package common

import (
	"net/http"
	"strings"

	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/rbac"

	"github.com/gin-gonic/gin"
)

// HasPermission reports whether the request carries the given org permission.
// Platform system admins (incl. while impersonating) bypass org permission
// checks, consistent with system_admin bypasses elsewhere.
func HasPermission(c *gin.Context, code string) bool {
	key := strings.ToLower(strings.TrimSpace(code))
	if key == "" {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(c.GetString(middleware.ContextSystemRole)), "system_admin") {
		return true
	}
	permissions, _ := c.Get(rbac.ContextPermissions)
	switch v := permissions.(type) {
	case map[string]struct{}:
		return rbac.PermissionSetAllows(v, key)
	case []string:
		return rbac.PermissionSetAllows(rbac.ToPermissionSet(v), key)
	}
	return false
}

// RequirePermission writes a 403 and returns false when the permission is absent.
func RequirePermission(c *gin.Context, code string) bool {
	if HasPermission(c, code) {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
	return false
}

// GetRequestOrgID returns the org id bound to the request context.
func GetRequestOrgID(c *gin.Context) string {
	return strings.TrimSpace(c.GetString(middleware.ContextOrgID))
}
