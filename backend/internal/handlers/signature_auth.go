package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"
	"software-web-manager/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	signHeaderAppID     = "X-App-Id"
	signHeaderTimestamp = "X-Timestamp"
	signHeaderNonce     = "X-Nonce"
	signHeaderSignature = "X-Signature"
	signHeaderVersion   = "X-Sign-Version"
	signVersionV1       = "v1"
	signWindowSeconds   = int64(300)

	ContextClientApp = "client_app"
	ContextClientOrg = "client_org"
	ContextClientScopes = "client_scopes"
	ContextClientSecretID = "client_secret_id"
)

type signatureHeaders struct {
	AppID     string
	Timestamp int64
	Nonce     string
	Signature string
	Version   string
}

type clientSecretCandidate struct {
	ID     uuid.UUID
	Secret string
	Scopes []string
	ExpiresAt *time.Time
}

func signatureError(c *gin.Context, status int, code string, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}

func parseSignatureHeaders(c *gin.Context, requireAppID bool) (*signatureHeaders, bool) {
	appID := strings.TrimSpace(c.GetHeader(signHeaderAppID))
	tsRaw := strings.TrimSpace(c.GetHeader(signHeaderTimestamp))
	nonce := strings.TrimSpace(c.GetHeader(signHeaderNonce))
	signature := strings.TrimSpace(c.GetHeader(signHeaderSignature))
	version := strings.TrimSpace(c.GetHeader(signHeaderVersion))

	if requireAppID {
		if appID == "" || tsRaw == "" || nonce == "" || signature == "" || version == "" {
			signatureError(c, http.StatusUnauthorized, "signature_missing", "signature headers missing")
			return nil, false
		}
	} else {
		if tsRaw == "" || nonce == "" || signature == "" || version == "" {
			signatureError(c, http.StatusUnauthorized, "signature_missing", "signature headers missing")
			return nil, false
		}
	}

	if !strings.EqualFold(version, signVersionV1) {
		signatureError(c, http.StatusUnauthorized, "signature_version_unsupported", "unsupported signature version")
		return nil, false
	}

	ts, err := strconv.ParseInt(tsRaw, 10, 64)
	if err != nil {
		signatureError(c, http.StatusUnauthorized, "signature_invalid", "invalid timestamp")
		return nil, false
	}
	now := time.Now().Unix()
	if absInt64(now-ts) > signWindowSeconds {
		signatureError(c, http.StatusUnauthorized, "signature_timestamp_expired", "signature timestamp expired")
		return nil, false
	}

	if _, err := uuid.Parse(nonce); err != nil {
		signatureError(c, http.StatusUnauthorized, "signature_invalid", "invalid nonce")
		return nil, false
	}

	return &signatureHeaders{
		AppID:     appID,
		Timestamp: ts,
		Nonce:     nonce,
		Signature: strings.ToLower(signature),
		Version:   strings.ToLower(version),
	}, true
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func readBodySHA256Hex(c *gin.Context) (string, error) {
	if c.Request == nil || c.Request.Body == nil {
		sum := sha256.Sum256(nil)
		return hex.EncodeToString(sum[:]), nil
	}
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return "", err
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(raw))
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func queryEscapeRFC3986(input string) string {
	s := url.QueryEscape(input)
	s = strings.ReplaceAll(s, "+", "%20")
	s = strings.ReplaceAll(s, "*", "%2A")
	return s
}

func buildCanonicalQuery(values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys)*2)
	for _, k := range keys {
		vs := append([]string{}, values[k]...)
		sort.Strings(vs)
		ek := queryEscapeRFC3986(k)
		for _, v := range vs {
			parts = append(parts, ek+"="+queryEscapeRFC3986(v))
		}
	}
	return strings.Join(parts, "&")
}

func buildCanonicalString(req *http.Request, bodySHA256 string, timestamp int64, nonce string, identity string) string {
	path := ""
	query := ""
	method := ""
	if req != nil {
		path = req.URL.Path
		query = buildCanonicalQuery(req.URL.Query())
		method = strings.ToUpper(req.Method)
	}
	return strings.Join([]string{
		method,
		path,
		query,
		bodySHA256,
		strconv.FormatInt(timestamp, 10),
		nonce,
		identity,
	}, "\n")
}

func signCanonical(secret string, canonical string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil))
}

func verifyReplay(ctx context.Context, storeKey string, h *Handler) error {
	if h == nil || h.ReplayStore == nil {
		return fmt.Errorf("replay store unavailable")
	}
	ok, err := h.ReplayStore.SetNX(ctx, storeKey, "1", 5*time.Minute).Result()
	if err != nil {
		return err
	}
	if !ok {
		return gorm.ErrDuplicatedKey
	}
	return nil
}

func requiredClientScope(path string) string {
	cleanPath := strings.ToLower(strings.TrimSpace(path))
	switch {
	case strings.HasSuffix(cleanPath, "/update-check"), strings.HasSuffix(cleanPath, "/updates/stream"):
		return "update:check"
	case strings.HasSuffix(cleanPath, "/events"), strings.HasSuffix(cleanPath, "/heartbeat"), strings.HasSuffix(cleanPath, "/feedback"):
		return "event:write"
	}
	return ""
}

func (h *Handler) loadClientSecretCandidates(app models.App) ([]clientSecretCandidate, error) {
	if !h.hasAppSecretsTable() {
		return nil, fmt.Errorf("missing app_secrets table, run migration 0033_app_secrets")
	}

	candidates := make([]clientSecretCandidate, 0, 2)
	var rows []models.AppSecret
	if err := h.DB.
		Where("app_id = ? AND revoked_at IS NULL", app.ID).
		Order("created_at DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		cipher := strings.TrimSpace(rows[i].SecretCiphertext)
		if cipher == "" {
			continue
		}
		secret, err := utils.DecryptAppSecret(h.Cfg.AppSecretMasterKey, cipher)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, clientSecretCandidate{
			ID:        rows[i].ID,
			Secret:    secret,
			Scopes:    appSecretScopesFromJSON(rows[i].ScopesJSON),
			ExpiresAt: rows[i].ExpiresAt,
		})
	}
	return candidates, nil
}

func (h *Handler) requireClientSignature() gin.HandlerFunc {
	return func(c *gin.Context) {
		headers, ok := parseSignatureHeaders(c, true)
		if !ok {
			return
		}
		appID, err := uuid.Parse(headers.AppID)
		if err != nil {
			signatureError(c, http.StatusUnauthorized, "signature_invalid", "invalid app id")
			return
		}

		var app models.App
		if err := h.DB.Where("id = ?", appID).First(&app).Error; err != nil {
			signatureError(c, http.StatusUnauthorized, "signature_invalid", "invalid app id")
			return
		}
		var org models.Org
		if err := h.DB.Where("id = ?", app.OrgID).First(&org).Error; err != nil {
			signatureError(c, http.StatusUnauthorized, "signature_invalid", "invalid app org")
			return
		}
		if strings.EqualFold(strings.TrimSpace(org.OrgType), "personal") {
			status := strings.ToLower(strings.TrimSpace(app.Status))
			if status != "" && status != "active" {
				code := "app_rejected"
				if status == "pending" {
					code = "app_pending_review"
				}
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error":            code,
					"status":           status,
					"rejection_reason": app.RejectionReason,
				})
				return
			}
		}
		candidates, err := h.loadClientSecretCandidates(app)
		if err != nil {
			signatureError(c, http.StatusUnauthorized, "signature_invalid", "app secret invalid")
			return
		}
		if len(candidates) == 0 {
			signatureError(c, http.StatusUnauthorized, "signature_invalid", "app secret not configured")
			return
		}

		bodyHash, err := readBodySHA256Hex(c)
		if err != nil {
			signatureError(c, http.StatusUnauthorized, "signature_invalid", "failed to read request body")
			return
		}
		canonical := buildCanonicalString(c.Request, bodyHash, headers.Timestamp, headers.Nonce, app.ID.String())
		var matched *clientSecretCandidate
		for i := range candidates {
			expected := signCanonical(candidates[i].Secret, canonical)
			if hmac.Equal([]byte(expected), []byte(headers.Signature)) {
				matched = &candidates[i]
				break
			}
		}
		if matched == nil {
			signatureError(c, http.StatusUnauthorized, "signature_invalid", "signature mismatch")
			return
		}

		if matched.ExpiresAt != nil && !matched.ExpiresAt.After(time.Now()) {
			signatureError(c, http.StatusUnauthorized, "signature_invalid", "app secret expired")
			return
		}

		requiredScope := requiredClientScope(c.Request.URL.Path)
		if requiredScope != "" && !scopeAllows(matched.Scopes, requiredScope) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "insufficient_scope",
					"message": "insufficient scope",
				},
				"required_scope": requiredScope,
			})
			return
		}

		replayKey := fmt.Sprintf("swm:sign:app:%s:%s", app.ID, headers.Nonce)
		if err := verifyReplay(c.Request.Context(), replayKey, h); err != nil {
			if err == gorm.ErrDuplicatedKey {
				signatureError(c, http.StatusUnauthorized, "signature_nonce_replayed", "nonce replayed")
				return
			}
			signatureError(c, http.StatusServiceUnavailable, "auth_replay_store_unavailable", "replay store unavailable")
			return
		}
		if matched.ID != uuid.Nil {
			_ = h.DB.Model(&models.AppSecret{}).Where("id = ?", matched.ID).Update("last_used_at", time.Now()).Error
		}
		c.Set(ContextClientApp, app)
		c.Set(ContextClientOrg, org)
		c.Set(ContextClientScopes, matched.Scopes)
		if matched.ID != uuid.Nil {
			c.Set(ContextClientSecretID, matched.ID.String())
		}
		c.Next()
	}
}

func (h *Handler) requireJWTRequestSignature() gin.HandlerFunc {
	return func(c *gin.Context) {
		headers, ok := parseSignatureHeaders(c, false)
		if !ok {
			return
		}
		rawToken := strings.TrimSpace(c.GetString(middleware.ContextRawToken))
		userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
		if rawToken == "" || userID == "" {
			signatureError(c, http.StatusUnauthorized, "signature_invalid", "missing token context")
			return
		}

		bodyHash, err := readBodySHA256Hex(c)
		if err != nil {
			signatureError(c, http.StatusUnauthorized, "signature_invalid", "failed to read request body")
			return
		}
		canonical := buildCanonicalString(c.Request, bodyHash, headers.Timestamp, headers.Nonce, userID)
		expected := signCanonical(rawToken, canonical)
		if !hmac.Equal([]byte(expected), []byte(headers.Signature)) {
			signatureError(c, http.StatusUnauthorized, "signature_invalid", "signature mismatch")
			return
		}
		replayKey := fmt.Sprintf("swm:sign:user:%s:%s", userID, headers.Nonce)
		if err := verifyReplay(c.Request.Context(), replayKey, h); err != nil {
			if err == gorm.ErrDuplicatedKey {
				signatureError(c, http.StatusUnauthorized, "signature_nonce_replayed", "nonce replayed")
				return
			}
			signatureError(c, http.StatusServiceUnavailable, "auth_replay_store_unavailable", "replay store unavailable")
			return
		}
		c.Next()
	}
}

func clientAppOrgFromContext(c *gin.Context) (models.App, models.Org, bool) {
	appAny, ok := c.Get(ContextClientApp)
	if !ok {
		return models.App{}, models.Org{}, false
	}
	orgAny, ok := c.Get(ContextClientOrg)
	if !ok {
		return models.App{}, models.Org{}, false
	}
	app, ok := appAny.(models.App)
	if !ok {
		return models.App{}, models.Org{}, false
	}
	org, ok := orgAny.(models.Org)
	if !ok {
		return models.App{}, models.Org{}, false
	}
	return app, org, true
}
