// Package ws serves the authenticated WebSocket endpoint for live ticket updates.
// It validates the connecting user, then hands the raw connection to the shared
// ws hub (services/ws), supplying a subscribe-authorization callback.
package ws

import (
	"errors"
	"net/http"
	"strings"

	"software-web-manager/backend/internal/auth"
	"software-web-manager/backend/internal/core"
	"software-web-manager/backend/internal/models"
	wssvc "software-web-manager/backend/internal/services/ws"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/websocket"
)

var (
	errUserNotActive = errors.New("user not active")
	errOrgNotActive  = errors.New("org not active")
)

// Handler serves the websocket endpoint over the shared core.
type Handler struct {
	*core.Handler
}

// New builds a ws handler over the shared core.
func New(core *core.Handler) *Handler {
	return &Handler{Handler: core}
}

// EnsureHub lazily initializes the shared hub on first use.
func (h *Handler) EnsureHub() {
	if h.Hub == nil {
		h.Hub = wssvc.NewHub()
	}
}

func (h *Handler) HandleWS(c *gin.Context) {
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}
	claims, err := auth.ParseToken(h.Cfg.JWTSecret, token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	if err := h.ValidateWSUser(claims); err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, errOrgNotActive) || errors.Is(err, errUserNotActive) {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	h.EnsureHub()

	systemRole := strings.ToLower(strings.TrimSpace(claims.SystemRole))
	orgID := strings.TrimSpace(claims.OrgID)

	server := websocket.Server{
		Handshake: func(config *websocket.Config, req *http.Request) error {
			if !h.IsWSOriginAllowed(req) {
				return errors.New("origin not allowed")
			}
			return nil
		},
		Handler: func(conn *websocket.Conn) {
			h.Hub.Serve(conn, func(ticketID string) bool {
				return h.CanSubscribeTicket(systemRole, orgID, ticketID)
			})
		},
	}

	server.ServeHTTP(c.Writer, c.Request)
}

func (h *Handler) ValidateWSUser(claims *auth.Claims) error {
	if claims == nil || strings.TrimSpace(claims.UserID) == "" {
		return errors.New("invalid user")
	}
	var user models.User
	if err := h.DB.Where("id = ?", claims.UserID).First(&user).Error; err != nil {
		return errors.New("invalid user")
	}
	if strings.ToLower(strings.TrimSpace(user.Status)) != "active" {
		return errUserNotActive
	}
	systemRole := strings.ToLower(strings.TrimSpace(claims.SystemRole))
	orgID := strings.TrimSpace(claims.OrgID)
	if systemRole != "system_admin" && orgID != "" {
		var org models.Org
		if err := h.DB.Where("id = ?", orgID).First(&org).Error; err != nil {
			return errors.New("invalid org")
		}
		if strings.ToLower(strings.TrimSpace(org.Status)) != "active" {
			return errOrgNotActive
		}
	}
	return nil
}

// CanSubscribeTicket reports whether a connection with the given system role and
// org may subscribe to a ticket's updates.
func (h *Handler) CanSubscribeTicket(systemRole, orgID, ticketID string) bool {
	if strings.ToLower(systemRole) == "system_admin" {
		return true
	}
	if orgID == "" {
		return false
	}
	var count int64
	if err := h.DB.Model(&models.Ticket{}).
		Where("id = ? AND org_id = ?", ticketID, orgID).
		Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func (h *Handler) IsWSOriginAllowed(req *http.Request) bool {
	origin := strings.TrimSpace(req.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	if len(h.Cfg.CORSOrigins) == 0 {
		return true
	}
	if len(h.Cfg.CORSOrigins) == 1 && strings.TrimSpace(h.Cfg.CORSOrigins[0]) == "*" {
		return true
	}
	for _, allowed := range h.Cfg.CORSOrigins {
		if strings.TrimSpace(allowed) == origin {
			return true
		}
	}
	return false
}
