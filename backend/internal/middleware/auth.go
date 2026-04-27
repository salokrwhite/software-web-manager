package middleware

import (
	"net/http"
	"strings"

	"software-web-manager/backend/internal/auth"
	"software-web-manager/backend/internal/config"

	"github.com/gin-gonic/gin"
)

const ContextUserID = "user_id"
const ContextOrgID = "org_id"
const ContextRole = "role"
const ContextSystemRole = "system_role"
const ContextRawToken = "raw_jwt_token"

func JWT(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization header"})
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid Authorization header"})
			return
		}
		claims, err := auth.ParseToken(cfg.JWTSecret, parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set(ContextRawToken, parts[1])
		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextOrgID, claims.OrgID)
		c.Set(ContextRole, claims.Role)
		c.Set(ContextSystemRole, claims.SystemRole)
		c.Next()
	}
}

