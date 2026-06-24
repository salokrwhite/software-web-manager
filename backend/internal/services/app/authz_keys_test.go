package app

import (
	"strings"
	"testing"

	"software-web-manager/backend/internal/auth"
	"software-web-manager/backend/internal/crypto"

	"github.com/google/uuid"
)

// The public key we hand to developers must be the one that verifies signatures
// produced by a signer built from the stored seed. This ties the service helpers
// to the real auth.AuthzSigner so a stored key and its advertised public_key
// never drift apart.
func TestGenerateAuthzKeypairMatchesSigner(t *testing.T) {
	seedHex, pubHex, err := GenerateAuthzKeypair()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	derived, err := PublicKeyHexFromSeed(seedHex)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	if derived != pubHex {
		t.Fatalf("derived pub %s != generated pub %s", derived, pubHex)
	}
	signer, err := auth.NewAuthzSigner(seedHex, "kid")
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	if signer.PublicKeyHex() != pubHex {
		t.Fatalf("signer pub %s != generated pub %s", signer.PublicKeyHex(), pubHex)
	}
}

func TestNormalizeSeedHexFromFullKey(t *testing.T) {
	seedHex, _, err := GenerateAuthzKeypair()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	// Round-trip a seed-form hex unchanged.
	norm, err := NormalizeSeedHex(seedHex)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if norm != strings.ToLower(seedHex) {
		t.Fatalf("normalize changed seed hex: %s != %s", norm, seedHex)
	}
	if len(norm) != 64 {
		t.Fatalf("expected 32-byte (64 hex) seed, got len %d", len(norm))
	}
}

func TestBuildAuthzKeyRowEncryptsSeed(t *testing.T) {
	const master = "unit-test-master-key"
	seedHex, pubHex, err := GenerateAuthzKeypair()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	appID := uuid.New()

	pending, err := BuildAuthzKeyRow(appID, "kid-1", seedHex, pubHex, AuthzKeyStatusPending, master)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if pending.ActivatedAt != nil {
		t.Fatalf("pending key should not have ActivatedAt set")
	}
	if pending.PrivateKeyCiphertext == seedHex || pending.PrivateKeyCiphertext == "" {
		t.Fatalf("seed should be encrypted, not stored as plaintext")
	}
	got, err := crypto.DecryptAppSecret(master, pending.PrivateKeyCiphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if got != seedHex {
		t.Fatalf("decrypted seed %s != original %s", got, seedHex)
	}

	active, err := BuildAuthzKeyRow(appID, "kid-2", seedHex, pubHex, AuthzKeyStatusActive, master)
	if err != nil {
		t.Fatalf("build active: %v", err)
	}
	if active.ActivatedAt == nil {
		t.Fatalf("active key should have ActivatedAt set")
	}
}

func TestDefaultAuthzKeyIDUniquePerCall(t *testing.T) {
	appID := uuid.New()
	a := DefaultAuthzKeyID(appID)
	b := DefaultAuthzKeyID(appID)
	if a == b {
		t.Fatalf("expected unique key ids, got %s twice", a)
	}
	if !strings.HasPrefix(a, "app-"+appID.String()[:8]+"-") {
		t.Fatalf("unexpected key id format: %s", a)
	}
	if !validAuthzKeyIDForTest(a) {
		t.Fatalf("default key id has invalid chars: %s", a)
	}
}

// validAuthzKeyIDForTest mirrors the handler's charset rule so we assert the
// default id is always acceptable as a custom id too.
func validAuthzKeyIDForTest(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, ch := range id {
		switch {
		case ch >= 'a' && ch <= 'z', ch >= 'A' && ch <= 'Z', ch >= '0' && ch <= '9':
		case ch == '-', ch == '_', ch == '.':
		default:
			return false
		}
	}
	return true
}
