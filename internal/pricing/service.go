package pricing

import (
	"context"
	"errors"
	"time"
)

// Service calculates costs based on workspace-scoped pricing.
//
// Contract:
// - Region-based pricing lookup (destination string acts as region/bucket)
// - Provider pricing is not exposed through this API (provider-specific rows may exist in storage,
//   but this service returns only the selected effective rate and the computed cost).
// - No telephony provider SDK calls.
// - Pure calculation + repository lookups.
type Service struct {
	repo RateRepository
	clock func() time.Time
}

func NewService(repo RateRepository) *Service {
	return &Service{repo: repo, clock: time.Now}
}

type CallCostRequest struct {
	WorkspaceID string
	Direction   CallDirection

	// Destination is the region/bucket identifier used for pricing resolution
	// (e.g., "US", "IN", "US-CA", "prefix:+1", destination_id).
	Destination string

	// DurationSeconds is the call duration in seconds (billable seconds are derived).
	DurationSeconds int

	// At determines which effective pricing to use. If zero, service clock is used.
	At time.Time
}

type CallCost struct {
	WorkspaceID string
	Direction   CallDirection
	Destination string

	Currency string

	BillableSeconds int
	BillableMinutes int

	RatePerMinuteMinor int64
	TotalMinor         int64
}

var (
	ErrPricingNotFound   = errors.New("pricing not found")
	ErrInvalidPricingReq = errors.New("invalid pricing request")
)

// CalculateCallCost computes the call cost for a given duration using region-based pricing.
func (s *Service) CalculateCallCost(ctx context.Context, req CallCostRequest) (CallCost, error) {
	if req.WorkspaceID == "" || req.Destination == "" {
		return CallCost{}, ErrInvalidPricingReq
	}
	if req.Direction != CallDirectionInbound && req.Direction != CallDirectionOutbound {
		return CallCost{}, ErrInvalidPricingReq
	}
	if req.DurationSeconds <= 0 {
		return CallCost{}, ErrInvalidPricingReq
	}

	at := req.At
	if at.IsZero() {
		at = s.clock().UTC()
	}

	mp, ok, err := s.repo.FindMinutePricing(ctx, req.WorkspaceID, req.Direction, req.Destination, at)
	if err != nil {
		return CallCost{}, err
	}
	if !ok {
		return CallCost{}, ErrPricingNotFound
	}

	billableSec := billableSeconds(req.DurationSeconds, mp.MinimumBillableSeconds, mp.BillingIncrementSeconds)
	billableMin := billableMinutesFromSeconds(billableSec)

	total := mp.RatePerMinuteMinor * int64(billableMin)

	return CallCost{
		WorkspaceID:        req.WorkspaceID,
		Direction:          req.Direction,
		Destination:        req.Destination,
		Currency:           mp.Currency,
		BillableSeconds:    billableSec,
		BillableMinutes:    billableMin,
		RatePerMinuteMinor: mp.RatePerMinuteMinor,
		TotalMinor:         total,
	}, nil
}

// RateRepository abstracts pricing persistence.
// Implementation can be Postgres, cached, etc.
//
// IMPORTANT: this interface intentionally does not return provider info.
type RateRepository interface {
	FindMinutePricing(ctx context.Context, workspaceID string, direction CallDirection, destination string, at time.Time) (MinutePricing, bool, error)
}

func billableSeconds(actualSec int, minSec int, incrementSec int) int {
	if actualSec < 0 {
		return 0
	}
	if minSec <= 0 {
		minSec = 0
	}
	if incrementSec <= 0 {
		incrementSec = 60
	}

	sec := actualSec
	if sec < minSec {
		sec = minSec
	}

	// round up to nearest increment
	q := sec / incrementSec
	r := sec % incrementSec
	if r != 0 {
		q++
	}
	return q * incrementSec
}

func billableMinutesFromSeconds(sec int) int {
	if sec <= 0 {
		return 0
	}
	m := sec / 60
	if sec%60 != 0 {
		m++
	}
	if m <= 0 {
		return 0
	}
	return m
}
