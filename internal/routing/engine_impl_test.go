package routing

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"telecom-platform/internal/rbac"
	"telecom-platform/internal/telephony"
	"telecom-platform/internal/wallet"
)

type stubWallet struct {
	bal wallet.Balance
	err error
}

func (s stubWallet) GetBalance(ctx context.Context, workspaceID, walletID string) (wallet.Balance, error) {
	return s.bal, s.err
}

type stubCampaigns struct {
	ev CampaignEvaluation
	err error
}

func (s stubCampaigns) EvaluateInbound(ctx context.Context, workspaceID, campaignID string, req telephony.InboundCallRequest) (CampaignEvaluation, error) {
	return s.ev, s.err
}

func TestRoutingEngine_AdminOverrideWins(t *testing.T) {
	e := NewRoutingEngine(stubWallet{bal: wallet.Balance{Currency: "USD", BalanceMinor: 0}}, stubCampaigns{ev: CampaignEvaluation{Allowed: false, Reason: "blocked"}}, rand.New(rand.NewSource(1)))

	d, err := e.Route(context.Background(), RouteInput{
		WorkspaceID:   "w",
		CampaignID:    "c",
		ActorRole:     rbac.RoleSuperAdmin,
		Inbound:       telephony.InboundCallRequest{WorkspaceID: "w", ProviderCallID: "p", From: "+1", To: "+2"},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if d.Action != ActionReject {
		// With no destinations, admin override cannot connect.
		t.Fatalf("expected reject when there are no destinations; got %q", d.Action)
	}
}

func TestRoutingEngine_InsufficientBalanceRejects(t *testing.T) {
	e := NewRoutingEngine(stubWallet{bal: wallet.Balance{Currency: "USD", BalanceMinor: 1}}, stubCampaigns{ev: CampaignEvaluation{Allowed: true, Destinations: []WeightedDestination{{TargetURI: "+1555", Weight: 1}}}}, rand.New(rand.NewSource(1)))

	d, err := e.Route(context.Background(), RouteInput{
		WorkspaceID:    "w",
		CampaignID:     "c",
		WalletID:       "wallet",
		EstimatedMinor: 10,
		Currency:       "USD",
		Inbound:        telephony.InboundCallRequest{WorkspaceID: "w", ProviderCallID: "p", From: "+1", To: "+2"},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if d.Action != ActionReject {
		t.Fatalf("expected reject, got %q", d.Action)
	}
	if d.Reason != "insufficient_balance" {
		t.Fatalf("expected insufficient_balance reason, got %q", d.Reason)
	}
}

func TestRoutingEngine_CampaignRulesThenWeightedPick(t *testing.T) {
	e := NewRoutingEngine(nil, stubCampaigns{ev: CampaignEvaluation{Allowed: true, Destinations: []WeightedDestination{{TargetURI: "sip:a", Weight: 1}, {TargetURI: "sip:b", Weight: 3}}}}, rand.New(rand.NewSource(1)))

	d, err := e.Route(context.Background(), RouteInput{
		WorkspaceID: "w",
		CampaignID:  "c",
		Inbound:     telephony.InboundCallRequest{WorkspaceID: "w", ProviderCallID: "p", From: "+1", To: "+2", OccurredAt: time.Now()},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if d.Action != ActionConnect {
		t.Fatalf("expected connect, got %q", d.Action)
	}
	if d.ConnectTo == "" {
		t.Fatalf("expected connect_to")
	}
}
