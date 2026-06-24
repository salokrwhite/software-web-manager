package auth

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Authz is the device-bound authorization verdict that the server signs and the
// client MUST verify before allowing the protected app to run. The signature is
// produced over a fixed canonical string (see authzCanonical) using an Ed25519
// private key held only on the server; the client verifies it with the matching
// embedded public key. Because the signature binds the client-supplied nonce and
// the device id, a captured response cannot be replayed on another launch or
// reused on another machine, and a fake offline server cannot forge it.
const (
	AuthzVersion       = "authz_v1"
	AuthzDecisionAllow = "allow"
	AuthzDecisionDeny  = "deny"
)

// AuthzClaims carries everything covered by the signature.
type AuthzClaims struct {
	AppID     string `json:"-"`
	DeviceID  string `json:"device_id"`
	Nonce     string `json:"nonce"`
	Decision  string `json:"decision"`
	Reason    string `json:"reason,omitempty"`
	IssuedAt  int64  `json:"issued_at"`
	ExpiresAt int64  `json:"expires_at"`
	KeyID     string `json:"key_id"`
}

// AuthzEnvelope is the JSON object embedded in client responses.
type AuthzEnvelope struct {
	Decision  string `json:"decision"`
	Nonce     string `json:"nonce"`
	DeviceID  string `json:"device_id"`
	IssuedAt  int64  `json:"issued_at"`
	ExpiresAt int64  `json:"expires_at"`
	KeyID     string `json:"key_id"`
	Reason    string `json:"reason,omitempty"`
	Signature string `json:"signature"`
}

// authzCanonical builds the exact byte string that gets signed/verified. The
// field order and formatting MUST stay identical on both the server (here) and
// the client verifier, or every signature will fail.
func authzCanonical(c AuthzClaims) string {
	return strings.Join([]string{
		AuthzVersion,
		"app_id:" + c.AppID,
		"device_id:" + c.DeviceID,
		"nonce:" + c.Nonce,
		"decision:" + c.Decision,
		"reason:" + c.Reason,
		"issued_at:" + strconv.FormatInt(c.IssuedAt, 10),
		"expires_at:" + strconv.FormatInt(c.ExpiresAt, 10),
		"key_id:" + c.KeyID,
	}, "\n")
}

// ParseEd25519PrivateKey accepts an Ed25519 private key encoded as hex or
// standard base64, in either the 32-byte seed form or the 64-byte full form.
func ParseEd25519PrivateKey(encoded string) (ed25519.PrivateKey, error) {
	raw, err := decodeKeyMaterial(encoded)
	if err != nil {
		return nil, err
	}
	switch len(raw) {
	case ed25519.SeedSize: // 32-byte seed
		return ed25519.NewKeyFromSeed(raw), nil
	case ed25519.PrivateKeySize: // 64-byte full private key
		return ed25519.PrivateKey(raw), nil
	default:
		return nil, fmt.Errorf("invalid ed25519 private key length: %d (want %d or %d)", len(raw), ed25519.SeedSize, ed25519.PrivateKeySize)
	}
}

func decodeKeyMaterial(encoded string) ([]byte, error) {
	v := strings.TrimSpace(encoded)
	if v == "" {
		return nil, fmt.Errorf("empty key material")
	}
	// Prefer hex when the string looks like hex and has even length.
	if len(v)%2 == 0 {
		if b, err := hex.DecodeString(v); err == nil {
			return b, nil
		}
	}
	if b, err := base64.StdEncoding.DecodeString(v); err == nil {
		return b, nil
	}
	if b, err := base64.RawStdEncoding.DecodeString(v); err == nil {
		return b, nil
	}
	return nil, fmt.Errorf("key material is neither valid hex nor base64")
}

// AuthzSigner signs authorization verdicts with a fixed key id.
type AuthzSigner struct {
	priv  ed25519.PrivateKey
	keyID string
}

// NewAuthzSigner parses the key and returns a ready signer.
func NewAuthzSigner(encodedPrivateKey, keyID string) (*AuthzSigner, error) {
	priv, err := ParseEd25519PrivateKey(encodedPrivateKey)
	if err != nil {
		return nil, err
	}
	kid := strings.TrimSpace(keyID)
	if kid == "" {
		return nil, fmt.Errorf("authz key id is required")
	}
	return &AuthzSigner{priv: priv, keyID: kid}, nil
}

// KeyID returns the configured key id.
func (s *AuthzSigner) KeyID() string { return s.keyID }

// PublicKeyHex returns the hex-encoded public key (useful for ops / embedding).
func (s *AuthzSigner) PublicKeyHex() string {
	pub, ok := s.priv.Public().(ed25519.PublicKey)
	if !ok {
		return ""
	}
	return hex.EncodeToString(pub)
}

// SignAllow produces a signed allow envelope bound to deviceID and nonce, valid
// for ttl. deviceID and nonce must be non-empty for a meaningful binding.
func (s *AuthzSigner) SignAllow(appID, deviceID, nonce string, ttl time.Duration) AuthzEnvelope {
	return s.sign(appID, deviceID, nonce, AuthzDecisionAllow, "", ttl)
}

// SignDeny produces a signed deny envelope with a reason.
func (s *AuthzSigner) SignDeny(appID, deviceID, nonce, reason string, ttl time.Duration) AuthzEnvelope {
	return s.sign(appID, deviceID, nonce, AuthzDecisionDeny, reason, ttl)
}

func (s *AuthzSigner) sign(appID, deviceID, nonce, decision, reason string, ttl time.Duration) AuthzEnvelope {
	now := time.Now().Unix()
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	claims := AuthzClaims{
		AppID:     appID,
		DeviceID:  deviceID,
		Nonce:     nonce,
		Decision:  decision,
		Reason:    reason,
		IssuedAt:  now,
		ExpiresAt: now + int64(ttl.Seconds()),
		KeyID:     s.keyID,
	}
	sig := ed25519.Sign(s.priv, []byte(authzCanonical(claims)))
	return AuthzEnvelope{
		Decision:  claims.Decision,
		Nonce:     claims.Nonce,
		DeviceID:  claims.DeviceID,
		IssuedAt:  claims.IssuedAt,
		ExpiresAt: claims.ExpiresAt,
		KeyID:     claims.KeyID,
		Reason:    claims.Reason,
		Signature: base64.StdEncoding.EncodeToString(sig),
	}
}
