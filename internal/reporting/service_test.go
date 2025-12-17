package reporting

import (
	"context"
	"testing"
	"time"

	"telecom-platform/internal/calls"
	"telecom-platform/internal/wallet"
)

func TestReporting_WorkspaceIsolation(t *testing.T) {
	repo := NewMemoryRepo()
	now := time.Unix(1700000000, 0).UTC()
	repo.Calls = []calls.Call{
		{CallID: "c1", WorkspaceID: "w1", CampaignID: "camp", Status: calls.CallStatusCompleted, DurationSeconds: 30, CreatedAt: now},
		{CallID: "c2", WorkspaceID: "w2", CampaignID: "camp", Status: calls.CallStatusCompleted, DurationSeconds: 50, CreatedAt: now},
	}
	svc := NewService(repo)

	out, err := svc.CallsSummary(context.Background(), CallsSummaryRequest{WorkspaceID: "w1", Range: TimeRange{From: now.Add(-time.Hour), To: now.Add(time.Hour)}})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.TotalCalls != 1 {
		t.Fatalf("expected 1 call, got %d", out.TotalCalls)
	}
}

func TestReporting_SpendSummaryAggregates(t *testing.T) {
	repo := NewMemoryRepo()
	now := time.Unix(1700000000, 0).UTC()
	repo.Ledgers = []wallet.WalletLedger{
		{ID: "l1", WorkspaceID: "w", WalletID: "wa", Currency: "USD", AmountMinor: 1000, CreatedAt: now},
		{ID: "l2", WorkspaceID: "w", WalletID: "wa", Currency: "USD", AmountMinor: -200, ExternalRef: "call:c1", CreatedAt: now},
		{ID: "l3", WorkspaceID: "w", WalletID: "wa", Currency: "USD", AmountMinor: -50, ExternalRef: "call:c2", CreatedAt: now},
		{ID: "l4", WorkspaceID: "w", WalletID: "wa", Currency: "USD", AmountMinor: 25, ExternalRef: "admin_manual_credit", CreatedAt: now},
	}
	svc := NewService(repo)

	out, err := svc.SpendSummary(context.Background(), SpendSummaryRequest{WorkspaceID: "w", WalletID: "wa", Range: TimeRange{From: now.Add(-time.Hour), To: now.Add(time.Hour)}, Currency: "USD"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.TotalDebitMinor != 250 {
		t.Fatalf("expected total debit 250, got %d", out.TotalDebitMinor)
	}
	if out.TotalCreditMinor != 1025 {
		t.Fatalf("expected total credit 1025, got %d", out.TotalCreditMinor)
	}
	if out.NetDeltaMinor != 775 {
		t.Fatalf("expected net 775, got %d", out.NetDeltaMinor)
	}
}

func TestReporting_ConversionMetrics(t *testing.T) {
	repo := NewMemoryRepo()
	now := time.Unix(1700000000, 0).UTC()
	repo.Calls = []calls.Call{
		{CallID: "c1", WorkspaceID: "w", CampaignID: "camp", Status: calls.CallStatusCompleted, CreatedAt: now},
		{CallID: "c2", WorkspaceID: "w", CampaignID: "camp", Status: calls.CallStatusFailed, CreatedAt: now},
	}
	repo.Conversions["w|camp"] = 1

	svc := NewService(repo)
	m, err := svc.ConversionMetrics(context.Background(), ConversionMetricsRequest{WorkspaceID: "w", CampaignID: "camp", Range: TimeRange{From: now.Add(-time.Hour), To: now.Add(time.Hour)}})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if m.CallsAttempted != 2 || m.CallsConnected != 1 || m.Conversions != 1 {
		t.Fatalf("unexpected metrics: %+v", m)
	}
	if m.ConnectionRate == 0 || m.ConversionRate == 0 {
		t.Fatalf("expected non-zero rates")
	}
}
