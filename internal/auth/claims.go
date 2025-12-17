package auth

import "github.com/golang-jwt/jwt/v5"

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

// Claims are the only supported JWT claims shape for this service.
// Multi-tenant invariant: WorkspaceID must be present for all non-admin activity.
// Hidden/admin override capabilities should be represented via separate server-side authorization checks.
type Claims struct {
	jwt.RegisteredClaims

	UserID      string    `json:"user_id"`
	WorkspaceID string    `json:"workspace_id"`
	Role        string    `json:"role"`
	TokenType   TokenType `json:"token_type"`
}
