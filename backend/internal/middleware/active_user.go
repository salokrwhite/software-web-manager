package middleware

import (
	"errors"
	"net/http"
	"strings"

	"software-web-manager/backend/internal/models"
	"software-web-manager/backend/internal/rbac"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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

// LoadOrgPermissionSetFunc resolves an org member's effective permission set.
type LoadOrgPermissionSetFunc func(orgID, role string) (map[string]struct{}, error)

// RequireActiveUser validates that the authenticated user is active and, when an
// org is in scope, that the user is still a member of an active org. It records
// the member role and resolved permission set on the context. Its dependencies
// (database handle and permission loader) are injected so the middleware does
// not depend on the handlers layer.
func RequireActiveUser(db *gorm.DB, loadOrgPermissions LoadOrgPermissionSetFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := strings.TrimSpace(c.GetString(ContextUserID))
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
			return
		}
		var user models.User
		if err := db.Where("id = ?", userID).First(&user).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
			return
		}
		if strings.ToLower(strings.TrimSpace(user.Status)) != "active" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "user not active", "code": UserStatusCode(user.Status)})
			return
		}

		systemRole := strings.ToLower(strings.TrimSpace(c.GetString(ContextSystemRole)))
		orgID := strings.TrimSpace(c.GetString(ContextOrgID))
		if systemRole != "system_admin" && orgID != "" {
			var member models.OrgMember
			if err := db.Where("scope_id = ? AND user_id = ?", orgID, user.ID).First(&member).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "org access revoked", "code": "org_access_revoked"})
					return
				}
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to query org membership"})
				return
			}
			var org models.Org
			if err := db.Where("id = ?", orgID).First(&org).Error; err == nil {
				if strings.ToLower(strings.TrimSpace(org.Status)) != "active" {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "org not active", "code": OrgStatusCode(org.Status)})
					return
				}
			}
			c.Set(ContextRole, member.Role)
			permissionSet, err := loadOrgPermissions(orgID, member.Role)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to load permissions"})
				return
			}
			c.Set(rbac.ContextPermissions, permissionSet)
		}
		c.Next()
	}
}
