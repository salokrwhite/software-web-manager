package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"software-web-manager/backend/internal/config"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type clientLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	lastSeen map[string]time.Time
	rate     rate.Limit
	burst    int
	ttl      time.Duration
}

func newClientLimiter(rps int, burst int) *clientLimiter {
	if rps <= 0 || burst <= 0 {
		return nil
	}
	return &clientLimiter{
		limiters: make(map[string]*rate.Limiter),
		lastSeen: make(map[string]time.Time),
		rate:     rate.Limit(rps),
		burst:    burst,
		ttl:      10 * time.Minute,
	}
}

func (c *clientLimiter) get(ip string) *rate.Limiter {
	c.mu.Lock()
	defer c.mu.Unlock()
	if lim, ok := c.limiters[ip]; ok {
		c.lastSeen[ip] = time.Now()
		return lim
	}
	lim := rate.NewLimiter(c.rate, c.burst)
	c.limiters[ip] = lim
	c.lastSeen[ip] = time.Now()
	return lim
}

func (c *clientLimiter) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	cutoff := time.Now().Add(-c.ttl)
	for ip, ts := range c.lastSeen {
		if ts.Before(cutoff) {
			delete(c.lastSeen, ip)
			delete(c.limiters, ip)
		}
	}
}

type allowlist struct {
	ips   map[string]struct{}
	nets  []*net.IPNet
	empty bool
}

func parseAllowlist(items []string) *allowlist {
	al := &allowlist{ips: make(map[string]struct{}), empty: true}
	for _, raw := range items {
		item := strings.TrimSpace(raw)
		if item == "" {
			continue
		}
		al.empty = false
		if strings.Contains(item, "/") {
			if _, cidr, err := net.ParseCIDR(item); err == nil {
				al.nets = append(al.nets, cidr)
				continue
			}
		}
		al.ips[item] = struct{}{}
	}
	return al
}

func (a *allowlist) allowed(ip string) bool {
	if a == nil || a.empty {
		return true
	}
	if _, ok := a.ips[ip]; ok {
		return true
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, cidr := range a.nets {
		if cidr.Contains(parsed) {
			return true
		}
	}
	return false
}

func ClientLimits(cfg config.Config) gin.HandlerFunc {
	al := parseAllowlist(cfg.ClientIPAllowlist)
	limiter := newClientLimiter(cfg.ClientRateLimitRPS, cfg.ClientRateLimitBurst)
	if limiter != nil {
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				limiter.cleanup()
			}
		}()
	}
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !al.allowed(ip) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "ip not allowed"})
			return
		}
		if limiter != nil && !limiter.get(ip).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}
