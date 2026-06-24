// Package online provides an in-memory presence tracker that records the last
// heartbeat time per device and reports how many devices are currently online
// for an app within a sliding TTL window.
package online

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Tracker keeps the most recent heartbeat timestamp for each (app, device) pair
// and expires entries older than the configured TTL.
type Tracker struct {
	mu   sync.Mutex
	ttl  time.Duration
	apps map[uuid.UUID]map[string]int64
}

// NewTracker builds a Tracker with the given TTL and starts its background
// cleanup loop. A non-positive TTL falls back to 120 seconds.
func NewTracker(ttl time.Duration) *Tracker {
	if ttl <= 0 {
		ttl = 120 * time.Second
	}
	tracker := &Tracker{
		ttl:  ttl,
		apps: make(map[uuid.UUID]map[string]int64),
	}
	tracker.startCleanup()
	return tracker
}

// WindowSeconds returns the online window size in seconds.
func (t *Tracker) WindowSeconds() int {
	return int(t.ttl.Seconds())
}

// Touch records that a device was seen at ts.
func (t *Tracker) Touch(appID uuid.UUID, deviceID string, ts time.Time) {
	if deviceID == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	devices := t.apps[appID]
	if devices == nil {
		devices = make(map[string]int64)
		t.apps[appID] = devices
	}
	devices[deviceID] = ts.Unix()
}

// Count returns the number of devices seen within the TTL window for an app and
// prunes expired entries as a side effect.
func (t *Tracker) Count(appID uuid.UUID, now time.Time) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	devices := t.apps[appID]
	if len(devices) == 0 {
		return 0
	}
	cutoff := now.Add(-t.ttl).Unix()
	count := 0
	for id, seen := range devices {
		if seen >= cutoff {
			count++
			continue
		}
		delete(devices, id)
	}
	if len(devices) == 0 {
		delete(t.apps, appID)
	}
	return count
}

// Cleanup removes all entries older than the TTL across every app.
func (t *Tracker) Cleanup(now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	cutoff := now.Add(-t.ttl).Unix()
	for appID, devices := range t.apps {
		for id, seen := range devices {
			if seen < cutoff {
				delete(devices, id)
			}
		}
		if len(devices) == 0 {
			delete(t.apps, appID)
		}
	}
}

func (t *Tracker) startCleanup() {
	interval := t.ttl / 2
	if interval < 10*time.Second {
		interval = 10 * time.Second
	}
	ticker := time.NewTicker(interval)
	go func() {
		for now := range ticker.C {
			t.Cleanup(now)
		}
	}()
}
