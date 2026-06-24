package handlers

import (
	"errors"
	"net/http"
	"strings"

	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handler) RequireActiveUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
			return
		}
		var user models.User
		if err := h.DB.Where("id = ?", userID).First(&user).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
			return
		}
		if strings.ToLower(strings.TrimSpace(user.Status)) != "active" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "user not active", "code": UserStatusCode(user.Status)})
			return
		}

		systemRole := strings.ToLower(strings.TrimSpace(c.GetString(middleware.ContextSystemRole)))
		orgID := strings.TrimSpace(c.GetString(middleware.ContextOrgID))
		if systemRole != "system_admin" && orgID != "" {
			var member models.OrgMember
			if err := h.DB.Where("scope_id = ? AND user_id = ?", orgID, user.ID).First(&member).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "org access revoked", "code": "org_access_revoked"})
					return
				}
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to query org membership"})
				return
			}
			var org models.Org
			if err := h.DB.Where("id = ?", orgID).First(&org).Error; err == nil {
				if strings.ToLower(strings.TrimSpace(org.Status)) != "active" {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "org not active", "code": OrgStatusCode(org.Status)})
					return
				}
			}
			c.Set(middleware.ContextRole, member.Role)
			permissionSet, err := h.LoadOrgPermissionSet(orgID, member.Role)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to load permissions"})
				return
			}
			c.Set(ContextPermissions, permissionSet)
		}
		c.Next()
	}
}


