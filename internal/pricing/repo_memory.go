package pricing

import (
	"context"
	"time"
)

// MemoryRepo is a simple in-memory repository useful for tests and early development.
// It is workspace-scoped and supports exact destination matches.
//
// NOTE: This is not intended for production; replace with Postgres implementation.
type MemoryRepo struct {
	Minute []MinutePricing
}

func (r *MemoryRepo) FindMinutePricing(ctx context.Context, workspaceID string, direction CallDirection, destination string, at time.Time) (MinutePricing, bool, error) {
	_ = ctx

	// Prefer the most recent effective pricing row.
	var best MinutePricing
	found := false

	for _, p := range r.Minute {
		if p.WorkspaceID != workspaceID {
			continue
		}
		if p.Direction != direction {
			continue
		}
		if p.Destination != destination {
			continue
		}
		if p.Status != PricingStatusActive {
			continue
		}
		if at.Before(p.EffectiveFrom) {
			continue
		}
		if p.EffectiveTo != nil && !at.Before(*p.EffectiveTo) {
			continue
		}

		if !found || p.EffectiveFrom.After(best.EffectiveFrom) {
			best = p
			found = true
		}
	}

	return best, found, nil
}
