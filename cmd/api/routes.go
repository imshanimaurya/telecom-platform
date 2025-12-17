package main

import (
	"errors"
	"telecom-platform/internal/auth"
	"telecom-platform/internal/httpapi"
	"telecom-platform/internal/rbac"
	"telecom-platform/internal/routing"
	"telecom-platform/internal/telephony"
	"telecom-platform/internal/wallet"

	"github.com/gin-gonic/gin"
)

// registerRoutes wires HTTP routes to handlers.
// Keep this file free of business logic. Handlers should delegate to internal modules.
func registerRoutes(r *gin.Engine, authMW gin.HandlerFunc) {
	// public
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Provider webhooks (public).
	// NOTE: This endpoint should be protected by Twilio signature validation in production.
	{
		re := routing.NewRoutingEngine(nil, nil, nil)
		router := routing.NewEngineAdapter(re, routing.AdapterOptions{})
		twilioProvider := telephony.NewTwilioProvider(router)
		h := telephony.TwilioWebhookHandler{
			Provider: twilioProvider,
			WorkspaceIDResolver: func(c *gin.Context, toNumber string) (string, error) {
				// TODO: Resolve workspace_id by looking up the dialed number in storage.
				// Kept as a function injection to avoid persistence assumptions here.
				return "", errors.New("workspace resolver not implemented")
			},
		}
		r.POST("/webhooks/twilio/voice", h.HandleInboundCall)
	}

	// protected API group
	v1 := r.Group("/v1")
	v1.Use(authMW)
	{
		h := httpapi.Handlers{
			// Auth manager is already used by authMW; login uses the same manager but is wired in main.
			// In this skeleton routes file we keep handlers lightweight and safe.
			Auth:   nil,
			Wallet: nil,
		}
		_ = h

		// Placeholder route to demonstrate identity extraction via context.
		v1.GET("/me", func(c *gin.Context) {
			uid, _ := auth.UserID(c.Request.Context())
			wid, _ := auth.WorkspaceID(c.Request.Context())
			role, _ := auth.Role(c.Request.Context())
			c.JSON(200, gin.H{"user_id": uid, "workspace_id": wid, "role": role})
		})

		// AUTH routes (token issuance).
		// NOTE: This is a placeholder login route; real credential validation is not implemented.
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/login", func(c *gin.Context) {
				c.AbortWithStatusJSON(501, gin.H{"error": "login handler not wired (requires auth manager DI)"})
			})
		}

		// WALLET routes
		wallets := v1.Group("/wallets")
		wallets.Use(rbac.RequireWorkspace())
		{
			wallets.GET("/:wallet_id/balance", func(c *gin.Context) {
				c.AbortWithStatusJSON(501, gin.H{"error": "wallet handler not wired (requires wallet service DI)"})
			})
		}

		// CALLS routes
		calls := v1.Group("/calls")
		calls.Use(rbac.RequireWorkspace())
		calls.Use(rbac.RequireAnyRole(rbac.RoleOwner, rbac.RoleAgent, rbac.RoleSuperAdmin))
		{
			calls.POST("/start", func(c *gin.Context) {
				// Placeholder only; actual call orchestration belongs to internal/calls.
				c.JSON(200, gin.H{"status": "queued"})
			})
		}

		// CAMPAIGNS routes
		campaigns := v1.Group("/campaigns")
		campaigns.Use(rbac.RequireWorkspace())
		campaigns.Use(rbac.RequireAnyRole(rbac.RoleOwner, rbac.RoleAnalyst, rbac.RoleSuperAdmin))
		{
			campaigns.GET("/", func(c *gin.Context) {
				c.AbortWithStatusJSON(501, gin.H{"error": "campaigns not implemented"})
			})
		}


		// ADMIN routes
		// Only owner/super_admin can access admin endpoints by default.
		// Hidden network_operator is intentionally NOT included unless explicitly desired.
		admin := v1.Group("/admin")
		admin.Use(rbac.RequireWorkspace())
		admin.Use(rbac.RequireAnyRole(rbac.RoleOwner, rbac.RoleSuperAdmin))
		{
			admin.GET("/ping", func(c *gin.Context) {
				c.JSON(200, gin.H{"status": "ok"})
			})

			// Admin wallet credit (placeholder wiring until DI is added).
			admin.POST("/wallets/manual-credit", func(c *gin.Context) {
				// Avoid constructing wallet service with nil dependencies.
				_ = wallet.ErrInvalidArgument
				c.AbortWithStatusJSON(501, gin.H{"error": "wallet admin handler not wired (requires wallet service DI)"})
			})
		}
	}
}
