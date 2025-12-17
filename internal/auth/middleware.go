package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const authorizationHeader = "Authorization"
const bearerPrefix = "Bearer "

// RequireAccessToken verifies an access token and injects identity into request context.
// It does not perform RBAC checks; those belong to internal/rbac.
func RequireAccessToken(m *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.GetHeader(authorizationHeader))
		if raw == "" || !strings.HasPrefix(raw, bearerPrefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		tok := strings.TrimPrefix(raw, bearerPrefix)

		claims, err := m.Verify(tok, TokenTypeAccess, time.Now())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		ctx := WithIdentity(c.Request.Context(), claims.UserID, claims.WorkspaceID, claims.Role)
		c.Request = c.Request.WithContext(ctx)

		// Also store on gin context for handler convenience.
		c.Set("user_id", claims.UserID)
		c.Set("workspace_id", claims.WorkspaceID)
		c.Set("role", claims.Role)

		c.Next()
	}
}
