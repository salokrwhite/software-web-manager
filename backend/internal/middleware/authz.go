package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// RequireSystemAdmin aborts requests whose authenticated user is not a system admin.
func RequireSystemAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		systemRole := strings.ToLower(strings.TrimSpace(c.GetString(ContextSystemRole)))
		if systemRole != "system_admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient system role"})
			return
		}
		c.Next()
	}
}
