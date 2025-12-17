package telephony

import (
	"net/http"
	"time"

	"telecom-platform/internal/routing"
	"telecom-platform/pkg/logger"

	"github.com/gin-gonic/gin"
)

// TwilioWebhookHandler converts the Twilio webhook to internal types,
// delegates routing to the provider adapter, and writes TwiML.
//
// No business logic here.
//
// Tenant scoping:
// - workspace_id is resolved by the caller (e.g., resolved from the dialed number via DB lookup)
//   and passed explicitly.

type TwilioWebhookHandler struct {
	Provider TelephonyProvider

	// WorkspaceIDResolver resolves which workspace owns the dialed number.
	// For now, it's an injected function to avoid any persistence assumptions in this skeleton.
	WorkspaceIDResolver func(c *gin.Context, toNumber string) (string, error)

	Now func() time.Time
}

func (h TwilioWebhookHandler) HandleInboundCall(c *gin.Context) {
	log := logger.FromGin(c)

	if h.Now == nil {
		h.Now = time.Now
	}
	if h.Provider == nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "telephony provider not configured"})
		return
	}
	if h.WorkspaceIDResolver == nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "workspace resolver not configured"})
		return
	}

	form, err := ParseTwilioInboundCall(c.Request)
	if err != nil {
		log.Warn("twilio webhook parse failed", "err", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid form"})
		return
	}

	workspaceID, err := h.WorkspaceIDResolver(c, form.To)
	if err != nil {
		log.Warn("workspace resolution failed", "to", form.To, "err", err)
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "unknown destination"})
		return
	}

	in := form.ToInboundCallRequest(workspaceID, h.Now())
	ctx := routing.WithClientIP(c.Request.Context(), c.ClientIP())

	res, err := h.Provider.HandleInboundCall(ctx, in)
	if err != nil {
		log.Error("inbound call routing failed", "err", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "routing failed"})
		return
	}

	twiml, err := RenderTwiML(res)
	if err != nil {
		log.Error("twiml render failed", "err", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "twiml failed"})
		return
	}

	c.Header("Content-Type", "application/xml")
	c.String(http.StatusOK, twiml)
}
