// Package auth holds pure, HTTP-context-free authentication business logic: the
// OIDC/OAuth primitives used by the SSO flow (PKCE, random tokens, audience checks,
// claim extraction) and the JWKS public-key client (fetch, cache, parse, lookup).
package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

// HTTPClient returns the HTTP client used for SSO/OIDC requests.
func HTTPClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

// SanitizeRedirect bounds a post-login redirect target to a safe in-app path.
func SanitizeRedirect(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || !strings.HasPrefix(v, "/") || strings.HasPrefix(v, "//") {
		return "/dashboard"
	}
	if strings.HasPrefix(v, "/login") || strings.HasPrefix(v, "/admin-login") {
		return "/dashboard"
	}
	return v
}

// RandomToken returns a base64url-encoded random token of n bytes.
func RandomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failure is effectively fatal; fall back to time-based bytes.
		ts := time.Now().UnixNano()
		for i := range b {
			b[i] = byte(ts >> (uint(i%8) * 8))
		}
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// PKCEChallenge derives the S256 PKCE code challenge for a verifier.
func PKCEChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// B64URLDecode decodes a base64url (no-padding) string, tolerating trailing '='.
func B64URLDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(strings.TrimRight(strings.TrimSpace(s), "="))
}

// AudienceContains reports whether an OIDC `aud` claim contains clientID.
func AudienceContains(aud interface{}, clientID string) bool {
	if clientID == "" {
		return false
	}
	switch v := aud.(type) {
	case string:
		return v == clientID
	case []interface{}:
		for _, a := range v {
			if s, ok := a.(string); ok && s == clientID {
				return true
			}
		}
	case []string:
		for _, s := range v {
			if s == clientID {
				return true
			}
		}
	}
	return false
}

// ClaimString extracts a string claim from an OIDC claim map.
func ClaimString(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// --- JWKS handling (RS256 public keys, with a short in-memory cache) ---

type jwkKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
	Alg string `json:"alg"`
	Use string `json:"use"`
}

type jwksEntry struct {
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

const jwksTTL = time.Hour

var (
	jwksMu    sync.Mutex
	jwksStore = map[string]jwksEntry{}
)

// PickKey selects a public key by kid, or the sole key when kid is empty.
func PickKey(keys map[string]*rsa.PublicKey, kid string) *rsa.PublicKey {
	if kid != "" {
		return keys[kid]
	}
	if len(keys) == 1 {
		for _, k := range keys {
			return k
		}
	}
	return nil
}

// FetchJWKSKeys fetches (and caches) the RSA signing keys at uri.
func FetchJWKSKeys(ctx context.Context, uri string, force bool) (map[string]*rsa.PublicKey, error) {
	jwksMu.Lock()
	entry, ok := jwksStore[uri]
	jwksMu.Unlock()
	if ok && !force && time.Since(entry.fetchedAt) < jwksTTL {
		return entry.keys, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := HTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jwks status %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var doc struct {
		Keys []jwkKey `json:"keys"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, err
	}
	keys := map[string]*rsa.PublicKey{}
	for _, k := range doc.Keys {
		if !strings.EqualFold(k.Kty, "RSA") {
			continue
		}
		pub, err := ParseRSAJWK(k.N, k.E)
		if err != nil {
			continue
		}
		keys[k.Kid] = pub
	}
	if len(keys) == 0 {
		return nil, errors.New("no usable RSA keys in jwks")
	}

	jwksMu.Lock()
	jwksStore[uri] = jwksEntry{keys: keys, fetchedAt: time.Now()}
	jwksMu.Unlock()
	return keys, nil
}

// ParseRSAJWK builds an RSA public key from base64url modulus/exponent strings.
func ParseRSAJWK(nStr, eStr string) (*rsa.PublicKey, error) {
	nb, err := B64URLDecode(nStr)
	if err != nil {
		return nil, err
	}
	eb, err := B64URLDecode(eStr)
	if err != nil {
		return nil, err
	}
	if len(nb) == 0 || len(eb) == 0 {
		return nil, errors.New("invalid jwk")
	}
	e := 0
	for _, b := range eb {
		e = e<<8 + int(b)
	}
	if e == 0 {
		return nil, errors.New("invalid jwk exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nb), E: e}, nil
}

// LookupKey resolves the signing key for kid from the JWKS at jwksURI, refreshing
// the cache once if the key is not initially found.
func LookupKey(ctx context.Context, jwksURI, kid string) (*rsa.PublicKey, error) {
	if strings.TrimSpace(jwksURI) == "" {
		return nil, errors.New("jwks_uri not configured")
	}
	keys, err := FetchJWKSKeys(ctx, jwksURI, false)
	if err != nil {
		return nil, err
	}
	if key := PickKey(keys, kid); key != nil {
		return key, nil
	}
	keys, err = FetchJWKSKeys(ctx, jwksURI, true)
	if err != nil {
		return nil, err
	}
	if key := PickKey(keys, kid); key != nil {
		return key, nil
	}
	return nil, errors.New("signing key not found")
}
