package auth

import (
	"testing"
	"time"

	"telecom-platform/internal/config"
)

func TestIssueAndVerifyAccessToken(t *testing.T) {
	m, err := NewManager(config.AuthConfig{
		JWTSecret:       "secret",
		JWTIssuer:       "issuer",
		JWTAudience:     "aud",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("manager: %v", err)
	}

	now := time.Unix(1700000000, 0).UTC()
	pair, err := m.IssuePair(now, "user-1", "ws-1", "member")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("expected token strings")
	}

	claims, err := m.Verify(pair.AccessToken, TokenTypeAccess, now.Add(1*time.Minute))
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims.UserID != "user-1" || claims.WorkspaceID != "ws-1" || claims.Role != "member" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestVerifyRejectsWrongTokenType(t *testing.T) {
	m, _ := NewManager(config.AuthConfig{JWTSecret: "secret", AccessTokenTTL: time.Minute, RefreshTokenTTL: time.Hour})
	p, err := m.IssuePair(time.Now(), "u", "w", "r")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if _, err := m.Verify(p.RefreshToken, TokenTypeAccess, time.Now()); err == nil {
		t.Fatalf("expected token_type mismatch")
	}
}
