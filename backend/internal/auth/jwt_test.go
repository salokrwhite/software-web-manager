package auth

import "testing"

func TestIssueTokensStampsUseAndVersion(t *testing.T) {
	const secret = "unit-test-secret"
	pair, err := IssueTokens(secret, "swm", "user-1", "org-1", "owner", "none", 7, 30, 720)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	access, err := ParseToken(secret, pair.AccessToken)
	if err != nil {
		t.Fatalf("parse access: %v", err)
	}
	if access.TokenUse != TokenUseAccess {
		t.Fatalf("access token_use = %q, want %q", access.TokenUse, TokenUseAccess)
	}
	if access.TokenVersion != 7 {
		t.Fatalf("access tv = %d, want 7", access.TokenVersion)
	}
	if access.UserID != "user-1" || access.OrgID != "org-1" {
		t.Fatalf("access claims not preserved: %+v", access)
	}

	refresh, err := ParseToken(secret, pair.RefreshToken)
	if err != nil {
		t.Fatalf("parse refresh: %v", err)
	}
	if refresh.TokenUse != TokenUseRefresh {
		t.Fatalf("refresh token_use = %q, want %q", refresh.TokenUse, TokenUseRefresh)
	}
	if refresh.TokenVersion != 7 {
		t.Fatalf("refresh tv = %d, want 7", refresh.TokenVersion)
	}

	// Wrong secret must not verify.
	if _, err := ParseToken("other-secret", pair.AccessToken); err == nil {
		t.Fatalf("expected parse failure with wrong secret")
	}
}
