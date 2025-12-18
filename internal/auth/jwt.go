package auth

import (
	"errors"
	"time"

	"telecom-platform/internal/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Manager struct {
	secret     []byte
	issuer     string
	audience   string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewManager(cfg config.AuthConfig) (*Manager, error) {
	if cfg.JWTSecret == "" {
		return nil, errors.New("JWT_SECRET is required")
	}

	return &Manager{
		secret:     []byte(cfg.JWTSecret),
		issuer:     cfg.JWTIssuer,
		audience:   cfg.JWTAudience,
		accessTTL:  cfg.AccessTokenTTL,
		refreshTTL: cfg.RefreshTokenTTL,
	}, nil
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

/* ===================== ISSUE TOKENS ===================== */

func (m *Manager) IssuePair(now time.Time, userID, workspaceID, role string) (TokenPair, error) {
	access, err := m.issue(
		now,
		TokenTypeAccess,
		userID,
		workspaceID,
		role,
		m.accessTTL,
	)
	if err != nil {
		return TokenPair{}, err
	}

	refresh, err := m.issue(
		now,
		TokenTypeRefresh,
		userID,
		workspaceID,
		"", // refresh tokens DO NOT carry role
		m.refreshTTL,
	)
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}

/* ===================== VERIFY TOKEN ===================== */

func (m *Manager) Verify(tokenString string, expected TokenType, now time.Time) (Claims, error) {
	var claims Claims

	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuedAt(),
		jwt.WithExpirationRequired(),
	)

	_, err := parser.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (any, error) {
		return m.secret, nil
	})
	if err != nil {
		return Claims{}, err
	}

	// Build ONE validator
	opts := []jwt.ValidatorOption{
		jwt.WithTimeFunc(func() time.Time { return now }),
		jwt.WithLeeway(30 * time.Second), // clock skew tolerance
		jwt.WithIssuedAt(),
		jwt.WithExpirationRequired(),
	}

	if m.issuer != "" {
		opts = append(opts, jwt.WithIssuer(m.issuer))
	}
	if m.audience != "" {
		opts = append(opts, jwt.WithAudience(m.audience))
	}

	validator := jwt.NewValidator(opts...)
	if err := validator.Validate(claims.RegisteredClaims); err != nil {
		return Claims{}, err
	}

	// Custom claims validation
	if claims.TokenType != expected {
		return Claims{}, errors.New("token_type mismatch")
	}
	if claims.UserID == "" {
		return Claims{}, errors.New("user_id missing")
	}
	if claims.WorkspaceID == "" {
		return Claims{}, errors.New("workspace_id missing")
	}

	// Role is required ONLY for access tokens
	if expected == TokenTypeAccess && claims.Role == "" {
		return Claims{}, errors.New("role missing in access token")
	}

	return claims, nil
}

/* ===================== INTERNAL ISSUE ===================== */

func (m *Manager) issue(
	now time.Time,
	tokenType TokenType,
	userID,
	workspaceID,
	role string,
	ttl time.Duration,
) (string, error) {

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
