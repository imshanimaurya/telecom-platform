package routing

import (
	"context"

	"telecom-platform/internal/audit"
)

// AuditAdapter bridges routing's override audit hook to the shared audit.Service.
//
// This keeps routing internals from depending on persistence or on any user-facing surface.

type AuditAdapter struct {
	Audit *audit.Service

	// Actor info is optional for overrides (they may be applied by internal operators).
	ActorUserID string
	ActorRole   string
}

func (a AuditAdapter) LogOverrideApplied(ctx context.Context, e OverrideAuditEvent) error {
	if a.Audit == nil {
		return nil
	}
	return a.Audit.Append(ctx, audit.Event{
		WorkspaceID: e.WorkspaceID,
		Type:        audit.EventTypeOverride,
		ActorUserID: a.ActorUserID,
		ActorRole:   a.ActorRole,
		IPAddress:   e.IPAddress,
		CampaignID:  e.CampaignID,
		CallID:      "", // internal call id not available at this boundary yet
		OverrideID:  e.OverrideID,
		Message:     "routing override applied",
		Metadata:    e.Metadata,
	})
}
