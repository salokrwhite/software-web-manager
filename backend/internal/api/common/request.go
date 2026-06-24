package common

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// RequestScheme returns the effective request scheme, honoring X-Forwarded-Proto.
func RequestScheme(c *gin.Context) string {
	if proto := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); proto != "" {
		return strings.TrimSpace(strings.Split(proto, ",")[0])
	}
	if c.Request != nil && c.Request.TLS != nil {
		return "https"
	}
	return "http"
}

// RequestHost returns the effective request host, honoring X-Forwarded-Host.
func RequestHost(c *gin.Context) string {
	if host := strings.TrimSpace(c.GetHeader("X-Forwarded-Host")); host != "" {
		return strings.TrimSpace(strings.Split(host, ",")[0])
	}
	return c.Request.Host
}

// SSODeriveRedirectURI derives the SSO callback URL from the incoming request.
func SSODeriveRedirectURI(c *gin.Context) string {
	return RequestScheme(c) + "://" + RequestHost(c) + "/api/auth/sso/callback"
}
