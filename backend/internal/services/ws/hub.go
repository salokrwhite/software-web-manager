package ws

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 30 * time.Second
)

// InboundMessage is a control frame received from a client.
type InboundMessage struct {
	Type     string `json:"type"`
	TicketID string `json:"ticket_id"`
}

// Event is a frame broadcast to subscribed clients.
type Event struct {
	Type     string `json:"type"`
	TicketID string `json:"ticket_id,omitempty"`
	OrgID    string `json:"org_id,omitempty"`
	Payload  any    `json:"payload,omitempty"`
	Message  string `json:"message,omitempty"`
}

// Client is a single connected websocket peer.
type Client struct {
	conn      *websocket.Conn
	send      chan []byte
	subs      map[string]struct{}
	hub       *Hub
	closeOnce sync.Once
	authorize func(ticketID string) bool
}

// Hub fans published payloads out to clients subscribed to a ticket topic.
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
	subs    map[string]map[*Client]struct{}
}

// NewHub builds an empty hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*Client]struct{}),
		subs:    make(map[string]map[*Client]struct{}),
	}
}

// Serve registers a connection and runs its read/write pumps until it closes.
// authorize decides whether the client may subscribe to a given ticket id.
func (h *Hub) Serve(conn *websocket.Conn, authorize func(ticketID string) bool) {
	client := &Client{
		conn:      conn,
		send:      make(chan []byte, 128),
		subs:      make(map[string]struct{}),
		hub:       h,
		authorize: authorize,
	}
	h.addClient(client)
	go client.writePump()
	client.readPump()
}

// Publish delivers payload to every client subscribed to ticketID.
func (h *Hub) Publish(ticketID string, payload []byte) {
	h.mu.RLock()
	listeners := h.subs[ticketID]
	if len(listeners) == 0 {
		h.mu.RUnlock()
		return
	}
	clients := make([]*Client, 0, len(listeners))
	for client := range listeners {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		client.enqueue(payload)
	}
}

func (h *Hub) addClient(client *Client) {
	h.mu.Lock()
	h.clients[client] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) removeClient(client *Client) {
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

func (h *Hub) subscribe(client *Client, ticketID string) {
	h.mu.Lock()
	listeners, ok := h.subs[ticketID]
	if !ok {
		listeners = make(map[*Client]struct{})
		h.subs[ticketID] = listeners
	}
	if _, exists := listeners[client]; !exists {
		listeners[client] = struct{}{}
		client.subs[ticketID] = struct{}{}
	}
	h.mu.Unlock()
}

func (h *Hub) unsubscribe(client *Client, ticketID string) {
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

func (c *Client) enqueue(payload []byte) {
	select {
	case c.send <- payload:
	default:
		c.close()
	}
}

func (c *Client) close() {
	c.closeOnce.Do(func() {
		if c.hub != nil {
			c.hub.removeClient(c)
		}
		close(c.send)
		_ = c.conn.Close()
	})
}

func (c *Client) readPump() {
	defer c.close()
	for {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		var raw []byte
		if err := websocket.Message.Receive(c.conn, &raw); err != nil {
			return
		}
		if len(raw) == 0 {
			continue
		}
		var msg InboundMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			c.enqueueJSON(Event{Type: "error", Message: "invalid message"})
			continue
		}
		switch strings.ToLower(strings.TrimSpace(msg.Type)) {
		case "ping":
			c.enqueueJSON(Event{Type: "pong"})
		case "pong":
			continue
		case "subscribe":
			ticketID := strings.TrimSpace(msg.TicketID)
			if ticketID == "" {
				c.enqueueJSON(Event{Type: "error", Message: "ticket_id required"})
				continue
			}
			if _, err := uuid.Parse(ticketID); err != nil {
				c.enqueueJSON(Event{Type: "error", Message: "invalid ticket_id"})
				continue
			}
			if c.authorize == nil || !c.authorize(ticketID) {
				c.enqueueJSON(Event{Type: "error", Message: "forbidden"})
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
			c.enqueueJSON(Event{Type: "error", Message: "unknown message"})
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
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
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := websocket.Message.Send(c.conn, string(payload)); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			pingPayload, err := json.Marshal(Event{Type: "ping"})
			if err != nil {
				continue
			}
			if err := websocket.Message.Send(c.conn, string(pingPayload)); err != nil {
				return
			}
		}
	}
}

func (c *Client) enqueueJSON(event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	c.enqueue(payload)
}
