package wallet

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"telecom-platform/internal/auth"
	"telecom-platform/internal/rbac"

	"github.com/gin-gonic/gin"
)

type fakeBalanceService struct {
	bal Balance
	err error
}

func (f fakeBalanceService) GetBalance(ctx context.Context, workspaceID, walletID string) (Balance, error) {
	return f.bal, f.err
}

func TestRequireSufficientBalance_BlocksWhenInsufficient(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	svc := fakeBalanceService{bal: Balance{WorkspaceID: "ws", WalletID: "w1", Currency: "USD", BalanceMinor: 50}}

	r.GET("/x", func(c *gin.Context) {
		ctx := auth.WithIdentity(c.Request.Context(), "u", "ws", rbac.RoleOwner)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}, RequireSufficientBalance(svc), func(c *gin.Context) {
		c.Status(200)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-Wallet-Id", "w1")
	req.Header.Set("X-Estimated-Cost-Minor", "100")
	req.Header.Set("X-Currency", "USD")

	r.ServeHTTP(w, req)
	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402, got %d", w.Code)
	}
}

func TestRequireSufficientBalance_AllowsAdminOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	svc := fakeBalanceService{bal: Balance{WorkspaceID: "ws", WalletID: "w1", Currency: "USD", BalanceMinor: 0}}

	r.GET("/x", func(c *gin.Context) {
		ctx := auth.WithIdentity(c.Request.Context(), "u", "ws", rbac.RoleSuperAdmin)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}, RequireSufficientBalance(svc), func(c *gin.Context) {
		c.Status(200)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-Wallet-Id", "w1")
	req.Header.Set("X-Estimated-Cost-Minor", "100")
	req.Header.Set("X-Currency", "USD")

	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
