package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"software-web-manager/backend/internal/api/common"
	"software-web-manager/backend/internal/auth"
	"software-web-manager/backend/internal/db/schema"
	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"
	authsvc "software-web-manager/backend/internal/services/auth"
	orgsvc "software-web-manager/backend/internal/services/org"
	"software-web-manager/backend/internal/services/system"
	"strconv"
	"strings"
	"time"

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
	if !schema.HasSystemSettingsTable(h.DB) {
		return ssoConfig{Enabled: system.DefaultSSOEnabled, DisplayName: system.DefaultSSODisplayName, Scopes: system.DefaultSSOScopes}, nil
	}
	items, err := system.NewService(h.DB).ListSettings()
	if err != nil {
		return ssoConfig{}, err
	}
	return ssoConfig{
		Enabled:           system.GetBool(items, system.SettingSSOEnabledKey, system.DefaultSSOEnabled),
		DisplayName:       system.GetString(items, system.SettingSSODisplayNameKey, system.DefaultSSODisplayName),
		Issuer:            system.GetString(items, system.SettingSSOIssuerKey, ""),
		AuthorizeEndpoint: system.GetString(items, system.SettingSSOAuthorizeEndpointKey, ""),
		TokenEndpoint:     system.GetString(items, system.SettingSSOTokenEndpointKey, ""),
		UserinfoEndpoint:  system.GetString(items, system.SettingSSOUserinfoEndpointKey, ""),
		JWKSURI:           system.GetString(items, system.SettingSSOJWKSURIKey, ""),
		ClientID:          system.GetString(items, system.SettingSSOClientIDKey, ""),
		ClientSecret:      system.GetString(items, system.SettingSSOClientSecretKey, ""),
		Scopes:            system.GetString(items, system.SettingSSOScopesKey, system.DefaultSSOScopes),
		RedirectURI:       system.GetString(items, system.SettingSSORedirectURIKey, ""),
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
	resp, err := authsvc.HTTPClient().Do(httpReq)
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

	state := authsvc.RandomToken(24)
	nonce := authsvc.RandomToken(24)
	verifier := authsvc.RandomToken(48)
	challenge := authsvc.PKCEChallenge(verifier)

	payload, err := json.Marshal(ssoAuthState{
		Nonce:        nonce,
		CodeVerifier: verifier,
		Redirect:     authsvc.SanitizeRedirect(c.Query("redirect")),
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

// ssoEndSessionEndpoint resolves the IdP browser-logout (/oauth2/logout) URL.
// The IdP exposes it at {issuer}/oauth2/logout; if the issuer is not set we
// derive it from the authorize endpoint (…/oauth2/authorize → …/oauth2/logout).
func ssoEndSessionEndpoint(cfg ssoConfig) string {
	if iss := strings.TrimRight(strings.TrimSpace(cfg.Issuer), "/"); iss != "" {
		return iss + "/oauth2/logout"
	}
	if ae := strings.TrimSpace(cfg.AuthorizeEndpoint); ae != "" {
		if idx := strings.LastIndex(ae, "/authorize"); idx >= 0 {
			return ae[:idx] + "/logout"
		}
	}
	return ""
}

// SSOLogoutURL builds the OIDC browser single-logout URL so the frontend can
// redirect the browser to the IdP and end the SSO session, not just the local
// session. The id_token_hint (the IdP id_token kept from login) is forwarded so
// the IdP can honor post_logout_redirect_uri without an extra confirmation page.
func (h *Handler) SSOLogoutURL(c *gin.Context) {
	cfg, err := h.getSSOConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load sso config"})
		return
	}
	if !cfg.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sso_disabled"})
		return
	}
	endSession := ssoEndSessionEndpoint(cfg)
	if endSession == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sso_not_configured"})
		return
	}
	q := url.Values{}
	q.Set("post_logout_redirect_uri", h.ssoFrontendBase(c, cfg)+"/login")
	if hint := strings.TrimSpace(c.Query("id_token_hint")); hint != "" {
		q.Set("id_token_hint", hint)
	}
	c.JSON(http.StatusOK, gin.H{"logout_url": endSession + "?" + q.Encode()})
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

	sub := strings.TrimSpace(authsvc.ClaimString(claims, "sub"))
	email := strings.ToLower(strings.TrimSpace(authsvc.ClaimString(claims, "email")))
	if email == "" && cfg.UserinfoEndpoint != "" && tok.AccessToken != "" {
		if ui, uiErr := h.ssoFetchUserinfo(c.Request.Context(), cfg, tok.AccessToken); uiErr == nil {
			if sub == "" {
				sub = strings.TrimSpace(authsvc.ClaimString(ui, "sub"))
			}
			email = strings.ToLower(strings.TrimSpace(authsvc.ClaimString(ui, "email")))
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
		h.redirectSSOError(c, frontendBase, middleware.UserStatusCode(user.Status))
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
	// Keep the IdP id_token so the frontend can pass it as id_token_hint when it
	// triggers OIDC browser single-logout (/oauth2/logout) on sign-out.
	frag.Set("sso_id_token", tok.IDToken)
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

	state := authsvc.RandomToken(24)
	nonce := authsvc.RandomToken(24)
	verifier := authsvc.RandomToken(48)
	challenge := authsvc.PKCEChallenge(verifier)

	payload, err := json.Marshal(ssoAuthState{
		Nonce:        nonce,
		CodeVerifier: verifier,
		Redirect:     authsvc.SanitizeRedirect(c.Query("redirect")),
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

	resp, err := authsvc.HTTPClient().Do(req)
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

	resp, err := authsvc.HTTPClient().Do(req)
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
		return authsvc.LookupKey(ctx, cfg.JWKSURI, kid)
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
	if !authsvc.AudienceContains(claims["aud"], cfg.ClientID) {
		return nil, errors.New("audience mismatch")
	}
	if n, _ := claims["nonce"].(string); n != nonce {
		return nil, errors.New("nonce mismatch")
	}
	return claims, nil
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
	return common.RequestScheme(c) + "://" + common.RequestHost(c)
}

func (h *Handler) ssoRedirectURI(c *gin.Context, cfg ssoConfig) string {
	if v := strings.TrimSpace(cfg.RedirectURI); v != "" {
		return v
	}
	return common.SSODeriveRedirectURI(c)
}

func (h *Handler) redirectSSOError(c *gin.Context, frontendBase, code string) {
	frag := url.Values{}
	frag.Set("error", code)
	c.Redirect(http.StatusFound, frontendBase+"/sso/callback#"+frag.Encode())
}

type ssoSession struct {
	tokens     auth.TokenPair
	orgID      string
	role       string
	systemRole string
	orgType    string
}

func (h *Handler) buildUserSession(user models.User) (ssoSession, string, error) {
	systemRole, err := orgsvc.NewService(h.DB).ResolveSystemRole(user.ID.String(), user.SystemRole)
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
		if schema.HasOrgTypeColumn(h.DB) {
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
				return ssoSession{}, middleware.OrgStatusCode(org.Status), nil
			}
			orgType = org.OrgType
		}
		effectiveRole := orgsvc.NewService(h.DB).ResolveEffectiveOrgRole(member.OrgID.String(), member.Role)
		tokens, err := auth.IssueTokens(h.Cfg.JWTSecret, h.Cfg.JWTIssuer, user.ID.String(), member.OrgID.String(), effectiveRole, systemRole, h.Cfg.AccessTokenMinutes, h.Cfg.RefreshTokenHours)
		if err != nil {
			return ssoSession{}, "", err
		}
		return ssoSession{tokens: tokens, orgID: member.OrgID.String(), role: effectiveRole, systemRole: systemRole, orgType: orgType}, "", nil
	}

	if schema.HasOrgTypeColumn(h.DB) {
		personalOrg, personalMember, err := orgsvc.NewService(h.DB).EnsurePersonalMember(user.ID.String())
		if err != nil {
			return ssoSession{}, "", err
		}
		if personalOrg.ID != (uuid.UUID{}) {
			if strings.ToLower(strings.TrimSpace(personalOrg.Status)) != "active" {
				return ssoSession{}, middleware.OrgStatusCode(personalOrg.Status), nil
			}
			effectiveRole := orgsvc.NewService(h.DB).ResolveEffectiveOrgRole(personalMember.OrgID.String(), personalMember.Role)
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
			return ssoSession{}, middleware.OrgStatusCode(org.Status), nil
		}
		orgType = org.OrgType
	}
	effectiveRole := orgsvc.NewService(h.DB).ResolveEffectiveOrgRole(member.OrgID.String(), member.Role)
	tokens, err := auth.IssueTokens(h.Cfg.JWTSecret, h.Cfg.JWTIssuer, user.ID.String(), member.OrgID.String(), effectiveRole, systemRole, h.Cfg.AccessTokenMinutes, h.Cfg.RefreshTokenHours)
	if err != nil {
		return ssoSession{}, "", err
	}
	return ssoSession{tokens: tokens, orgID: member.OrgID.String(), role: effectiveRole, systemRole: systemRole, orgType: orgType}, "", nil
}
