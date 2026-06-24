package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"software-web-manager/backend/internal/auth"
	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/net/websocket"
)

const (
	wsWriteWait  = 10 * time.Second
	wsPongWait   = 60 * time.Second
	wsPingPeriod = 30 * time.Second
)

type wsInboundMessage struct {
	Type     string `json:"type"`
	TicketID string `json:"ticket_id"`
}

type wsEvent struct {
	Type     string `json:"type"`
	TicketID string `json:"ticket_id,omitempty"`
	OrgID    string `json:"org_id,omitempty"`
	Payload  any    `json:"payload,omitempty"`
	Message  string `json:"message,omitempty"`
}

type wsClient struct {
	conn       *websocket.Conn
	send       chan []byte
	userID     string
	orgID      string
	systemRole string
	subs       map[string]struct{}
	hub        *wsHub
	closeOnce  sync.Once
}

type wsHub struct {
	mu      sync.RWMutex
	clients map[*wsClient]struct{}
	subs    map[string]map[*wsClient]struct{}
}

func newWSHub() *wsHub {
	return &wsHub{
		clients: make(map[*wsClient]struct{}),
		subs:    make(map[string]map[*wsClient]struct{}),
	}
}

func (h *Handler) EnsureHub() {
	if h.Hub == nil {
		h.Hub = newWSHub()
	}
}

func (h *wsHub) addClient(client *wsClient) {
	h.mu.Lock()
	h.clients[client] = struct{}{}
	h.mu.Unlock()
}

func (h *wsHub) removeClient(client *wsClient) {
	h.mu.Lock()
	delete(h.clients, client)
	for ticketID := range client.subs {
		if listeners, ok := h.subs[ticketID]; ok {
			delete(listeners, client)
			if len(listeners) == 0 {
				delete(h.subs, ticketID)
			}
		}
	}
	h.mu.Unlock()
}

func (h *wsHub) subscribe(client *wsClient, ticketID string) {
	h.mu.Lock()
	listeners, ok := h.subs[ticketID]
	if !ok {
		listeners = make(map[*wsClient]struct{})
		h.subs[ticketID] = listeners
	}
	if _, exists := listeners[client]; !exists {
		listeners[client] = struct{}{}
		client.subs[ticketID] = struct{}{}
	}
	h.mu.Unlock()
}

func (h *wsHub) unsubscribe(client *wsClient, ticketID string) {
	h.mu.Lock()
	if listeners, ok := h.subs[ticketID]; ok {
		delete(listeners, client)
		if len(listeners) == 0 {
			delete(h.subs, ticketID)
		}
	}
	delete(client.subs, ticketID)
	h.mu.Unlock()
}

func (h *wsHub) publish(ticketID string, payload []byte) {
	h.mu.RLock()
	listeners := h.subs[ticketID]
	if len(listeners) == 0 {
		h.mu.RUnlock()
		return
	}
	clients := make([]*wsClient, 0, len(listeners))
	for client := range listeners {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		client.enqueue(payload)
	}
}

func (c *wsClient) enqueue(payload []byte) {
	select {
	case c.send <- payload:
	default:
		c.close()
	}
}

func (c *wsClient) close() {
	c.closeOnce.Do(func() {
		if c.hub != nil {
			c.hub.removeClient(c)
		}
		close(c.send)
		_ = c.conn.Close()
	})
}

func (c *wsClient) readPump(handler *Handler) {
	defer c.close()
	for {
		_ = c.conn.SetReadDeadline(time.Now().Add(wsPongWait))
		var raw []byte
		if err := websocket.Message.Receive(c.conn, &raw); err != nil {
			return
		}
		if len(raw) == 0 {
			continue
		}
		var msg wsInboundMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			c.enqueueJSON(wsEvent{Type: "error", Message: "invalid message"})
			continue
		}
		switch strings.ToLower(strings.TrimSpace(msg.Type)) {
		case "ping":
			c.enqueueJSON(wsEvent{Type: "pong"})
		case "pong":
			continue
		case "subscribe":
			ticketID := strings.TrimSpace(msg.TicketID)
			if ticketID == "" {
				c.enqueueJSON(wsEvent{Type: "error", Message: "ticket_id required"})
				continue
			}
			if _, err := uuid.Parse(ticketID); err != nil {
				c.enqueueJSON(wsEvent{Type: "error", Message: "invalid ticket_id"})
				continue
			}
			if !handler.CanSubscribeTicket(c, ticketID) {
				c.enqueueJSON(wsEvent{Type: "error", Message: "forbidden"})
				return
			}
			c.hub.subscribe(c, ticketID)
		case "unsubscribe":
			ticketID := strings.TrimSpace(msg.TicketID)
			if ticketID == "" {
				continue
			}
			c.hub.unsubscribe(c, ticketID)
		default:
			c.enqueueJSON(wsEvent{Type: "error", Message: "unknown message"})
		}
	}
}

func (c *wsClient) writePump() {
	ticker := time.NewTicker(wsPingPeriod)
	defer func() {
		ticker.Stop()
		c.close()
	}()
	for {
		select {
		case payload, ok := <-c.send:
			if !ok {
				return
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := websocket.Message.Send(c.conn, string(payload)); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			pingPayload, err := json.Marshal(wsEvent{Type: "ping"})
			if err != nil {
				continue
			}
			if err := websocket.Message.Send(c.conn, string(pingPayload)); err != nil {
				return
			}
		}
	}
}

func (c *wsClient) enqueueJSON(event wsEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	c.enqueue(payload)
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

	server := websocket.Server{
		Handshake: func(config *websocket.Config, req *http.Request) error {
			if !h.IsWSOriginAllowed(req) {
				return errors.New("origin not allowed")
			}
			return nil
		},
		Handler: func(conn *websocket.Conn) {
			client := &wsClient{
				conn:       conn,
				send:       make(chan []byte, 128),
				userID:     strings.TrimSpace(claims.UserID),
				orgID:      strings.TrimSpace(claims.OrgID),
				systemRole: strings.ToLower(strings.TrimSpace(claims.SystemRole)),
				subs:       make(map[string]struct{}),
				hub:        h.Hub,
			}
			h.Hub.addClient(client)
			go client.writePump()
			client.readPump(h)
		},
	}

	server.ServeHTTP(c.Writer, c.Request)
}

var (
	errUserNotActive = errors.New("user not active")
	errOrgNotActive  = errors.New("org not active")
)

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

func (h *Handler) CanSubscribeTicket(client *wsClient, ticketID string) bool {
	if strings.ToLower(client.systemRole) == "system_admin" {
		return true
	}
	if client.orgID == "" {
		return false
	}
	var count int64
	if err := h.DB.Model(&models.Ticket{}).
		Where("id = ? AND org_id = ?", ticketID, client.orgID).
		Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func (h *Handler) PublishTicketEvent(eventType, ticketID, orgID string, payload any) {
	if h.Hub == nil {
		return
	}
	event := wsEvent{
		Type:     eventType,
		TicketID: ticketID,
		OrgID:    orgID,
		Payload:  payload,
	}
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.Hub.publish(ticketID, data)
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

