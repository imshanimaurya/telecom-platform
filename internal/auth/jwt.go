package auth

import (
	"errors"
	"time"

	"telecom-platform/internal/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Manager struct {
	secret        []byte
	issuer        string
	audience      string
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

func NewManager(cfg config.AuthConfig) (*Manager, error) {
	if cfg.JWTSecret == "" {
		return nil, errors.New("JWT_SECRET is required")
	}
	m := &Manager{
		secret:     []byte(cfg.JWTSecret),
		issuer:     cfg.JWTIssuer,
		audience:   cfg.JWTAudience,
		accessTTL:  cfg.AccessTokenTTL,
		refreshTTL: cfg.RefreshTokenTTL,
	}
	return m, nil
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

func (m *Manager) IssuePair(now time.Time, userID, workspaceID, role string) (TokenPair, error) {
	access, err := m.issue(now, TokenTypeAccess, userID, workspaceID, role, m.accessTTL)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, err := m.issue(now, TokenTypeRefresh, userID, workspaceID, role, m.refreshTTL)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

func (m *Manager) Verify(tokenString string, expected TokenType, now time.Time) (Claims, error) {
	claims := Claims{}

	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuedAt(),
		jwt.WithExpirationRequired(),
	)

	tok, err := parser.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (any, error) {
		return m.secret, nil
	})
	if err != nil {
		return Claims{}, err
	}
	if !tok.Valid {
		return Claims{}, errors.New("invalid token")
	}

	// Standard claim validation.
	// We validate RegisteredClaims using jwt/v5's validator.
	validator := jwt.NewValidator(
		jwt.WithTimeFunc(func() time.Time { return now }),
		jwt.WithIssuedAt(),
		jwt.WithExpirationRequired(),
	)
	if m.issuer != "" {
		validator = jwt.NewValidator(
			jwt.WithTimeFunc(func() time.Time { return now }),
			jwt.WithIssuedAt(),
			jwt.WithExpirationRequired(),
			jwt.WithIssuer(m.issuer),
		)
	}
	if m.audience != "" {
		// If issuer is also set, include it in the same validator.
		opts := []any{
			jwt.WithTimeFunc(func() time.Time { return now }),
			jwt.WithIssuedAt(),
			jwt.WithExpirationRequired(),
			jwt.WithAudience(m.audience),
		}
		if m.issuer != "" {
			opts = append(opts, jwt.WithIssuer(m.issuer))
		}
		// Rebuild validator using the options slice.
		validator = jwt.NewValidator(
			jwt.WithTimeFunc(func() time.Time { return now }),
			jwt.WithIssuedAt(),
			jwt.WithExpirationRequired(),
			jwt.WithAudience(m.audience),
		)
		if m.issuer != "" {
			validator = jwt.NewValidator(
				jwt.WithTimeFunc(func() time.Time { return now }),
				jwt.WithIssuedAt(),
				jwt.WithExpirationRequired(),
				jwt.WithAudience(m.audience),
				jwt.WithIssuer(m.issuer),
			)
		}
		_ = opts
	}
	if err := validator.Validate(claims.RegisteredClaims); err != nil {
		return Claims{}, err
	}

	if claims.TokenType != expected {
		return Claims{}, errors.New("token_type mismatch")
	}
	if claims.UserID == "" {
		return Claims{}, errors.New("user_id missing")
	}
	if claims.WorkspaceID == "" {
		return Claims{}, errors.New("workspace_id missing")
	}
	if claims.Role == "" {
		return Claims{}, errors.New("role missing")
	}

	return claims, nil
}

func (m *Manager) issue(now time.Time, tokenType TokenType, userID, workspaceID, role string, ttl time.Duration) (string, error) {
	jti := uuid.NewString()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Audience:  audienceOrNil(m.audience),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        jti,
		},
		UserID:      userID,
		WorkspaceID: workspaceID,
		Role:        role,
		TokenType:   tokenType,
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(m.secret)
}

func audienceOrNil(aud string) jwt.ClaimStrings {
	if aud == "" {
		return nil
	}
	return jwt.ClaimStrings{aud}
}
