package routing

import (
	"context"
	"errors"

	"telecom-platform/internal/telephony"
)

// Engine decides what to do with an inbound call.
//
// IMPORTANT: This is an interface-only contract for now.
// Business rules and persistence belong to internal/routing (and later internal/calls),
// but provider adapters must only depend on this abstraction.
//
// This keeps provider-specific HTTP/webhook code free of business logic.
//
// Multi-tenancy: req.WorkspaceID must always be set.
//
// Returns a telephony.InboundCallResult describing the provider boundary action.
// Example actions: connect to a SIP URI, reject, or hangup.
//
// NOTE: A concrete implementation can be added later without changing adapters.
// For now, NewNoopEngine can be used in main wiring.
//
//go:generate echo "no codegen"

type Engine interface {
	RouteInboundCall(ctx context.Context, req telephony.InboundCallRequest) (telephony.InboundCallResult, error)
}

// NewNoopEngine returns an engine that always rejects.
func NewNoopEngine() Engine { return noopEngine{} }

type noopEngine struct{}

func (noopEngine) RouteInboundCall(ctx context.Context, req telephony.InboundCallRequest) (telephony.InboundCallResult, error) {
	if req.WorkspaceID == "" {
		return telephony.InboundCallResult{}, errors.New("routing: workspace_id required")
	}
	return telephony.InboundCallResult{WorkspaceID: req.WorkspaceID, Action: telephony.InboundCallActionReject}, nil
}

// NewEngineAdapter adapts the richer RoutingEngine (Decision-based) into the
// minimal provider-facing Engine interface.
//
// This allows provider adapters to stay stable while routing evolves.
func NewEngineAdapter(engine *RoutingEngine, opts AdapterOptions) Engine {
	return engineAdapter{engine: engine, opts: opts}
}

type AdapterOptions struct {
	// CampaignIDResolver resolves the campaign used for this inbound request.
	// For example, it may map dialed number to campaign.
	CampaignIDResolver func(ctx context.Context, req telephony.InboundCallRequest) (campaignID string, err error)

	// WalletContextResolver resolves wallet + estimated charge (optional).
	WalletContextResolver func(ctx context.Context, req telephony.InboundCallRequest) (walletID string, estMinor int64, currency string, err error)

	// RoleResolver resolves actor role (for admin override decisions).
	RoleResolver func(ctx context.Context, req telephony.InboundCallRequest) (role string, err error)
}

type engineAdapter struct {
	engine *RoutingEngine
	opts   AdapterOptions
}

func (a engineAdapter) RouteInboundCall(ctx context.Context, req telephony.InboundCallRequest) (telephony.InboundCallResult, error) {
	if a.engine == nil {
		return telephony.InboundCallResult{}, errors.New("routing: engine is nil")
	}

	campaignID := ""
	if a.opts.CampaignIDResolver != nil {
		cid, err := a.opts.CampaignIDResolver(ctx, req)
		if err != nil {
			return telephony.InboundCallResult{}, err
		}
		campaignID = cid
	}

	walletID := ""
	var estMinor int64
	currency := ""
	if a.opts.WalletContextResolver != nil {
		wid, est, cur, err := a.opts.WalletContextResolver(ctx, req)
		if err != nil {
			return telephony.InboundCallResult{}, err
		}
		walletID, estMinor, currency = wid, est, cur
	}

	role := ""
	if a.opts.RoleResolver != nil {
		r, err := a.opts.RoleResolver(ctx, req)
		if err != nil {
			return telephony.InboundCallResult{}, err
		}
		role = r
	}

	d, err := a.engine.Route(ctx, RouteInput{
		WorkspaceID:    req.WorkspaceID,
		CampaignID:     campaignID,
		ActorRole:      role,
		WalletID:       walletID,
		EstimatedMinor: estMinor,
		Currency:       currency,
		Inbound:        req,
	})
	if err != nil {
		return telephony.InboundCallResult{}, err
	}

	res := telephony.InboundCallResult{WorkspaceID: d.WorkspaceID, CallID: ""}
	switch d.Action {
	case ActionReject:
		res.Action = telephony.InboundCallActionReject
	case ActionHangup:
		res.Action = telephony.InboundCallActionHangup
	case ActionConnect:
		res.Action = telephony.InboundCallActionConnect
		res.ConnectTo = d.ConnectTo
	default:
		return telephony.InboundCallResult{}, errors.New("routing: unknown decision action")
	}

	return res, nil
}
