package handlers

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type OnlineTracker struct {
	mu   sync.Mutex
	ttl  time.Duration
	apps map[uuid.UUID]map[string]int64
}

func NewOnlineTracker(ttl time.Duration) *OnlineTracker {
	if ttl <= 0 {
		ttl = 120 * time.Second
	}
	tracker := &OnlineTracker{
		ttl:  ttl,
		apps: make(map[uuid.UUID]map[string]int64),
	}
	tracker.startCleanup()
	return tracker
}

func (t *OnlineTracker) WindowSeconds() int {
	return int(t.ttl.Seconds())
}

func (t *OnlineTracker) Touch(appID uuid.UUID, deviceID string, ts time.Time) {
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

func (t *OnlineTracker) Count(appID uuid.UUID, now time.Time) int {
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

func (t *OnlineTracker) Cleanup(now time.Time) {
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

func (t *OnlineTracker) startCleanup() {
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

