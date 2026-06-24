// Package api is the HTTP composition root. It wires the shared handler core and
// the per-domain route groups together. Keeping composition here (rather than in
// the handlers package) lets domain subpackages import the core without creating
// an import cycle.
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
	"software-web-manager/backend/internal/api/org"
	"software-web-manager/backend/internal/api/profile"
	"software-web-manager/backend/internal/api/release"
	"software-web-manager/backend/internal/api/system"
	"software-web-manager/backend/internal/api/ticket"
	"software-web-manager/backend/internal/handlers"
	"software-web-manager/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes mounts every route group on the engine.
func RegisterRoutes(r *gin.Engine, h *handlers.Handler, installMode bool) {
	orgAPI := org.New(h)
	profileAPI := profile.New(h)
	systemAPI := system.New(h)
	clientAPI := clientapi.New(h)
	authAPI := auth.New(h)
	installWrap := func(handler gin.HandlerFunc) gin.HandlerFunc { return wrapWithInstallCheck(h, handler) }

	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true, "install_mode": installMode}) })
	if strings.TrimSpace(h.Cfg.LocalStoragePath) != "" {
		r.GET("/files/*filepath", wrapWithInstallCheck(h, h.ServeLocalFile))
	}
	r.GET("/api/ws", wrapWithInstallCheck(h, h.HandleWS))
	r.GET("/api/apps/:id/online/stream", wrapWithInstallCheck(h, h.StreamOnlineCount))

	apiGroup := r.Group("/api")
	{
		apiGroup.GET("/install/status", h.GetInstallStatus)
		apiGroup.POST("/install/test-db", h.TestDatabase)
		apiGroup.POST("/install", h.Install)
	}

	authAPI.RegisterPublicRoutes(apiGroup, installWrap)
	apiGroup.GET("/public/settings", wrapWithInstallCheck(h, h.GetPublicSettings))
	apiGroup.GET("/org-invites/:token", wrapWithInstallCheck(h, orgAPI.GetOrgInvitePublic))
	apiGroup.POST("/org-invites/:token/accept", wrapWithInstallCheck(h, orgAPI.AcceptOrgInvite))

	apiAuth := r.Group("/api")
	apiAuth.Use(installCheckMiddleware(h))
	apiAuth.Use(middleware.JWT(h.Cfg))
	apiAuth.Use(h.RequireJWTRequestSignature())
	apiAuth.Use(h.RequireActiveUser())
	{
		orgAPI.RegisterRoutes(apiAuth)

		app.New(h).RegisterRoutes(apiAuth)
		apiAuth.GET("/apps/:id/online", h.GetOnlineCount)
		apiAuth.GET("/apps/:id/online/stream-token", h.IssueOnlineStreamToken)
		apiAuth.GET("/apps/:id/online/devices", h.ListOnlineDevices)
		apiAuth.GET("/apps/:id/blocked-devices", h.ListBlockedDevices)
		apiAuth.POST("/apps/:id/blocked-devices", h.BlockDeviceByDeviceID)
		feedback.New(h).RegisterRoutes(apiAuth)
		apiAuth.GET("/apps/:id/region-rules", h.GetAppRegionRules)
		apiAuth.PATCH("/apps/:id/region-rules", h.UpdateAppRegionRules)
		release.New(h).RegisterRoutes(apiAuth)

		analytics.New(h).RegisterRoutes(apiAuth)
		audit.New(h).RegisterRoutes(apiAuth)
		ticket.New(h).RegisterRoutes(apiAuth)
		clientAPI.RegisterRoutes(apiAuth)
		apiAuth.POST("/devices/:id/block", h.BlockDevice)
		apiAuth.POST("/devices/:id/unblock", h.UnblockDevice)
		profileAPI.RegisterRoutes(apiAuth)
		apiAuth.GET("/profile/sso/bind", authAPI.SSOBindStart)
		apiAuth.POST("/profile/sso/unbind", authAPI.SSOUnbind)
		apiAuth.GET("/geo/resolve", h.ResolveGeo)
		apiAuth.GET("/geo/regions", h.ListGeoRegions)
	}

	apiSystem := r.Group("/api/system")
	apiSystem.Use(installCheckMiddleware(h))
	apiSystem.Use(middleware.JWT(h.Cfg))
	apiSystem.Use(h.RequireJWTRequestSignature())
	apiSystem.Use(h.RequireActiveUser())
	apiSystem.Use(middleware.RequireSystemAdmin())
	{
		systemAPI.RegisterRoutes(apiSystem)
		ticket.New(h).RegisterSystemRoutes(apiSystem)
		profileAPI.RegisterSystemRoutes(apiSystem)
		apiSystem.GET("/profile/sso/bind", authAPI.SSOBindStart)
		apiSystem.POST("/profile/sso/unbind", authAPI.SSOUnbind)
		apiSystem.GET("/settings", h.GetSystemSettings)
		apiSystem.PATCH("/settings", h.UpdateSystemSettings)
		apiSystem.POST("/settings/mail/test", h.TestSystemSMTP)
		apiSystem.POST("/settings/sso/discover", authAPI.SSODiscover)
	}

	client := r.Group("/api/client")
	client.Use(installCheckMiddleware(h))
	client.Use(middleware.ClientLimits(h.Cfg))
	client.Use(h.RequireClientSignature())
	{
		client.POST("/update-check", h.UpdateCheck)
		client.GET("/updates/stream", h.HandleClientUpdateStream)
		clientAPI.RegisterClientRoutes(client)
		client.POST("/feedback", feedback.New(h).ClientSubmitFeedback)
	}
}

func wrapWithInstallCheck(h *handlers.Handler, handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "system not installed"})
			return
		}
		handler(c)
	}
}

func installCheckMiddleware(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "system not installed"})
			c.Abort()
			return
		}
		c.Next()
	}
}
