package rbac

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"telecom-platform/internal/auth"

	"github.com/gin-gonic/gin"
)

func TestRequireAnyRole_SuperAdminBypasses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/x", func(c *gin.Context) {
		ctx := auth.WithIdentity(c.Request.Context(), "u", "w", RoleSuperAdmin)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}, RequireWorkspace(), RequireAnyRole(RoleOwner), func(c *gin.Context) {
		c.Status(200)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRequireAnyRole_HiddenRoleDeniedUnlessAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/x", func(c *gin.Context) {
		ctx := auth.WithIdentity(c.Request.Context(), "u", "w", RoleNetworkOperator)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}, RequireWorkspace(), RequireAnyRole(RoleOwner), func(c *gin.Context) {
		c.Status(200)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestRequireAnyRole_WorkspaceRequired(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/x", func(c *gin.Context) {
		ctx := auth.WithIdentity(c.Request.Context(), "u", "", RoleOwner)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}, RequireWorkspace(), RequireAnyRole(RoleOwner), func(c *gin.Context) {
		c.Status(200)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
