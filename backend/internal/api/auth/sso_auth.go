package auth

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"software-web-manager/backend/internal/api/common"
	"software-web-manager/backend/internal/middleware"
	authsvc "software-web-manager/backend/internal/services/auth"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type ssoAuthState struct {
	Nonce        string `json:"nonce"`
	CodeVerifier string `json:"code_verifier"`
	Redirect     string `json:"redirect"`
	Purpose      string `json:"purpose"`
	UserID       string `json:"user_id"`
}

const ssoStateTTL = 10 * time.Minute

func ssoStateKey(state string) string { return "swm:sso:state:" + state }

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
	cfg, err := authsvc.NewService(h.DB, h.Cfg).LoadSSOConfig()
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

// SSOLogoutURL builds the OIDC browser single-logout URL so the frontend can
// redirect the browser to the IdP and end the SSO session, not just the local
// session. The id_token_hint (the IdP id_token kept from login) is forwarded so
// the IdP can honor post_logout_redirect_uri without an extra confirmation page.
func (h *Handler) SSOLogoutURL(c *gin.Context) {
	cfg, err := authsvc.NewService(h.DB, h.Cfg).LoadSSOConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load sso config"})
		return
	}
	if !cfg.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sso_disabled"})
		return
	}
	endSession := authsvc.SSOEndSessionEndpoint(cfg)
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
	svc := authsvc.NewService(h.DB, h.Cfg)
	cfg, cfgErr := svc.LoadSSOConfig()
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

	tok, err := authsvc.ExchangeCode(c.Request.Context(), cfg, code, st.CodeVerifier, h.ssoRedirectURI(c, cfg))
	if err != nil {
		h.redirectSSOError(c, frontendBase, "sso_token_exchange_failed")
		return
	}

	claims, err := authsvc.VerifyIDToken(c.Request.Context(), cfg, tok.IDToken, st.Nonce)
	if err != nil {
		h.redirectSSOError(c, frontendBase, "sso_id_token_invalid")
		return
	}

	sub := strings.TrimSpace(authsvc.ClaimString(claims, "sub"))
	email := strings.ToLower(strings.TrimSpace(authsvc.ClaimString(claims, "email")))
	if email == "" && cfg.UserinfoEndpoint != "" && tok.AccessToken != "" {
		if ui, uiErr := authsvc.FetchUserinfo(c.Request.Context(), cfg, tok.AccessToken); uiErr == nil {
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

	user, err := svc.ResolveUser(sub, email)
	if err != nil {
		if errors.Is(err, authsvc.ErrSSOUserNotProvisioned) {
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

	sess, err := svc.BuildUserSession(user)
	if err != nil {
		var onae *authsvc.OrgNotActiveError
		if errors.As(err, &onae) {
			h.redirectSSOError(c, frontendBase, middleware.OrgStatusCode(onae.Status))
			return
		}
		if errors.Is(err, authsvc.ErrSSOUserNoOrg) {
			h.redirectSSOError(c, frontendBase, "user_no_org")
			return
		}
		h.redirectSSOError(c, frontendBase, "sso_error")
		return
	}

	frag := url.Values{}
	frag.Set("access_token", sess.Tokens.AccessToken)
	frag.Set("refresh_token", sess.Tokens.RefreshToken)
	frag.Set("expires_in", strconv.FormatInt(sess.Tokens.ExpiresIn, 10))
	frag.Set("system_role", sess.SystemRole)
	frag.Set("org_id", sess.OrgID)
	frag.Set("role", sess.Role)
	frag.Set("org_type", sess.OrgType)
	frag.Set("email", user.Email)
	frag.Set("redirect", st.Redirect)
	// Keep the IdP id_token so the frontend can pass it as id_token_hint when it
	// triggers OIDC browser single-logout (/oauth2/logout) on sign-out.
	frag.Set("sso_id_token", tok.IDToken)
	c.Redirect(http.StatusFound, frontendBase+"/sso/callback#"+frag.Encode())
}

func (h *Handler) SSOBindStart(c *gin.Context) {
	cfg, err := authsvc.NewService(h.DB, h.Cfg).LoadSSOConfig()
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
	if err := authsvc.NewService(h.DB, h.Cfg).BindSSO(userID, sub); err != nil {
		if errors.Is(err, authsvc.ErrSSOAlreadyBound) {
			h.redirectSSOError(c, frontendBase, "sso_already_bound")
			return
		}
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
	if err := authsvc.NewService(h.DB, h.Cfg).UnbindSSO(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unbind sso"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) ssoFrontendBase(c *gin.Context, cfg authsvc.SSOConfig) string {
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

func (h *Handler) ssoRedirectURI(c *gin.Context, cfg authsvc.SSOConfig) string {
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
