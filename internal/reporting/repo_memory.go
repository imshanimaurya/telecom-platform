package reporting

import (
	"context"
	"errors"
	"sync"
	"time"

	"telecom-platform/internal/calls"
	"telecom-platform/internal/wallet"
)

// MemoryRepo is a simple in-memory reporting repository for tests and early development.
// It enforces workspace isolation on reads.

type MemoryRepo struct {
	mu sync.Mutex

	Calls   []calls.Call
	Ledgers []wallet.WalletLedger

	Conversions map[string]int // key: workspace_id|campaign_id
}

func NewMemoryRepo() *MemoryRepo { return &MemoryRepo{Conversions: map[string]int{}} }

func (r *MemoryRepo) ListCalls(ctx context.Context, workspaceID string, from, to time.Time, campaignID string) ([]calls.Call, error) {
	if workspaceID == "" {
		return nil, errors.New("workspace_id required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]calls.Call, 0)
	for _, c := range r.Calls {
		if c.WorkspaceID != workspaceID {
			continue
		}
		if !c.CreatedAt.IsZero() {
			if c.CreatedAt.Before(from) || !c.CreatedAt.Before(to) {
				continue
			}
		}
		if campaignID != "" && c.CampaignID != campaignID {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

func (r *MemoryRepo) ListWalletLedger(ctx context.Context, workspaceID string, from, to time.Time, walletID string) ([]wallet.WalletLedger, error) {
	if workspaceID == "" {
		return nil, errors.New("workspace_id required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]wallet.WalletLedger, 0)
	for _, l := range r.Ledgers {
		if l.WorkspaceID != workspaceID {
			continue
		}
		if !l.CreatedAt.IsZero() {
			if l.CreatedAt.Before(from) || !l.CreatedAt.Before(to) {
				continue
			}
		}
		if walletID != "" && l.WalletID != walletID {
			continue
		}
		out = append(out, l)
	}
	return out, nil
}

func (r *MemoryRepo) ListConversions(ctx context.Context, workspaceID string, from, to time.Time, campaignID string) (int, error) {
	if workspaceID == "" {
		return 0, errors.New("workspace_id required")
	}
	if campaignID == "" {
		return 0, nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.Conversions[workspaceID+"|"+campaignID], nil
}
