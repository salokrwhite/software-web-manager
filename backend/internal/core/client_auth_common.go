package core

import (
	"encoding/json"
	"errors"
	"strings"

	"gorm.io/datatypes"
)

var ErrInsufficientScope = errors.New("insufficient scope")
var ErrAppPending = errors.New("app_pending_review")
var ErrAppRejected = errors.New("app_rejected")

func scopeAllows(scopes []string, scope string) bool {
	if len(scopes) == 0 {
		return true
	}
	for _, s := range scopes {
		if s == "*" || strings.EqualFold(s, scope) {
			return true
		}
	}
	return false
}

func parseScopesJSON(raw datatypes.JSON) []string {
	if len(raw) == 0 {
		return nil
	}
	var scopes []string
	if err := json.Unmarshal(raw, &scopes); err != nil {
		return nil
	}
	return normalizeScopes(scopes)
}

func normalizeScopes(scopes []string) []string {
	if len(scopes) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(scopes))
	out := make([]string, 0, len(scopes))
	for _, item := range scopes {
		val := strings.ToLower(strings.TrimSpace(item))
		if val == "" {
			continue
		}
		if _, ok := seen[val]; ok {
			continue
		}
		seen[val] = struct{}{}
		out = append(out, val)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func defaultAppSecretScopes() []string {
	return []string{"update:check", "event:write"}
}

func SanitizeAppSecretScopes(scopes []string) []string {
	normalized := normalizeScopes(scopes)
	if len(normalized) == 0 {
		return defaultAppSecretScopes()
	}
	out := make([]string, 0, len(normalized))
	for _, item := range normalized {
		switch item {
		case "update:check", "event:write", "*":
			out = append(out, item)
		}
	}
	if len(out) == 0 {
		return defaultAppSecretScopes()
	}
	return out
}

func AppSecretScopesJSON(scopes []string) datatypes.JSON {
	if len(scopes) == 0 {
		return datatypes.JSON([]byte("[]"))
	}
	b, err := json.Marshal(scopes)
	if err != nil {
		return datatypes.JSON([]byte("[]"))
	}
	return datatypes.JSON(b)
}

func AppSecretScopesFromJSON(raw datatypes.JSON) []string {
	return SanitizeAppSecretScopes(parseScopesJSON(raw))
}
