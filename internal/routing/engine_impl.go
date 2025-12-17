package routing

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"telecom-platform/internal/rbac"
	"telecom-platform/internal/telephony"
	"telecom-platform/internal/wallet"
)

// RoutingEngine evaluates routing for inbound/outbound call attempts.
//
// Priority:
//  1) Admin override
//  2) Wallet balance
//  3) Campaign rules
//  4) Weighted destination selection
//
// Return routing decision only. No side effects (no DB writes, no provider calls).
//
// Notes:
// - Admin override means privileged actor can force connect even if wallet/campaign would block.
// - Wallet balance check can block (reject) when insufficient.
// - Campaign rules can block or restrict destinations.
// - Weighted selection chooses a destination when multiple are eligible.

type RoutingEngine struct {
	Overrides *AdminOverrideEngine

	Wallet wallet.BalanceService
	Campaigns CampaignService

	RNG *rand.Rand
	Now func() time.Time
}

// CampaignService is the minimal abstraction needed to evaluate campaign rules.
// A real implementation can live in internal/campaigns and use persistence.
//
// It returns campaign config and a set of eligible destinations for this call.
// If the campaign is not found or not allowed, it should return a typed error in future;
// for now we use generic errors and let the engine decide to reject.

type CampaignService interface {
	EvaluateInbound(ctx context.Context, workspaceID, campaignID string, req telephony.InboundCallRequest) (CampaignEvaluation, error)
}

type CampaignEvaluation struct {
	Allowed bool
	Reason  string

	Destinations []WeightedDestination
}

type WeightedDestination struct {
	// TargetURI is a provider-agnostic dial target.
	// Examples:
	// - sip:agent-123@pbx.example.com
	// - +15551234567
	TargetURI string

	// Weight must be > 0.
	Weight int
}

type RouteInput struct {
	WorkspaceID string
	CampaignID  string

	// ActorRole participates in admin override.
	ActorRole string

	WalletID        string
	EstimatedMinor  int64
	Currency        string

	Inbound telephony.InboundCallRequest
}

func NewRoutingEngine(walletSvc wallet.BalanceService, campaigns CampaignService, rng *rand.Rand) *RoutingEngine {
	return &RoutingEngine{Wallet: walletSvc, Campaigns: campaigns, RNG: rng, Now: time.Now}
}

func (e *RoutingEngine) Route(ctx context.Context, in RouteInput) (Decision, error) {
	if in.WorkspaceID == "" {
		return Decision{}, errors.New("routing: workspace_id required")
	}

	// 0) Silent, expiry-based overrides (no user visibility)
	if e.Overrides != nil {
		d, applied, err := e.Overrides.Decide(ctx, in.WorkspaceID, in.CampaignID, in.Inbound)
		if err != nil {
			return Decision{}, err
		}
		if applied {
			return d, nil
		}
	}

	// 1) Admin override
	if rbac.IsSuperAdmin(in.ActorRole) || in.ActorRole == rbac.RoleNetworkOperator {
		// Still need a destination. If campaign logic exists, use it, but do not block.
		if in.CampaignID != "" && e.Campaigns != nil {
			ev, err := e.Campaigns.EvaluateInbound(ctx, in.WorkspaceID, in.CampaignID, in.Inbound)
			if err == nil {
				if dest, ok := e.pickDestination(ev.Destinations); ok {
					return Decision{WorkspaceID: in.WorkspaceID, CampaignID: in.CampaignID, Action: ActionConnect, ConnectTo: dest, Reason: "admin_override"}, nil
				}
			}
		}
		// Fallback: reject (no eligible destination).
		return Decision{WorkspaceID: in.WorkspaceID, CampaignID: in.CampaignID, Action: ActionReject, Reason: "admin_override_no_destination"}, nil
	}

	// 2) Wallet balance
	if in.EstimatedMinor > 0 {
		if e.Wallet == nil {
			return Decision{}, errors.New("routing: wallet service not configured")
		}
		if in.WalletID == "" {
			return Decision{}, errors.New("routing: wallet_id required when estimated cost is provided")
		}
		if in.Currency == "" {
			return Decision{}, errors.New("routing: currency required when estimated cost is provided")
		}

		bal, err := e.Wallet.GetBalance(ctx, in.WorkspaceID, in.WalletID)
		if err != nil {
			return Decision{}, err
		}
		if bal.Currency != in.Currency {
			return Decision{WorkspaceID: in.WorkspaceID, CampaignID: in.CampaignID, Action: ActionReject, Reason: "wallet_currency_mismatch"}, nil
		}
		if bal.BalanceMinor < in.EstimatedMinor {
			return Decision{WorkspaceID: in.WorkspaceID, CampaignID: in.CampaignID, Action: ActionReject, Reason: "insufficient_balance"}, nil
		}
	}

	// 3) Campaign rules
	if in.CampaignID == "" {
		return Decision{WorkspaceID: in.WorkspaceID, Action: ActionReject, Reason: "campaign_id_required"}, nil
	}
	if e.Campaigns == nil {
		return Decision{}, errors.New("routing: campaign service not configured")
	}

	ev, err := e.Campaigns.EvaluateInbound(ctx, in.WorkspaceID, in.CampaignID, in.Inbound)
	if err != nil {
		return Decision{}, err
	}
	if !ev.Allowed {
		reason := ev.Reason
		if reason == "" {
			reason = "campaign_blocked"
		}
		return Decision{WorkspaceID: in.WorkspaceID, CampaignID: in.CampaignID, Action: ActionReject, Reason: reason}, nil
	}

	// 4) Weighted destination selection
	if dest, ok := e.pickDestination(ev.Destinations); ok {
		return Decision{WorkspaceID: in.WorkspaceID, CampaignID: in.CampaignID, Action: ActionConnect, ConnectTo: dest, Reason: "selected"}, nil
	}
	return Decision{WorkspaceID: in.WorkspaceID, CampaignID: in.CampaignID, Action: ActionReject, Reason: "no_eligible_destination"}, nil
}

func (e *RoutingEngine) pickDestination(dests []WeightedDestination) (string, bool) {
	var total int
	for _, d := range dests {
		if d.Weight <= 0 {
			continue
		}
		total += d.Weight
	}
	if total <= 0 {
		return "", false
	}

	rng := e.RNG
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	r := rng.Intn(total) // 0..total-1

	var acc int
	for _, d := range dests {
		if d.Weight <= 0 {
			continue
		}
		acc += d.Weight
		if r < acc {
			return d.TargetURI, true
		}
	}
	return "", false
}
