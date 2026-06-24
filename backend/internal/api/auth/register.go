// Package auth holds the authentication-domain HTTP handlers: login/registration,
// enterprise registration, email-code registration, and the OIDC SSO flows.
package auth

import (
	"software-web-manager/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

// Handler serves the auth-domain endpoints.
type Handler struct {
	*handlers.Handler
}

// New builds an auth handler over the shared core.
func New(core *handlers.Handler) *Handler {
	return &Handler{Handler: core}
}

// RegisterPublicRoutes wires the unauthenticated auth routes onto the public API group.
// wrap applies the shared install-check guard to each handler.
func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup, wrap func(gin.HandlerFunc) gin.HandlerFunc) {
	rg.POST("/auth/register", wrap(h.Register))
	rg.POST("/auth/register/send-code", wrap(h.SendRegisterEmailCode))
	rg.POST("/auth/enterprise-register", wrap(h.EnterpriseRegister))
	rg.GET("/auth/enterprise-status/:id", wrap(h.GetEnterpriseStatus))
	rg.POST("/auth/enterprise-resubmit", wrap(h.EnterpriseResubmit))
	rg.POST("/auth/login", wrap(h.Login))
	rg.POST("/auth/admin-login", wrap(h.AdminLogin))
	rg.POST("/auth/refresh", wrap(h.Refresh))
	rg.GET("/auth/sso/login", wrap(h.SSOLogin))
	rg.GET("/auth/sso/callback", wrap(h.SSOCallback))
	rg.GET("/auth/sso/logout", wrap(h.SSOLogoutURL))
}
