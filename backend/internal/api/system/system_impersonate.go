package system

import (
	"net/http"
	"software-web-manager/backend/internal/auth"
	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"
	"software-web-manager/backend/internal/rbac"
	orgsvc "software-web-manager/backend/internal/services/org"
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Impersonate(c *gin.Context) {
	var req impersonateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	role := strings.ToLower(strings.TrimSpace(req.Role))
	if role == "" {
		role = "owner"
	}
	if !rbac.IsValidRole(role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
		return
	}
	var org models.Org
	if err := h.DB.Where("id = ?", req.OrgID).First(&org).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "org not found"})
		return
	}

	userID := c.GetString(middleware.ContextUserID)
	systemRole := orgsvc.NormalizeSystemRole(c.GetString(middleware.ContextSystemRole))
	var tvUser models.User
	_ = h.DB.Select("token_version").Where("id = ?", userID).First(&tvUser).Error
	tokens, err := auth.IssueTokens(h.Cfg.JWTSecret, h.Cfg.JWTIssuer, userID, org.ID.String(), role, systemRole, tvUser.TokenVersion, h.Cfg.AccessTokenMinutes, h.Cfg.RefreshTokenHours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"tokens":        tokens,
		"org_id":        org.ID,
		"role":          role,
		"system_role":   systemRole,
		"impersonating": true,
	})
}
