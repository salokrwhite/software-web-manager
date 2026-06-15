package handlers

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
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"software-web-manager/backend/internal/auth"
	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ssoConfig struct {
	Enabled           bool
	DisplayName       string
	Issuer            string
	AuthorizeEndpoint string
	TokenEndpoint     string
	UserinfoEndpoint  string
	JWKSURI           string
	ClientID          string
	ClientSecret      string
	Scopes            string
	RedirectURI       string
}

type ssoAuthState struct {
	Nonce        string `json:"nonce"`
	CodeVerifier string `json:"code_verifier"`
	Redirect     string `json:"redirect"`
	Purpose      string `json:"purpose"`
	UserID       string `json:"user_id"`
}

type ssoTokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	Scope       string `json:"scope"`
}

var errSSOUserNotProvisioned = errors.New("sso user not provisioned")

const ssoStateTTL = 10 * time.Minute

func ssoStateKey(state string) string { return "swm:sso:state:" + state }

func (h *Handler) getSSOConfig() (ssoConfig, error) {
	if !h.hasSystemSettingsTable() {
		return ssoConfig{Enabled: defaultSSOEnabled, DisplayName: defaultSSODisplayName, Scopes: defaultSSOScopes}, nil
	}
	items, err := h.listSystemSettings()
	if err != nil {
		return ssoConfig{}, err
	}
	return ssoConfig{
		Enabled:           getBoolSetting(items, systemSettingSSOEnabledKey, defaultSSOEnabled),
		DisplayName:       getStringSetting(items, systemSettingSSODisplayNameKey, defaultSSODisplayName),
		Issuer:            getStringSetting(items, systemSettingSSOIssuerKey, ""),
		AuthorizeEndpoint: getStringSetting(items, systemSettingSSOAuthorizeEndpointKey, ""),
		TokenEndpoint:     getStringSetting(items, systemSettingSSOTokenEndpointKey, ""),
		UserinfoEndpoint:  getStringSetting(items, systemSettingSSOUserinfoEndpointKey, ""),
		JWKSURI:           getStringSetting(items, systemSettingSSOJWKSURIKey, ""),
		ClientID:          getStringSetting(items, systemSettingSSOClientIDKey, ""),
		ClientSecret:      getStringSetting(items, systemSettingSSOClientSecretKey, ""),
		Scopes:            getStringSetting(items, systemSettingSSOScopesKey, defaultSSOScopes),
		RedirectURI:       getStringSetting(items, systemSettingSSORedirectURIKey, ""),
	}, nil
}

type ssoDiscoverRequest struct {
	DiscoveryURL string `json:"discovery_url" binding:"required"`
}

type ssoDiscoveryDoc struct {
	Issuer                string   `json:"issuer"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	TokenEndpoint         string   `json:"token_endpoint"`
	UserinfoEndpoint      string   `json:"userinfo_endpoint"`
	JWKSURI               string   `json:"jwks_uri"`
	ScopesSupported       []string `json:"scopes_supported"`
}

func (h *Handler) SSODiscover(c *gin.Context) {
	var req ssoDiscoverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "discovery_url required"})
		return
	}
	raw := strings.TrimSpace(req.DiscoveryURL)
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid discovery_url"})
		return
	}
	if !strings.Contains(u.Path, "/.well-known/") {
		u.Path = strings.TrimRight(u.Path, "/") + "/.well-known/openid-configuration"
		u.RawQuery = ""
		raw = u.String()
	}

	httpReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, raw, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid discovery_url"})
		return
	}
	httpReq.Header.Set("Accept", "application/json")
	resp, err := ssoHTTPClient().Do(httpReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "discovery_fetch_failed"})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{"error": "discovery_fetch_failed"})
		return
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var doc ssoDiscoveryDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "discovery_parse_failed"})
		return
	}
	if strings.TrimSpace(doc.AuthorizationEndpoint) == "" || strings.TrimSpace(doc.TokenEndpoint) == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "discovery_parse_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"issuer":             doc.Issuer,
		"authorize_endpoint": doc.AuthorizationEndpoint,
		"token_endpoint":     doc.TokenEndpoint,
		"userinfo_endpoint":  doc.UserinfoEndpoint,
		"jwks_uri":           doc.JWKSURI,
		"scopes_supported":   doc.ScopesSupported,
	})
}

func (h *Handler) SSOLogin(c *gin.Context) {
	cfg, err := h.getSSOConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load sso config"})
		return
	}
	if !cfg.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sso_disabled"})
		return
	}
	if cfg.AuthorizeEndpoint == "" || cfg.TokenEndpoint == "" || cfg.ClientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sso_not_configured"})
		return
	}
	if h.ReplayStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "sso_store_unavailable"})
		return
	}

	state := randomToken(24)
	nonce := randomToken(24)
	verifier := randomToken(48)
	challenge := pkceChallenge(verifier)

	payload, err := json.Marshal(ssoAuthState{
		Nonce:        nonce,
		CodeVerifier: verifier,
		Redirect:     sanitizeSSORedirect(c.Query("redirect")),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to prepare sso state"})
		return
	}
	if err := h.ReplayStore.Set(c.Request.Context(), ssoStateKey(state), string(payload), ssoStateTTL).Err(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "sso_store_unavailable"})
		return
	}

	q := url.Values{}
	q.Set("client_id", cfg.ClientID)
	q.Set("redirect_uri", h.ssoRedirectURI(c, cfg))
	q.Set("response_type", "code")
	q.Set("scope", cfg.Scopes)
	q.Set("state", state)
	q.Set("nonce", nonce)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	c.JSON(http.StatusOK, gin.H{"authorize_url": cfg.AuthorizeEndpoint + "?" + q.Encode()})
}

func (h *Handler) SSOCallback(c *gin.Context) {
	cfg, cfgErr := h.getSSOConfig()
	frontendBase := h.ssoFrontendBase(c, cfg)
	if cfgErr != nil {
		h.redirectSSOError(c, frontendBase, "sso_error")
		return
	}
	if !cfg.Enabled {
		h.redirectSSOError(c, frontendBase, "sso_disabled")
		return
	}
	if e := strings.TrimSpace(c.Query("error")); e != "" {
		h.redirectSSOError(c, frontendBase, e)
		return
	}

	state := strings.TrimSpace(c.Query("state"))
	code := strings.TrimSpace(c.Query("code"))
	if state == "" || code == "" {
		h.redirectSSOError(c, frontendBase, "sso_invalid_request")
		return
	}
	if h.ReplayStore == nil {
		h.redirectSSOError(c, frontendBase, "sso_store_unavailable")
		return
	}

	raw, err := h.ReplayStore.GetDel(c.Request.Context(), ssoStateKey(state)).Result()
	if err != nil || strings.TrimSpace(raw) == "" {
		h.redirectSSOError(c, frontendBase, "sso_state_expired")
		return
	}
	var st ssoAuthState
	if err := json.Unmarshal([]byte(raw), &st); err != nil {
		h.redirectSSOError(c, frontendBase, "sso_state_invalid")
		return
	}

	tok, err := h.ssoExchangeCode(c.Request.Context(), cfg, code, st.CodeVerifier, h.ssoRedirectURI(c, cfg))
	if err != nil {
		h.redirectSSOError(c, frontendBase, "sso_token_exchange_failed")
		return
	}

	claims, err := h.ssoVerifyIDToken(c.Request.Context(), cfg, tok.IDToken, st.Nonce)
	if err != nil {
		h.redirectSSOError(c, frontendBase, "sso_id_token_invalid")
		return
	}

	sub := strings.TrimSpace(claimString(claims, "sub"))
	email := strings.ToLower(strings.TrimSpace(claimString(claims, "email")))
	if email == "" && cfg.UserinfoEndpoint != "" && tok.AccessToken != "" {
		if ui, uiErr := h.ssoFetchUserinfo(c.Request.Context(), cfg, tok.AccessToken); uiErr == nil {
			if sub == "" {
				sub = strings.TrimSpace(claimString(ui, "sub"))
			}
			email = strings.ToLower(strings.TrimSpace(claimString(ui, "email")))
		}
	}
	if sub == "" {
		h.redirectSSOError(c, frontendBase, "sso_missing_sub")
		return
	}

	if st.Purpose == "bind" {
		h.ssoCompleteBind(c, frontendBase, st, sub)
		return
	}

	user, err := h.ssoResolveUser(sub, email)
	if err != nil {
		if errors.Is(err, errSSOUserNotProvisioned) {
			h.redirectSSOError(c, frontendBase, "sso_account_not_provisioned")
			return
		}
		h.redirectSSOError(c, frontendBase, "sso_error")
		return
	}
	if strings.ToLower(strings.TrimSpace(user.Status)) != "active" {
		h.redirectSSOError(c, frontendBase, userStatusCode(user.Status))
		return
	}

	sess, blockCode, err := h.buildUserSession(user)
	if err != nil {
		h.redirectSSOError(c, frontendBase, "sso_error")
		return
	}
	if blockCode != "" {
		h.redirectSSOError(c, frontendBase, blockCode)
		return
	}

	frag := url.Values{}
	frag.Set("access_token", sess.tokens.AccessToken)
	frag.Set("refresh_token", sess.tokens.RefreshToken)
	frag.Set("expires_in", strconv.FormatInt(sess.tokens.ExpiresIn, 10))
	frag.Set("system_role", sess.systemRole)
	frag.Set("org_id", sess.orgID)
	frag.Set("role", sess.role)
	frag.Set("org_type", sess.orgType)
	frag.Set("email", user.Email)
	frag.Set("redirect", st.Redirect)
	c.Redirect(http.StatusFound, frontendBase+"/sso/callback#"+frag.Encode())
}

func (h *Handler) ssoResolveUser(sub, email string) (models.User, error) {
	var user models.User
	if sub != "" {
		if err := h.DB.Where("sso_sub = ?", sub).First(&user).Error; err == nil {
			return user, nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return models.User{}, err
		}
	}
	if email == "" {
		return models.User{}, errSSOUserNotProvisioned
	}
	if err := h.DB.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.User{}, errSSOUserNotProvisioned
		}
		return models.User{}, err
	}
	if sub != "" && (user.SSOSub == nil || strings.TrimSpace(*user.SSOSub) != sub) {
		subCopy := sub
		if err := h.DB.Model(&models.User{}).Where("id = ?", user.ID).Update("sso_sub", &subCopy).Error; err != nil {
			return models.User{}, err
		}
		user.SSOSub = &subCopy
	}
	return user, nil
}

func (h *Handler) SSOBindStart(c *gin.Context) {
	cfg, err := h.getSSOConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load sso config"})
		return
	}
	if !cfg.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sso_disabled"})
		return
	}
	if cfg.AuthorizeEndpoint == "" || cfg.TokenEndpoint == "" || cfg.ClientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sso_not_configured"})
		return
	}
	if h.ReplayStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "sso_store_unavailable"})
		return
	}
	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	state := randomToken(24)
	nonce := randomToken(24)
	verifier := randomToken(48)
	challenge := pkceChallenge(verifier)

	payload, err := json.Marshal(ssoAuthState{
		Nonce:        nonce,
		CodeVerifier: verifier,
		Redirect:     sanitizeSSORedirect(c.Query("redirect")),
		Purpose:      "bind",
		UserID:       userID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to prepare sso state"})
		return
	}
	if err := h.ReplayStore.Set(c.Request.Context(), ssoStateKey(state), string(payload), ssoStateTTL).Err(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "sso_store_unavailable"})
		return
	}

	q := url.Values{}
	q.Set("client_id", cfg.ClientID)
	q.Set("redirect_uri", h.ssoRedirectURI(c, cfg))
	q.Set("response_type", "code")
	q.Set("scope", cfg.Scopes)
	q.Set("state", state)
	q.Set("nonce", nonce)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	c.JSON(http.StatusOK, gin.H{"authorize_url": cfg.AuthorizeEndpoint + "?" + q.Encode()})
}

func (h *Handler) ssoCompleteBind(c *gin.Context, frontendBase string, st ssoAuthState, sub string) {
	userID := strings.TrimSpace(st.UserID)
	if userID == "" {
		h.redirectSSOError(c, frontendBase, "sso_invalid_request")
		return
	}
	var existing models.User
	err := h.DB.Where("sso_sub = ?", sub).First(&existing).Error
	if err == nil && existing.ID.String() != userID {
		h.redirectSSOError(c, frontendBase, "sso_already_bound")
		return
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		h.redirectSSOError(c, frontendBase, "sso_error")
		return
	}
	subCopy := sub
	if err := h.DB.Model(&models.User{}).Where("id = ?", userID).Update("sso_sub", &subCopy).Error; err != nil {
		h.redirectSSOError(c, frontendBase, "sso_error")
		return
	}

	frag := url.Values{}
	frag.Set("sso_bound", "1")
	frag.Set("redirect", st.Redirect)
	c.Redirect(http.StatusFound, frontendBase+"/sso/callback#"+frag.Encode())
}

func (h *Handler) SSOUnbind(c *gin.Context) {
	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	if err := h.DB.Model(&models.User{}).Where("id = ?", userID).Update("sso_sub", nil).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unbind sso"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) ssoExchangeCode(ctx context.Context, cfg ssoConfig, code, verifier, redirectURI string) (ssoTokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("code_verifier", verifier)
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return ssoTokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := ssoHTTPClient().Do(req)
	if err != nil {
		return ssoTokenResponse{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return ssoTokenResponse{}, fmt.Errorf("token endpoint status %d", resp.StatusCode)
	}
	var tok ssoTokenResponse
	if err := json.Unmarshal(body, &tok); err != nil {
		return ssoTokenResponse{}, err
	}
	if strings.TrimSpace(tok.IDToken) == "" {
		return ssoTokenResponse{}, errors.New("missing id_token")
	}
	return tok, nil
}

func (h *Handler) ssoFetchUserinfo(ctx context.Context, cfg ssoConfig, accessToken string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.UserinfoEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := ssoHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo status %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (h *Handler) ssoVerifyIDToken(ctx context.Context, cfg ssoConfig, idToken, nonce string) (jwt.MapClaims, error) {
	if strings.TrimSpace(idToken) == "" {
		return nil, errors.New("empty id_token")
	}
	keyFunc := func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		kid, _ := t.Header["kid"].(string)
		return h.ssoLookupKey(ctx, cfg.JWKSURI, kid)
	}
	claims := jwt.MapClaims{}
	if _, err := jwt.ParseWithClaims(idToken, claims, keyFunc, jwt.WithValidMethods([]string{"RS256"})); err != nil {
		return nil, err
	}
	if cfg.Issuer != "" {
		if iss, _ := claims["iss"].(string); iss != cfg.Issuer {
			return nil, errors.New("issuer mismatch")
		}
	}
	if !audienceContains(claims["aud"], cfg.ClientID) {
		return nil, errors.New("audience mismatch")
	}
	if n, _ := claims["nonce"].(string); n != nonce {
		return nil, errors.New("nonce mismatch")
	}
	return claims, nil
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

func (h *Handler) ssoLookupKey(ctx context.Context, jwksURI, kid string) (*rsa.PublicKey, error) {
	if strings.TrimSpace(jwksURI) == "" {
		return nil, errors.New("jwks_uri not configured")
	}
	keys, err := fetchJWKSKeys(ctx, jwksURI, false)
	if err != nil {
		return nil, err
	}
	if key := pickKey(keys, kid); key != nil {
		return key, nil
	}
	keys, err = fetchJWKSKeys(ctx, jwksURI, true)
	if err != nil {
		return nil, err
	}
	if key := pickKey(keys, kid); key != nil {
		return key, nil
	}
	return nil, errors.New("signing key not found")
}

func pickKey(keys map[string]*rsa.PublicKey, kid string) *rsa.PublicKey {
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

func fetchJWKSKeys(ctx context.Context, uri string, force bool) (map[string]*rsa.PublicKey, error) {
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
	resp, err := ssoHTTPClient().Do(req)
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
		pub, err := parseRSAJWK(k.N, k.E)
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

func parseRSAJWK(nStr, eStr string) (*rsa.PublicKey, error) {
	nb, err := b64urlDecode(nStr)
	if err != nil {
		return nil, err
	}
	eb, err := b64urlDecode(eStr)
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

func ssoHTTPClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

func (h *Handler) ssoFrontendBase(c *gin.Context, cfg ssoConfig) string {
	if base := strings.TrimRight(strings.TrimSpace(h.Cfg.WebBaseURL), "/"); base != "" {
		return base
	}
	if cfg.RedirectURI != "" {
		if u, err := url.Parse(cfg.RedirectURI); err == nil && u.Scheme != "" && u.Host != "" {
			return u.Scheme + "://" + u.Host
		}
	}
	return requestScheme(c) + "://" + requestHost(c)
}

func (h *Handler) ssoRedirectURI(c *gin.Context, cfg ssoConfig) string {
	if v := strings.TrimSpace(cfg.RedirectURI); v != "" {
		return v
	}
	return ssoDeriveRedirectURI(c)
}

func ssoDeriveRedirectURI(c *gin.Context) string {
	return requestScheme(c) + "://" + requestHost(c) + "/api/auth/sso/callback"
}

func requestScheme(c *gin.Context) string {
	if proto := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); proto != "" {
		return strings.TrimSpace(strings.Split(proto, ",")[0])
	}
	if c.Request != nil && c.Request.TLS != nil {
		return "https"
	}
	return "http"
}

func requestHost(c *gin.Context) string {
	if host := strings.TrimSpace(c.GetHeader("X-Forwarded-Host")); host != "" {
		return strings.TrimSpace(strings.Split(host, ",")[0])
	}
	return c.Request.Host
}

func (h *Handler) redirectSSOError(c *gin.Context, frontendBase, code string) {
	frag := url.Values{}
	frag.Set("error", code)
	c.Redirect(http.StatusFound, frontendBase+"/sso/callback#"+frag.Encode())
}

func sanitizeSSORedirect(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || !strings.HasPrefix(v, "/") || strings.HasPrefix(v, "//") {
		return "/dashboard"
	}
	if strings.HasPrefix(v, "/login") || strings.HasPrefix(v, "/admin-login") {
		return "/dashboard"
	}
	return v
}

func randomToken(n int) string {
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

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func b64urlDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(strings.TrimRight(strings.TrimSpace(s), "="))
}

func audienceContains(aud interface{}, clientID string) bool {
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

func claimString(m map[string]interface{}, key string) string {
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

type ssoSession struct {
	tokens     auth.TokenPair
	orgID      string
	role       string
	systemRole string
	orgType    string
}

func (h *Handler) buildUserSession(user models.User) (ssoSession, string, error) {
	systemRole, err := h.resolveSystemRole(user.ID.String(), user.SystemRole)
	if err != nil {
		return ssoSession{}, "", err
	}

	if systemRole == "system_admin" {
		tokens, err := auth.IssueTokens(h.Cfg.JWTSecret, h.Cfg.JWTIssuer, user.ID.String(), "", "", systemRole, h.Cfg.AccessTokenMinutes, h.Cfg.RefreshTokenHours)
		if err != nil {
			return ssoSession{}, "", err
		}
		return ssoSession{tokens: tokens, systemRole: systemRole}, "", nil
	}

	// Enterprise admins (org_admin) are locked to their enterprise org and cannot
	// switch orgs, so bind the SSO session straight to that org (mirroring
	// AdminLogin) instead of falling through to the personal-org branch below.
	if systemRole == "org_admin" {
		var member models.OrgMember
		memberLoaded := false
		if h.hasOrgTypeColumn() {
			if err := h.DB.Raw(`
				SELECT m.scope_id, m.user_id, m.role, m.created_at
				FROM memberships m
				JOIN orgs o ON o.id = m.scope_id
				WHERE m.scope_type = 'org' AND m.user_id = ? AND COALESCE(o.org_type, '') <> 'personal'
				ORDER BY o.created_at DESC
				LIMIT 1
			`, user.ID).Scan(&member).Error; err != nil {
				return ssoSession{}, "", err
			}
			if member.OrgID != (uuid.UUID{}) {
				memberLoaded = true
			}
		}
		if !memberLoaded {
			if err := h.DB.Where("scope_type = ? AND user_id = ?", models.ScopeOrg, user.ID).First(&member).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ssoSession{}, "user_no_org", nil
				}
				return ssoSession{}, "", err
			}
		}
		orgType := ""
		var org models.Org
		if err := h.DB.Where("id = ?", member.OrgID).First(&org).Error; err == nil {
			if strings.ToLower(strings.TrimSpace(org.Status)) != "active" {
				return ssoSession{}, orgStatusCode(org.Status), nil
			}
			orgType = org.OrgType
		}
		effectiveRole := h.resolveEffectiveOrgRole(member.OrgID.String(), member.Role)
		tokens, err := auth.IssueTokens(h.Cfg.JWTSecret, h.Cfg.JWTIssuer, user.ID.String(), member.OrgID.String(), effectiveRole, systemRole, h.Cfg.AccessTokenMinutes, h.Cfg.RefreshTokenHours)
		if err != nil {
			return ssoSession{}, "", err
		}
		return ssoSession{tokens: tokens, orgID: member.OrgID.String(), role: effectiveRole, systemRole: systemRole, orgType: orgType}, "", nil
	}

	if h.hasOrgTypeColumn() {
		personalOrg, personalMember, err := h.ensurePersonalOrgMember(user.ID.String())
		if err != nil {
			return ssoSession{}, "", err
		}
		if personalOrg.ID != (uuid.UUID{}) {
			if strings.ToLower(strings.TrimSpace(personalOrg.Status)) != "active" {
				return ssoSession{}, orgStatusCode(personalOrg.Status), nil
			}
			effectiveRole := h.resolveEffectiveOrgRole(personalMember.OrgID.String(), personalMember.Role)
			tokens, err := auth.IssueTokens(h.Cfg.JWTSecret, h.Cfg.JWTIssuer, user.ID.String(), personalMember.OrgID.String(), effectiveRole, systemRole, h.Cfg.AccessTokenMinutes, h.Cfg.RefreshTokenHours)
			if err != nil {
				return ssoSession{}, "", err
			}
			return ssoSession{tokens: tokens, orgID: personalMember.OrgID.String(), role: effectiveRole, systemRole: systemRole, orgType: personalOrg.OrgType}, "", nil
		}
	}

	var member models.OrgMember
	if err := h.DB.Where("scope_type = ? AND user_id = ?", models.ScopeOrg, user.ID).First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ssoSession{}, "user_no_org", nil
		}
		return ssoSession{}, "", err
	}
	orgType := ""
	var org models.Org
	if err := h.DB.Where("id = ?", member.OrgID).First(&org).Error; err == nil {
		if strings.ToLower(strings.TrimSpace(org.Status)) != "active" {
			return ssoSession{}, orgStatusCode(org.Status), nil
		}
		orgType = org.OrgType
	}
	effectiveRole := h.resolveEffectiveOrgRole(member.OrgID.String(), member.Role)
	tokens, err := auth.IssueTokens(h.Cfg.JWTSecret, h.Cfg.JWTIssuer, user.ID.String(), member.OrgID.String(), effectiveRole, systemRole, h.Cfg.AccessTokenMinutes, h.Cfg.RefreshTokenHours)
	if err != nil {
		return ssoSession{}, "", err
	}
	return ssoSession{tokens: tokens, orgID: member.OrgID.String(), role: effectiveRole, systemRole: systemRole, orgType: orgType}, "", nil
}
