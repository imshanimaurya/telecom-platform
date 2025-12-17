package rbac

import (
	"net/http"

	"telecom-platform/internal/auth"

	"github.com/gin-gonic/gin"
)

// RequireWorkspace enforces the multi-tenant invariant: workspace_id must exist in context.
// This does not validate membership; that belongs to the authorization layer once persistence exists.
func RequireWorkspace() gin.HandlerFunc {
	return func(c *gin.Context) {
		wid, err := auth.WorkspaceID(c.Request.Context())
		if err != nil || wid == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "workspace_id required"})
			return
		}
		c.Next()
	}
}

// RequireAnyRole allows access if the caller has any of the provided roles.
// Rules:
// - super_admin bypasses all checks
// - network_operator is a hidden role, and will be denied unless explicitly allowed
// - workspace isolation is enforced via RequireWorkspace (use it in the chain)
func RequireAnyRole(allowed ...string) gin.HandlerFunc {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, r := range allowed {
		allowedSet[r] = struct{}{}
	}

	return func(c *gin.Context) {
		role, err := auth.Role(c.Request.Context())
		if err != nil || role == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "role required"})
			return
		}

		// super_admin bypasses all
		if IsSuperAdmin(role) {
			c.Next()
			return
		}

		// hidden roles are opt-in only
		if IsHiddenRole(role) {
			if _, ok := allowedSet[role]; !ok {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
				return
			}
		}

		if _, ok := allowedSet[role]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}
