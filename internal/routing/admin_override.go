package routing

import (
	"context"
	"errors"
	"time"

	"telecom-platform/internal/telephony"
)

// AdminOverrideEngine applies silent, expiry-based routing overrides.
//
// Requirements:
// - Silent routing: callers/users must not be able to infer that an override was used.
//   That means: do NOT surface special reasons/messages to user-facing APIs.
// - Expiry based: overrides must be time-bounded.
// - Internal audit logging: every applied override should be recorded.
// - No user visibility: audit is internal-only.
//
// This component returns a Decision only and does not call providers.
// It is intended to be placed *ahead of* normal routing evaluation.

type AdminOverrideEngine struct {
	Store OverrideStore
	Audit AuditLogger
	Now   func() time.Time
}

// OverrideStore resolves currently-active overrides.
// Implementations may use Postgres/Redis.
//
// SECURITY NOTE:
// Keep this data plane accessible only to privileged internal services.

type OverrideStore interface {
	// GetActiveOverride returns an active override if one exists for this request.
	// If none exists, it returns (Override{}, false, nil).
	GetActiveOverride(ctx context.Context, workspaceID, campaignID string, req telephony.InboundCallRequest, now time.Time) (Override, bool, error)
}

// AuditLogger records internal-only audit events.
// Implementation should write to an internal audit table/stream.

type AuditLogger interface {
	LogOverrideApplied(ctx context.Context, e OverrideAuditEvent) error
}

type Override struct {
	WorkspaceID string
	CampaignID  string
	// OverrideID is optional but recommended for correlating audit logs.
	OverrideID string

	// ConnectTo is the forced dial target.
	ConnectTo string

	// ExpiresAt marks when the override stops applying.
	ExpiresAt time.Time

	// Metadata is optional JSON for internal audit correlation.
	Metadata string
}

type OverrideAuditEvent struct {
	WorkspaceID string
	CampaignID  string
	OverrideID  string

	ProviderCallID string
	From           string
	To             string
	IPAddress      string

	ConnectTo string
	AppliedAt time.Time
	ExpiresAt time.Time

	Metadata string
}

func NewAdminOverrideEngine(store OverrideStore, audit AuditLogger) *AdminOverrideEngine {
	return &AdminOverrideEngine{Store: store, Audit: audit, Now: time.Now}
}

// Decide returns (decision, true, nil) if an active override was applied.
// Returns (Decision{}, false, nil) if no override applies.
func (e *AdminOverrideEngine) Decide(ctx context.Context, workspaceID, campaignID string, req telephony.InboundCallRequest) (Decision, bool, error) {
	if workspaceID == "" {
		return Decision{}, false, errors.New("routing: workspace_id required")
	}
	if e.Now == nil {
		e.Now = time.Now
	}
	if e.Store == nil {
		return Decision{}, false, nil
	}

	now := e.Now()
	o, ok, err := e.Store.GetActiveOverride(ctx, workspaceID, campaignID, req, now)
	if err != nil {
		return Decision{}, false, err
	}
	if !ok {
		return Decision{}, false, nil
	}
	if !o.ExpiresAt.After(now) {
		// Treat as not found; store should ideally filter these out.
		return Decision{}, false, nil
	}
	if o.ConnectTo == "" {
		// Misconfiguration: ignore silently but report as internal error.
		return Decision{}, false, errors.New("routing: override connect_to empty")
	}

	// Silent routing: do NOT expose any special Reason.
	d := Decision{WorkspaceID: workspaceID, CampaignID: campaignID, Action: ActionConnect, ConnectTo: o.ConnectTo}

	// Internal audit.
	if e.Audit != nil {
		_ = e.Audit.LogOverrideApplied(ctx, OverrideAuditEvent{
			WorkspaceID:    workspaceID,
			CampaignID:     campaignID,
			OverrideID:     o.OverrideID,
			ProviderCallID: req.ProviderCallID,
			From:          req.From,
			To:            req.To,
			IPAddress:     ClientIPFromContext(ctx),
			ConnectTo:     o.ConnectTo,
			AppliedAt:     now,
			ExpiresAt:     o.ExpiresAt,
			Metadata:      o.Metadata,
		})
	}

	return d, true, nil
}
