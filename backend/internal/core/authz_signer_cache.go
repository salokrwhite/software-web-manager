package core

import (
	"sync"
	"time"

	"software-web-manager/backend/internal/auth"
	"software-web-manager/backend/internal/crypto"
	"software-web-manager/backend/internal/db/schema"
	"software-web-manager/backend/internal/models"
	appsvc "software-web-manager/backend/internal/services/app"
)

// authzSignerCacheTTL bounds how long a resolved per-app signer is cached. The
// cache only avoids the per-request seed decrypt + Ed25519 parse; it is not a
// correctness mechanism. Rotation/revocation invalidate the entry within the
// same process immediately (see Invalidate) and across instances within the TTL.
const authzSignerCacheTTL = 60 * time.Second

type cachedAuthzSigner struct {
	signer    *auth.AuthzSigner // may be nil (means: no signer for this app right now)
	expiresAt time.Time
}

// AuthzSignerCache is an in-memory, TTL-bounded cache of the active signer per
// app. It is safe for concurrent use. Cross-instance consistency relies on the
// TTL alone (in-memory invalidation cannot reach other processes), which is the
// only approach that is consistent in a multi-instance deployment.
type AuthzSignerCache struct {
	mu  sync.RWMutex
	ttl time.Duration
	m   map[string]cachedAuthzSigner
}

// NewAuthzSignerCache builds an empty cache with the default TTL.
func NewAuthzSignerCache() *AuthzSignerCache {
	return &AuthzSignerCache{ttl: authzSignerCacheTTL, m: make(map[string]cachedAuthzSigner)}
}

func (cache *AuthzSignerCache) get(appID string) (*auth.AuthzSigner, bool) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	entry, ok := cache.m[appID]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.signer, true
}

func (cache *AuthzSignerCache) set(appID string, signer *auth.AuthzSigner) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.m[appID] = cachedAuthzSigner{signer: signer, expiresAt: time.Now().Add(cache.ttl)}
}

// Invalidate drops the cached signer for an app so the next request re-resolves
// it. Called after activate/revoke so changes take effect immediately on this
// instance (other instances catch up within the TTL).
func (cache *AuthzSignerCache) Invalidate(appID string) {
	if cache == nil {
		return
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	delete(cache.m, appID)
}

// AuthzSignerForApp returns the signer to use for the given app's verdicts, or
// nil when the client must fail closed. Resolution order:
//  1. the app's active, non-revoked authz key (cached);
//  2. otherwise, the platform fallback signer when AUTHZ_PLATFORM_FALLBACK is on;
//  3. otherwise nil (no envelope -> client fail-closed).
func (h *Handler) AuthzSignerForApp(app models.App) *auth.AuthzSigner {
	appID := app.ID.String()
	if h.AuthzSignerCache != nil {
		if signer, ok := h.AuthzSignerCache.get(appID); ok {
			return signer
		}
	}
	signer := h.resolveAuthzSigner(app)
	if h.AuthzSignerCache != nil {
		h.AuthzSignerCache.set(appID, signer)
	}
	return signer
}

func (h *Handler) resolveAuthzSigner(app models.App) *auth.AuthzSigner {
	if h.DB != nil && schema.HasAppAuthzKeysTable(h.DB) {
		key, err := appsvc.NewService(h.DB).ActiveAuthzKey(app.ID)
		if err == nil {
			seedHex, derr := crypto.DecryptAppSecret(h.Cfg.AppSecretMasterKey, key.PrivateKeyCiphertext)
			if derr == nil {
				if signer, serr := auth.NewAuthzSigner(seedHex, key.KeyID); serr == nil {
					return signer
				}
			}
			// Decrypt/parse failure on an app key: fall through to the platform
			// fallback (or nil) rather than silently using the wrong key.
		}
	}
	if h.Cfg.AuthzPlatformFallback {
		return h.AuthzSigner // may itself be nil when no platform key is configured
	}
	return nil
}
