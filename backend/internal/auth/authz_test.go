package auth

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"testing"
	"time"
)

// devSeed matches config.DevAuthzSigningKey; its public key is embedded in the
// client for local testing. Kept here as a literal to avoid an import cycle.
const devSeed = "3d6bd866a72a631ad51e4c495b6dd81062d8a36acb28dbf245bc37bfb3734b28"
const devPubHex = "b0c76d5262f54411ca3ea382b6795b8ca1f4a8a10e0b7750e58c7753893258f7"

func TestDevKeypairMatches(t *testing.T) {
	signer, err := NewAuthzSigner(devSeed, "authz-dev")
	if err != nil {
		t.Fatalf("NewAuthzSigner: %v", err)
	}
	if got := signer.PublicKeyHex(); got != devPubHex {
		t.Fatalf("public key mismatch: got %s want %s", got, devPubHex)
	}
}

func TestSignAllowVerifies(t *testing.T) {
	signer, err := NewAuthzSigner(devSeed, "authz-dev")
	if err != nil {
		t.Fatalf("NewAuthzSigner: %v", err)
	}
	env := signer.SignAllow("11111111-1111-1111-1111-111111111111", "pcid-abc", "nonce-xyz", time.Minute)

	if env.Decision != AuthzDecisionAllow {
		t.Fatalf("decision = %s", env.Decision)
	}

	// Reconstruct the canonical string exactly as the client must, then verify.
	claims := AuthzClaims{
		AppID:     "11111111-1111-1111-1111-111111111111",
		DeviceID:  env.DeviceID,
		Nonce:     env.Nonce,
		Decision:  env.Decision,
		Reason:    env.Reason,
		IssuedAt:  env.IssuedAt,
		ExpiresAt: env.ExpiresAt,
		KeyID:     env.KeyID,
	}
	pub, _ := hex.DecodeString(devPubHex)
	sig, err := base64.StdEncoding.DecodeString(env.Signature)
	if err != nil {
		t.Fatalf("decode sig: %v", err)
	}
	if !ed25519.Verify(ed25519.PublicKey(pub), []byte(authzCanonical(claims)), sig) {
		t.Fatal("signature did not verify")
	}

	// A tampered device id must break verification (binding works).
	claims.DeviceID = "pcid-evil"
	if ed25519.Verify(ed25519.PublicKey(pub), []byte(authzCanonical(claims)), sig) {
		t.Fatal("signature verified after tampering device id")
	}
}

func TestParseEd25519PrivateKeyForms(t *testing.T) {
	// 32-byte seed (hex)
	if _, err := ParseEd25519PrivateKey(devSeed); err != nil {
		t.Fatalf("seed hex: %v", err)
	}
	// 64-byte full key (hex)
	full := hex.EncodeToString(ed25519.NewKeyFromSeed(mustHex(t, devSeed)))
	if _, err := ParseEd25519PrivateKey(full); err != nil {
		t.Fatalf("full hex: %v", err)
	}
	// base64 seed
	if _, err := ParseEd25519PrivateKey(base64.StdEncoding.EncodeToString(mustHex(t, devSeed))); err != nil {
		t.Fatalf("seed base64: %v", err)
	}
	if _, err := ParseEd25519PrivateKey("not-a-key"); err == nil {
		t.Fatal("expected error for junk key")
	}
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("hex: %v", err)
	}
	return b
}
