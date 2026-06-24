// Package api is the HTTP composition root. It wires the shared core (internal/core)
// and the per-domain route groups together. Keeping composition here (rather than in
// the core package) lets domain subpackages import the core without creating an
// import cycle.
package api

import (
	"net/http"
	"strings"

	"software-web-manager/backend/internal/api/analytics"
	"software-web-manager/backend/internal/api/app"
	"software-web-manager/backend/internal/api/audit"
	"software-web-manager/backend/internal/api/auth"
	clientapi "software-web-manager/backend/internal/api/client"
	"software-web-manager/backend/internal/api/feedback"
	geoapi "software-web-manager/backend/internal/api/geo"
	installapi "software-web-manager/backend/internal/api/install"
	"software-web-manager/backend/internal/api/org"
	"software-web-manager/backend/internal/api/profile"
	"software-web-manager/backend/internal/api/release"
	"software-web-manager/backend/internal/api/system"
	"software-web-manager/backend/internal/api/ticket"
	wsapi "software-web-manager/backend/internal/api/ws"
	"software-web-manager/backend/internal/core"
	"software-web-manager/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes mounts every route group on the engine.
func RegisterRoutes(r *gin.Engine, h *core.Handler, installMode bool) {
	orgAPI := org.New(h)
	profileAPI := profile.New(h)
	systemAPI := system.New(h)
	clientAPI := clientapi.New(h)
	authAPI := auth.New(h)
	wsAPI := wsapi.New(h)
	geoAPI := geoapi.New(h)
	installWrap := func(handler gin.HandlerFunc) gin.HandlerFunc { return wrapWithInstallCheck(h, handler) }

	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true, "install_mode": installMode}) })
	if strings.TrimSpace(h.Cfg.LocalStoragePath) != "" {
		r.GET("/files/*filepath", wrapWithInstallCheck(h, h.ServeLocalFile))
	}
	r.GET("/api/ws", wrapWithInstallCheck(h, wsAPI.HandleWS))
	r.GET("/api/apps/:id/online/stream", wrapWithInstallCheck(h, clientAPI.StreamOnlineCount))

	apiGroup := r.Group("/api")
	{
		installapi.New(h).RegisterRoutes(apiGroup)
	}

	authAPI.RegisterPublicRoutes(apiGroup, installWrap)
	apiGroup.GET("/public/settings", wrapWithInstallCheck(h, systemAPI.GetPublicSettings))
	apiGroup.GET("/org-invites/:token", wrapWithInstallCheck(h, orgAPI.GetOrgInvitePublic))
	apiGroup.POST("/org-invites/:token/accept", wrapWithInstallCheck(h, orgAPI.AcceptOrgInvite))

	apiAuth := r.Group("/api")
	apiAuth.Use(installCheckMiddleware(h))
	apiAuth.Use(middleware.JWT(h.Cfg))
	apiAuth.Use(h.RequireJWTRequestSignature())
	apiAuth.Use(middleware.RequireActiveUser(h.DB, h.LoadOrgPermissionSet))
	{
		orgAPI.RegisterRoutes(apiAuth)

		app.New(h).RegisterRoutes(apiAuth)
		feedback.New(h).RegisterRoutes(apiAuth)
		release.New(h).RegisterRoutes(apiAuth)

		analytics.New(h).RegisterRoutes(apiAuth)
		audit.New(h).RegisterRoutes(apiAuth)
		ticket.New(h).RegisterRoutes(apiAuth)
		clientAPI.RegisterRoutes(apiAuth)
		profileAPI.RegisterRoutes(apiAuth)
		apiAuth.GET("/profile/sso/bind", authAPI.SSOBindStart)
		apiAuth.POST("/profile/sso/unbind", authAPI.SSOUnbind)
		geoAPI.RegisterRoutes(apiAuth)
	}

	apiSystem := r.Group("/api/system")
	apiSystem.Use(installCheckMiddleware(h))
	apiSystem.Use(middleware.JWT(h.Cfg))
	apiSystem.Use(h.RequireJWTRequestSignature())
	apiSystem.Use(middleware.RequireActiveUser(h.DB, h.LoadOrgPermissionSet))
	apiSystem.Use(middleware.RequireSystemAdmin())
	{
		systemAPI.RegisterRoutes(apiSystem)
		ticket.New(h).RegisterSystemRoutes(apiSystem)
		profileAPI.RegisterSystemRoutes(apiSystem)
		apiSystem.GET("/profile/sso/bind", authAPI.SSOBindStart)
		apiSystem.POST("/profile/sso/unbind", authAPI.SSOUnbind)
		apiSystem.POST("/settings/sso/discover", authAPI.SSODiscover)
	}

	client := r.Group("/api/client")
	client.Use(installCheckMiddleware(h))
	client.Use(middleware.ClientLimits(h.Cfg))
	client.Use(h.RequireClientSignature())
	{
		clientAPI.RegisterClientRoutes(client)
		client.POST("/feedback", feedback.New(h).ClientSubmitFeedback)
	}
}

func wrapWithInstallCheck(h *core.Handler, handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "system not installed"})
			return
		}
		handler(c)
	}
}

func installCheckMiddleware(h *core.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "system not installed"})
			c.Abort()
			return
		}
		c.Next()
	}
}
