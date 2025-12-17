package wallet

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"telecom-platform/internal/auth"
	"telecom-platform/internal/rbac"

	"github.com/gin-gonic/gin"
)

const (
	headerWalletID          = "X-Wallet-Id"
	headerEstimatedCostMinor = "X-Estimated-Cost-Minor"
	headerCurrency          = "X-Currency"
)

// BalanceService is the minimal wallet service interface needed by middleware.
type BalanceService interface {
	GetBalance(ctx context.Context, workspaceID, walletID string) (Balance, error)
}

// RequireSufficientBalance blocks the request if available balance is below the estimated cost.
//
// How it works (generic / non-business-logic):
// - Reads wallet_id from header: X-Wallet-Id
// - Reads estimated charge from header: X-Estimated-Cost-Minor (int64)
// - Reads currency from header: X-Currency
// - Uses auth context for workspace_id and role
//
// Admin override:
// - super_admin bypasses
// - hidden network_operator bypasses
// - (others can be added later by RBAC policy)
func RequireSufficientBalance(svc BalanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := auth.Role(c.Request.Context())
		if rbac.IsSuperAdmin(role) || role == rbac.RoleNetworkOperator {
			c.Next()
			return
		}

		workspaceID, err := auth.WorkspaceID(c.Request.Context())
		if err != nil || workspaceID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "workspace_id required"})
			return
		}

		walletID := strings.TrimSpace(c.GetHeader(headerWalletID))
		if walletID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "wallet id required"})
			return
		}

		estMinorStr := strings.TrimSpace(c.GetHeader(headerEstimatedCostMinor))
		if estMinorStr == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "estimated cost required"})
			return
		}
		estMinor, err := strconv.ParseInt(estMinorStr, 10, 64)
		if err != nil || estMinor <= 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "estimated cost invalid"})
			return
		}

		currency := strings.TrimSpace(c.GetHeader(headerCurrency))
		if currency == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "currency required"})
			return
		}

		bal, err := svc.GetBalance(c.Request.Context(), workspaceID, walletID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "balance lookup failed"})
			return
		}
		if bal.Currency != currency {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "currency mismatch"})
			return
		}
		if bal.BalanceMinor < estMinor {
			// 402 Payment Required is semantically appropriate.
			c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{"error": "insufficient balance"})
			return
		}

		c.Next()
	}
}
