package core

import (
	"strings"
	"testing"
	"time"

	"software-web-manager/backend/internal/auth"
)

func testSigner(t *testing.T) *auth.AuthzSigner {
	t.Helper()
	signer, err := auth.NewAuthzSigner(strings.Repeat("ab", 32), "kid")
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	return signer
}

func TestAuthzSignerCacheSetGet(t *testing.T) {
	cache := NewAuthzSignerCache()
	signer := testSigner(t)
	cache.set("app-1", signer)

	got, ok := cache.get("app-1")
	if !ok || got != signer {
		t.Fatalf("expected cached signer, ok=%v got=%v", ok, got)
	}
	if _, ok := cache.get("app-missing"); ok {
		t.Fatalf("expected miss for unknown app")
	}
}

func TestAuthzSignerCacheExpiry(t *testing.T) {
	cache := NewAuthzSignerCache()
	// Insert an already-expired entry directly (same package) to avoid sleeping.
	cache.m["app-1"] = cachedAuthzSigner{signer: testSigner(t), expiresAt: time.Now().Add(-time.Minute)}
	if _, ok := cache.get("app-1"); ok {
		t.Fatalf("expected expired entry to miss")
	}
}

func TestAuthzSignerCacheInvalidate(t *testing.T) {
	cache := NewAuthzSignerCache()
	cache.set("app-1", testSigner(t))
	cache.Invalidate("app-1")
	if _, ok := cache.get("app-1"); ok {
		t.Fatalf("expected invalidated entry to miss")
	}

	// Invalidate on a nil cache must not panic (callers may have no cache wired).
	var nilCache *AuthzSignerCache
	nilCache.Invalidate("app-1")
}

// A cached nil signer (meaning "no signer / fail closed") is a valid, honored
// entry and should not be treated as a miss within its TTL.
func TestAuthzSignerCacheCachesNil(t *testing.T) {
	cache := NewAuthzSignerCache()
	cache.set("app-1", nil)
	got, ok := cache.get("app-1")
	if !ok {
		t.Fatalf("expected nil-signer entry to be a hit")
	}
	if got != nil {
		t.Fatalf("expected nil signer, got %v", got)
	}
}
