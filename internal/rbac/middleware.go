package rbac

import (
	"net/http"

	"telecom-platform/internal/auth"

	"github.com/gin-gonic/gin"
)

/*
RequireWorkspace enforces the multi-tenant invariant.
workspace_id MUST exist in context for all protected routes.
*/
func RequireWorkspace() gin.HandlerFunc {
	return func(c *gin.Context) {
		wid, err := auth.WorkspaceIDFromGin(c)
		if err != nil || wid == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "workspace_id required",
			})
			return
		}
		c.Next()
	}
}

/*
RequireAnyRole allows access if caller has ANY allowed role.

Rules:
- super_admin bypasses all checks
- hidden roles are denied unless explicitly allowed
- workspace isolation enforced internally (fail-safe)
*/
func RequireAnyRole(allowed ...string) gin.HandlerFunc {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, r := range allowed {
		allowedSet[r] = struct{}{}
	}

	return func(c *gin.Context) {
		// Always enforce workspace
		wid, err := auth.WorkspaceIDFromGin(c)
		if err != nil || wid == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "workspace_id required",
			})
			return
		}

		role, err := auth.RoleFromGin(c)
		if err != nil || role == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "role required",
			})
			return
		}

		// Super admin bypass
		if IsSuperAdmin(role) {
			c.Next()
			return
		}

		// Role must be explicitly allowed
		if _, ok := allowedSet[role]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "forbidden",
			})
			return
		}

		// Hidden roles must be explicitly listed
		if IsHiddenRole(role) {
			if _, ok := allowedSet[role]; !ok {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "forbidden",
				})
				return
			}
		}

		c.Next()
	}
}
