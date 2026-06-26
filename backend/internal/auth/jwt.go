package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Token "use" values distinguish short-lived access tokens from long-lived
// refresh tokens so a refresh token cannot be replayed as an access token.
const (
	TokenUseAccess  = "access"
	TokenUseRefresh = "refresh"
)

type Claims struct {
	UserID     string `json:"uid"`
	OrgID      string `json:"oid"`
	Role       string `json:"role"`
	SystemRole string `json:"system_role"`
	// TokenUse is "access" or "refresh"; middleware accepts only access tokens,
	// the refresh endpoint accepts only refresh tokens.
	TokenUse string `json:"token_use,omitempty"`
	// TokenVersion is the user's session epoch; a mismatch with the user's current
	// token_version (bumped on password change, etc.) revokes the token.
	TokenVersion int `json:"tv,omitempty"`
	jwt.RegisteredClaims
}

type OnlineStreamClaims struct {
	UserID       string `json:"uid"`
	OrgID        string `json:"oid"`
	SystemRole   string `json:"system_role"`
	AppID        string `json:"app_id"`
	Purpose      string `json:"purpose"`
	TokenVersion int    `json:"tv,omitempty"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

func IssueTokens(secret, issuer, userID, orgID, role, systemRole string, tokenVersion, accessMinutes, refreshHours int) (TokenPair, error) {
	now := time.Now()
	accessExp := now.Add(time.Duration(accessMinutes) * time.Minute)
	refreshExp := now.Add(time.Duration(refreshHours) * time.Hour)

	accessClaims := Claims{
		UserID:       userID,
		OrgID:        orgID,
		Role:         role,
		SystemRole:   systemRole,
		TokenUse:     TokenUseAccess,
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(accessExp),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	refreshClaims := Claims{
		UserID:       userID,
		OrgID:        orgID,
		Role:         role,
		SystemRole:   systemRole,
		TokenUse:     TokenUseRefresh,
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(refreshExp),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(secret))
	if err != nil {
		return TokenPair{}, err
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(secret))
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(time.Until(accessExp).Seconds()),
	}, nil
}

func ParseToken(secret string, tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

func IssueOnlineStreamToken(secret, issuer, userID, orgID, systemRole, appID string, tokenVersion int, ttl time.Duration) (string, int64, error) {
	now := time.Now()
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	expiresAt := now.Add(ttl)
	claims := OnlineStreamClaims{
		UserID:       userID,
		OrgID:        orgID,
		SystemRole:   systemRole,
		AppID:        appID,
		Purpose:      "online_stream",
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		return "", 0, err
	}
	return token, int64(time.Until(expiresAt).Seconds()), nil
}

func ParseOnlineStreamToken(secret string, tokenStr string) (*OnlineStreamClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &OnlineStreamClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*OnlineStreamClaims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}
