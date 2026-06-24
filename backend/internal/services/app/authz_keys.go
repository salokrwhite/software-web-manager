package app

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"software-web-manager/backend/internal/crypto"
	"software-web-manager/backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Authz key lifecycle states. Exactly one key per app is `active` at a time; the
// server signs verdicts with it. `pending` keys are published to clients before
// the server starts signing with them (zero-downtime rotation); `retired` keys
// are superseded or revoked.
const (
	AuthzKeyStatusPending = "pending"
	AuthzKeyStatusActive  = "active"
	AuthzKeyStatusRetired = "retired"

	AuthzKeyAlgorithmEd25519 = "ed25519"
)

// GenerateAuthzKeypair creates a fresh Ed25519 keypair, returning the 32-byte
// seed and the public key, both hex-encoded. The seed is what gets encrypted and
// stored; the public key is handed to the developer to embed in their client.
func GenerateAuthzKeypair() (seedHex string, publicHex string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	return hex.EncodeToString(priv.Seed()), hex.EncodeToString(pub), nil
}

// PublicKeyHexFromSeed derives the Ed25519 public key (hex) from a seed. The seed
// may be hex or base64, in 32-byte seed or 64-byte full-key form (matching
// auth.ParseEd25519PrivateKey's tolerance), so it works for both freshly
// generated keys and the platform key reused during backfill.
func PublicKeyHexFromSeed(encodedSeed string) (string, error) {
	raw, err := decodeSeedMaterial(encodedSeed)
	if err != nil {
		return "", err
	}
	var priv ed25519.PrivateKey
	switch len(raw) {
	case ed25519.SeedSize:
		priv = ed25519.NewKeyFromSeed(raw)
	case ed25519.PrivateKeySize:
		priv = ed25519.PrivateKey(raw)
	default:
		return "", fmt.Errorf("invalid ed25519 key length: %d", len(raw))
	}
	pub, ok := priv.Public().(ed25519.PublicKey)
	if !ok {
		return "", fmt.Errorf("failed to derive ed25519 public key")
	}
	return hex.EncodeToString(pub), nil
}

// NormalizeSeedHex returns the canonical 32-byte hex seed for an Ed25519 key
// given in hex/base64, seed or full form. Used so every stored ciphertext holds
// a consistent hex seed regardless of how the input key was encoded.
func NormalizeSeedHex(encodedSeed string) (string, error) {
	raw, err := decodeSeedMaterial(encodedSeed)
	if err != nil {
		return "", err
	}
	switch len(raw) {
	case ed25519.SeedSize:
		return hex.EncodeToString(raw), nil
	case ed25519.PrivateKeySize:
		return hex.EncodeToString(ed25519.PrivateKey(raw).Seed()), nil
	default:
		return "", fmt.Errorf("invalid ed25519 key length: %d", len(raw))
	}
}

func decodeSeedMaterial(encoded string) ([]byte, error) {
	v := strings.TrimSpace(encoded)
	if v == "" {
		return nil, fmt.Errorf("empty key material")
	}
	if len(v)%2 == 0 {
		if b, err := hex.DecodeString(v); err == nil {
			return b, nil
		}
	}
	// base64 fallthrough handled by callers that only ever pass hex would be
	// unreachable; keep hex-only here since both generated and platform seeds are
	// stored/derived as hex.
	return nil, fmt.Errorf("key material is not valid hex")
}

// DefaultAuthzKeyID builds the default public key id for an app: a stable,
// app-scoped, date-stamped identifier with a short random suffix so repeated
// creates on the same day never collide on the (app_id, key_id) unique index.
func DefaultAuthzKeyID(appID uuid.UUID) string {
	suffix := make([]byte, 2)
	_, _ = rand.Read(suffix)
	return fmt.Sprintf("app-%s-%s-%s", appID.String()[:8], time.Now().UTC().Format("20060102"), hex.EncodeToString(suffix))
}

// BuildAuthzKeyRow constructs (without persisting) an AppAuthzKey from a seed,
// encrypting the seed with masterKey. When status is active, ActivatedAt is set.
func BuildAuthzKeyRow(appID uuid.UUID, keyID, seedHex, publicHex, status, masterKey string) (models.AppAuthzKey, error) {
	cipher, err := crypto.EncryptAppSecret(masterKey, seedHex)
	if err != nil {
		return models.AppAuthzKey{}, err
	}
	row := models.AppAuthzKey{
		AppID:                appID,
		KeyID:                keyID,
		Algorithm:            AuthzKeyAlgorithmEd25519,
		PrivateKeyCiphertext: cipher,
		PublicKey:            publicHex,
		Status:               status,
	}
	if status == AuthzKeyStatusActive {
		now := time.Now()
		row.ActivatedAt = &now
	}
	return row, nil
}

// ProvisionActiveAuthzKey generates a fresh independent keypair and inserts it as
// the app's active authz key using tx. Used to auto-provision a key when a new
// app is created so it is isolated from day one.
func ProvisionActiveAuthzKey(tx *gorm.DB, masterKey string, appID uuid.UUID) (models.AppAuthzKey, error) {
	seedHex, pubHex, err := GenerateAuthzKeypair()
	if err != nil {
		return models.AppAuthzKey{}, err
	}
	row, err := BuildAuthzKeyRow(appID, DefaultAuthzKeyID(appID), seedHex, pubHex, AuthzKeyStatusActive, masterKey)
	if err != nil {
		return row, err
	}
	if err := tx.Create(&row).Error; err != nil {
		return row, err
	}
	return row, nil
}

// ActiveAuthzKey returns the current active, non-revoked authz key for the app,
// or gorm.ErrRecordNotFound when the app has none.
func (s *Service) ActiveAuthzKey(appID uuid.UUID) (models.AppAuthzKey, error) {
	var key models.AppAuthzKey
	err := s.DB.
		Where("app_id = ? AND status = ? AND revoked_at IS NULL", appID, AuthzKeyStatusActive).
		Order("activated_at DESC").
		First(&key).Error
	return key, err
}

// CountActiveAuthzKeys counts the app's active, non-revoked authz keys. Used to
// refuse revoking the only active key when there is no platform fallback.
func (s *Service) CountActiveAuthzKeys(appID uuid.UUID) (int64, error) {
	var n int64
	err := s.DB.Model(&models.AppAuthzKey{}).
		Where("app_id = ? AND status = ? AND revoked_at IS NULL", appID, AuthzKeyStatusActive).
		Count(&n).Error
	return n, err
}
