// Package clientupdate provides the in-memory publish/subscribe hub used by the
// client update SSE stream. Producers publish Events (release published, rolled
// back, maintenance, device shutdown, ...) and each connected client holds a
// Subscription that receives the events matching its app/channel/platform topic.
//
// The hub is transport-agnostic: it deals only with Events and Subscriptions and
// never touches gin or the database.
package clientupdate

import (
	"sync"
	"sync/atomic"
	"time"
)

const maxConnectionsPerIPApp = 50

// Event is a single client-update notification delivered over the SSE stream.
type Event struct {
	ID                 string     `json:"id"`
	EventType          string     `json:"event_type"`
	OrgID              string     `json:"org_id"`
	AppID              string     `json:"app_id"`
	DeviceID           string     `json:"device_id,omitempty"`
	ChannelCode        string     `json:"channel_code"`
	Platform           string     `json:"platform"`
	Arch               string     `json:"arch"`
	ReleaseID          string     `json:"release_id"`
	PublishedAt        time.Time  `json:"published_at"`
	Reason             string     `json:"reason"`
	Message            string     `json:"message,omitempty"`
	MaintenanceStartAt *time.Time `json:"maintenance_start_at,omitempty"`
}

// Subscription is one connected client's interest. Fields are populated by the
// caller before Subscribe; ID is assigned by the hub.
type Subscription struct {
	ID          int64
	ConnKey     string
	OrgID       string
	AppID       string
	DeviceID    string
	ChannelCode string
	Platform    string
	Arch        string
	Send        chan Event
}

// Hub fans published Events out to matching subscriptions.
type Hub struct {
	mu          sync.RWMutex
	nextID      int64
	subs        map[int64]*Subscription
	connCounter map[string]int
}

// NewHub builds an empty hub.
func NewHub() *Hub {
	return &Hub{
		subs:        make(map[int64]*Subscription),
		connCounter: make(map[string]int),
	}
}

// Subscribe registers a subscription, enforcing the per-IP-per-app connection
// cap. It returns false when the cap is exceeded.
func (h *Hub) Subscribe(sub *Subscription) bool {
	if h == nil || sub == nil {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.connCounter[sub.ConnKey] >= maxConnectionsPerIPApp {
		return false
	}
	id := atomic.AddInt64(&h.nextID, 1)
	sub.ID = id
	h.subs[id] = sub
	h.connCounter[sub.ConnKey]++
	return true
}

// Unsubscribe removes a subscription by id.
func (h *Hub) Unsubscribe(id int64) {
	if h == nil || id == 0 {
		return
	}
	h.mu.Lock()
	sub, ok := h.subs[id]
	if ok {
		delete(h.subs, id)
		if sub.ConnKey != "" {
			if h.connCounter[sub.ConnKey] <= 1 {
				delete(h.connCounter, sub.ConnKey)
			} else {
				h.connCounter[sub.ConnKey]--
			}
		}
	}
	h.mu.Unlock()
}

// Publish delivers evt to every matching subscription. Slow subscribers whose
// buffered channel is full are dropped to protect publisher throughput.
func (h *Hub) Publish(evt Event) {
	if h == nil {
		return
	}
	h.mu.RLock()
	targets := make([]*Subscription, 0, len(h.subs))
	for _, sub := range h.subs {
		if !matchesTopic(sub, evt) {
			continue
		}
		targets = append(targets, sub)
	}
	h.mu.RUnlock()

	for _, sub := range targets {
		select {
		case sub.Send <- evt:
		default:
			// slow subscribers are dropped to protect publisher throughput
			go h.Unsubscribe(sub.ID)
		}
	}
}

func matchesTopic(sub *Subscription, evt Event) bool {
	if sub == nil {
		return false
	}
	if sub.OrgID != evt.OrgID || sub.AppID != evt.AppID {
		return false
	}
	if evt.DeviceID != "" && sub.DeviceID != evt.DeviceID {
		return false
	}
	if evt.ChannelCode != "" && sub.ChannelCode != evt.ChannelCode {
		return false
	}
	if evt.Platform != "" && evt.Platform != "universal" && sub.Platform != evt.Platform {
		return false
	}
	if evt.Arch != "" && evt.Arch != "universal" && sub.Arch != evt.Arch {
		return false
	}
	return true
}
