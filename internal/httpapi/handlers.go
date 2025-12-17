package httpapi

import (
	"net/http"
	"time"

	"telecom-platform/internal/auth"
	"telecom-platform/internal/rbac"
	"telecom-platform/internal/wallet"

	"github.com/gin-gonic/gin"
)

// Handlers groups HTTP handlers for dependency injection.
// Keep these thin: parse/validate input, call internal services, return JSON.

type Handlers struct {
	Auth   *auth.Manager
	Wallet *wallet.Service
}

// --- Auth ---

type loginRequest struct {
	UserID      string `json:"user_id"`
	WorkspaceID string `json:"workspace_id"`
	Role        string `json:"role"`
}

// Login issues a JWT token pair.
//
// NOTE: This is a skeleton-only endpoint. Real systems must validate credentials.
func (h Handlers) Login(c *gin.Context) {
	if h.Auth == nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "auth not configured"})
		return
	}
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if req.UserID == "" || req.WorkspaceID == "" || req.Role == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "user_id, workspace_id, role required"})
		return
	}
	pair, err := h.Auth.IssuePair(time.Now(), req.UserID, req.WorkspaceID, req.Role)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "token issuance failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": pair.AccessToken, "refresh_token": pair.RefreshToken})
}

// --- Wallet ---

type adminManualCreditRequest struct {
	WalletID string `json:"wallet_id"`

	AmountMinor    int64  `json:"amount_minor"`
	Currency       string `json:"currency"`
	Reason         string `json:"reason"`
	IdempotencyKey string `json:"idempotency_key"`
	Metadata       string `json:"metadata,omitempty"`
}

func (h Handlers) GetWalletBalance(c *gin.Context) {
	if h.Wallet == nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "wallet not configured"})
		return
	}
	workspaceID, err := auth.WorkspaceID(c.Request.Context())
	if err != nil || workspaceID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "workspace_id required"})
		return
	}
	walletID := c.Param("wallet_id")
	if walletID == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "wallet_id required"})
		return
	}
	bal, err := h.Wallet.GetBalance(c.Request.Context(), workspaceID, walletID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "balance lookup failed"})
		return
	}
	c.JSON(http.StatusOK, bal)
}

// AdminManualCredit performs an admin-only wallet credit.
// RBAC: owner or super_admin.
func (h Handlers) AdminManualCredit(c *gin.Context) {
	if h.Wallet == nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "wallet not configured"})
		return
	}
	workspaceID, err := auth.WorkspaceID(c.Request.Context())
	if err != nil || workspaceID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "workspace_id required"})
		return
	}
	adminUserID, _ := auth.UserID(c.Request.Context())
	adminRole, _ := auth.Role(c.Request.Context())

	var req adminManualCreditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if req.WalletID == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "wallet_id required"})
		return
	}

	_, _, bal, err := h.Wallet.AdminManualCredit(c.Request.Context(), workspaceID, req.WalletID, adminUserID, adminRole, wallet.AdminCreditRequest{
		AmountMinor:    req.AmountMinor,
		Currency:       req.Currency,
		Reason:         req.Reason,
		IdempotencyKey: req.IdempotencyKey,
		Metadata:       req.Metadata,
	})
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bal)
}

func RequireAdminAny(c *gin.Context) {
	_ = c
}

// Convenience middleware bundles.

func RequireWorkspaceAndAnyRole(roles ...string) []gin.HandlerFunc {
	return []gin.HandlerFunc{rbac.RequireWorkspace(), rbac.RequireAnyRole(roles...)}
}
