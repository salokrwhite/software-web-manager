package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	authcore "software-web-manager/backend/internal/auth"
	"software-web-manager/backend/internal/db/schema"
	"software-web-manager/backend/internal/models"
	orgsvc "software-web-manager/backend/internal/services/org"
	systemsvc "software-web-manager/backend/internal/services/system"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SSOConfig holds the resolved OIDC/SSO provider configuration.
type SSOConfig struct {
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

// SSOTokenResponse is the token endpoint response.
type SSOTokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	Scope       string `json:"scope"`
}

// SSOSession is the resolved identity + issued tokens for an SSO login.
type SSOSession struct {
	Tokens     authcore.TokenPair
	OrgID      string
	Role       string
	SystemRole string
	OrgType    string
}

// Sentinel errors for SSO flows.
var (
	// ErrSSOUserNotProvisioned indicates no local account matches the SSO identity.
	ErrSSOUserNotProvisioned = errors.New("sso user not provisioned")
	// ErrSSOUserNoOrg indicates the user has no org membership.
	ErrSSOUserNoOrg = errors.New("sso user no org")
	// ErrSSOAlreadyBound indicates the SSO subject is already bound to another user.
	ErrSSOAlreadyBound = errors.New("sso already bound")
)

// LoadSSOConfig resolves the SSO provider configuration from system settings,
// falling back to defaults when the settings table is absent.
func (s *Service) LoadSSOConfig() (SSOConfig, error) {
	if !schema.HasSystemSettingsTable(s.DB) {
		return SSOConfig{Enabled: systemsvc.DefaultSSOEnabled, DisplayName: systemsvc.DefaultSSODisplayName, Scopes: systemsvc.DefaultSSOScopes}, nil
	}
	items, err := systemsvc.NewService(s.DB).ListSettings()
	if err != nil {
		return SSOConfig{}, err
	}
	return SSOConfig{
		Enabled:           systemsvc.GetBool(items, systemsvc.SettingSSOEnabledKey, systemsvc.DefaultSSOEnabled),
		DisplayName:       systemsvc.GetString(items, systemsvc.SettingSSODisplayNameKey, systemsvc.DefaultSSODisplayName),
		Issuer:            systemsvc.GetString(items, systemsvc.SettingSSOIssuerKey, ""),
		AuthorizeEndpoint: systemsvc.GetString(items, systemsvc.SettingSSOAuthorizeEndpointKey, ""),
		TokenEndpoint:     systemsvc.GetString(items, systemsvc.SettingSSOTokenEndpointKey, ""),
		UserinfoEndpoint:  systemsvc.GetString(items, systemsvc.SettingSSOUserinfoEndpointKey, ""),
		JWKSURI:           systemsvc.GetString(items, systemsvc.SettingSSOJWKSURIKey, ""),
		ClientID:          systemsvc.GetString(items, systemsvc.SettingSSOClientIDKey, ""),
		ClientSecret:      systemsvc.GetString(items, systemsvc.SettingSSOClientSecretKey, ""),
		Scopes:            systemsvc.GetString(items, systemsvc.SettingSSOScopesKey, systemsvc.DefaultSSOScopes),
		RedirectURI:       systemsvc.GetString(items, systemsvc.SettingSSORedirectURIKey, ""),
	}, nil
}

// SSOEndSessionEndpoint resolves the IdP browser-logout (/oauth2/logout) URL.
// The IdP exposes it at {issuer}/oauth2/logout; if the issuer is not set we
// derive it from the authorize endpoint (…/oauth2/authorize → …/oauth2/logout).
func SSOEndSessionEndpoint(cfg SSOConfig) string {
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

// ExchangeCode exchanges an authorization code for tokens at the token endpoint.
func ExchangeCode(ctx context.Context, cfg SSOConfig, code, verifier, redirectURI string) (SSOTokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("code_verifier", verifier)
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return SSOTokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := HTTPClient().Do(req)
	if err != nil {
		return SSOTokenResponse{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return SSOTokenResponse{}, fmt.Errorf("token endpoint status %d", resp.StatusCode)
	}
	var tok SSOTokenResponse
	if err := json.Unmarshal(body, &tok); err != nil {
		return SSOTokenResponse{}, err
	}
	if strings.TrimSpace(tok.IDToken) == "" {
		return SSOTokenResponse{}, errors.New("missing id_token")
	}
	return tok, nil
}

// FetchUserinfo fetches the OIDC userinfo claims with the access token.
func FetchUserinfo(ctx context.Context, cfg SSOConfig, accessToken string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.UserinfoEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := HTTPClient().Do(req)
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

// VerifyIDToken validates an OIDC id_token's signature (via JWKS), issuer,
// audience, and nonce, returning its claims.
func VerifyIDToken(ctx context.Context, cfg SSOConfig, idToken, nonce string) (jwt.MapClaims, error) {
	if strings.TrimSpace(idToken) == "" {
		return nil, errors.New("empty id_token")
	}
	keyFunc := func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		kid, _ := t.Header["kid"].(string)
		return LookupKey(ctx, cfg.JWKSURI, kid)
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
	if !AudienceContains(claims["aud"], cfg.ClientID) {
		return nil, errors.New("audience mismatch")
	}
	if n, _ := claims["nonce"].(string); n != nonce {
		return nil, errors.New("nonce mismatch")
	}
	return claims, nil
}

// ResolveUser finds the local user for an SSO identity, linking the subject to a
// matching email account when needed. Returns ErrSSOUserNotProvisioned when no
// account matches.
func (s *Service) ResolveUser(sub, email string) (models.User, error) {
	var user models.User
	if sub != "" {
		if err := s.DB.Where("sso_sub = ?", sub).First(&user).Error; err == nil {
			return user, nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return models.User{}, err
		}
	}
	if email == "" {
		return models.User{}, ErrSSOUserNotProvisioned
	}
	if err := s.DB.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.User{}, ErrSSOUserNotProvisioned
		}
		return models.User{}, err
	}
	if sub != "" && (user.SSOSub == nil || strings.TrimSpace(*user.SSOSub) != sub) {
		subCopy := sub
		if err := s.DB.Model(&models.User{}).Where("id = ?", user.ID).Update("sso_sub", &subCopy).Error; err != nil {
			return models.User{}, err
		}
		user.SSOSub = &subCopy
	}
	return user, nil
}

// BuildUserSession resolves the user's org context and issues a token pair. It
// returns OrgNotActiveError when the bound org is inactive and ErrSSOUserNoOrg
// when the user has no org membership.
func (s *Service) BuildUserSession(user models.User) (*SSOSession, error) {
	systemRole, err := orgsvc.NewService(s.DB).ResolveSystemRole(user.ID.String(), user.SystemRole)
	if err != nil {
		return nil, err
	}

	if systemRole == "system_admin" {
		tokens, err := s.issueTokens(user.ID.String(), "", "", systemRole)
		if err != nil {
			return nil, err
		}
		return &SSOSession{Tokens: tokens, SystemRole: systemRole}, nil
	}

	// Enterprise admins (org_admin) are locked to their enterprise org and cannot
	// switch orgs, so bind the SSO session straight to that org (mirroring
	// AdminLogin) instead of falling through to the personal-org branch below.
	if systemRole == "org_admin" {
		var member models.OrgMember
		memberLoaded := false
		if schema.HasOrgTypeColumn(s.DB) {
			if err := s.DB.Raw(`
				SELECT m.scope_id, m.user_id, m.role, m.created_at
				FROM memberships m
				JOIN orgs o ON o.id = m.scope_id
				WHERE m.scope_type = 'org' AND m.user_id = ? AND COALESCE(o.org_type, '') <> 'personal'
				ORDER BY o.created_at DESC
				LIMIT 1
			`, user.ID).Scan(&member).Error; err != nil {
				return nil, err
			}
			if member.OrgID != (uuid.UUID{}) {
				memberLoaded = true
			}
		}
		if !memberLoaded {
			if err := s.DB.Where("scope_type = ? AND user_id = ?", models.ScopeOrg, user.ID).First(&member).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, ErrSSOUserNoOrg
				}
				return nil, err
			}
		}
		orgType := ""
		var org models.Org
		if err := s.DB.Where("id = ?", member.OrgID).First(&org).Error; err == nil {
			if strings.ToLower(strings.TrimSpace(org.Status)) != "active" {
				return nil, &OrgNotActiveError{Status: org.Status}
			}
			orgType = org.OrgType
		}
		effectiveRole := orgsvc.NewService(s.DB).ResolveEffectiveOrgRole(member.OrgID.String(), member.Role)
		tokens, err := s.issueTokens(user.ID.String(), member.OrgID.String(), effectiveRole, systemRole)
		if err != nil {
			return nil, err
		}
		return &SSOSession{Tokens: tokens, OrgID: member.OrgID.String(), Role: effectiveRole, SystemRole: systemRole, OrgType: orgType}, nil
	}

	if schema.HasOrgTypeColumn(s.DB) {
		personalOrg, personalMember, err := orgsvc.NewService(s.DB).EnsurePersonalMember(user.ID.String())
		if err != nil {
			return nil, err
		}
		if personalOrg.ID != (uuid.UUID{}) {
			if strings.ToLower(strings.TrimSpace(personalOrg.Status)) != "active" {
				return nil, &OrgNotActiveError{Status: personalOrg.Status}
			}
			effectiveRole := orgsvc.NewService(s.DB).ResolveEffectiveOrgRole(personalMember.OrgID.String(), personalMember.Role)
			tokens, err := s.issueTokens(user.ID.String(), personalMember.OrgID.String(), effectiveRole, systemRole)
			if err != nil {
				return nil, err
			}
			return &SSOSession{Tokens: tokens, OrgID: personalMember.OrgID.String(), Role: effectiveRole, SystemRole: systemRole, OrgType: personalOrg.OrgType}, nil
		}
	}

	var member models.OrgMember
	if err := s.DB.Where("scope_type = ? AND user_id = ?", models.ScopeOrg, user.ID).First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSSOUserNoOrg
		}
		return nil, err
	}
	orgType := ""
	var org models.Org
	if err := s.DB.Where("id = ?", member.OrgID).First(&org).Error; err == nil {
		if strings.ToLower(strings.TrimSpace(org.Status)) != "active" {
			return nil, &OrgNotActiveError{Status: org.Status}
		}
		orgType = org.OrgType
	}
	effectiveRole := orgsvc.NewService(s.DB).ResolveEffectiveOrgRole(member.OrgID.String(), member.Role)
	tokens, err := s.issueTokens(user.ID.String(), member.OrgID.String(), effectiveRole, systemRole)
	if err != nil {
		return nil, err
	}
	return &SSOSession{Tokens: tokens, OrgID: member.OrgID.String(), Role: effectiveRole, SystemRole: systemRole, OrgType: orgType}, nil
}

// BindSSO links an SSO subject to a user, returning ErrSSOAlreadyBound when the
// subject is already bound to a different user.
func (s *Service) BindSSO(userID, sub string) error {
	var existing models.User
	err := s.DB.Where("sso_sub = ?", sub).First(&existing).Error
	if err == nil && existing.ID.String() != userID {
		return ErrSSOAlreadyBound
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	subCopy := sub
	return s.DB.Model(&models.User{}).Where("id = ?", userID).Update("sso_sub", &subCopy).Error
}

// UnbindSSO clears the SSO subject binding for a user.
func (s *Service) UnbindSSO(userID string) error {
	return s.DB.Model(&models.User{}).Where("id = ?", userID).Update("sso_sub", nil).Error
}
