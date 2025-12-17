package routing

import (
	"context"
	"testing"
	"time"

	"telecom-platform/internal/telephony"
)

type memOverrideStore struct {
	over Override
	ok   bool
	err  error
}

func (m memOverrideStore) GetActiveOverride(ctx context.Context, workspaceID, campaignID string, req telephony.InboundCallRequest, now time.Time) (Override, bool, error) {
	return m.over, m.ok, m.err
}

type memAudit struct {
	called bool
	event  OverrideAuditEvent
}

func (m *memAudit) LogOverrideApplied(ctx context.Context, e OverrideAuditEvent) error {
	m.called = true
	m.event = e
	return nil
}

func TestAdminOverrideEngine_AppliesWhenActiveAndSilent(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()

	a := &memAudit{}
	e := NewAdminOverrideEngine(memOverrideStore{over: Override{WorkspaceID: "w", CampaignID: "c", ConnectTo: "sip:test", ExpiresAt: now.Add(5 * time.Minute)}, ok: true}, a)
	e.Now = func() time.Time { return now }

	dec, applied, err := e.Decide(context.Background(), "w", "c", telephony.InboundCallRequest{WorkspaceID: "w", ProviderCallID: "pc", From: "+1", To: "+2"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !applied {
		t.Fatalf("expected applied")
	}
	if dec.Action != ActionConnect || dec.ConnectTo != "sip:test" {
		t.Fatalf("unexpected decision: %+v", dec)
	}
	if dec.Reason != "" {
		t.Fatalf("expected silent decision (no reason), got %q", dec.Reason)
	}
	if !a.called {
		t.Fatalf("expected audit called")
	}
}

func TestAdminOverrideEngine_IgnoresExpired(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	e := NewAdminOverrideEngine(memOverrideStore{over: Override{WorkspaceID: "w", CampaignID: "c", ConnectTo: "sip:test", ExpiresAt: now.Add(-1 * time.Second)}, ok: true}, &memAudit{})
	e.Now = func() time.Time { return now }

	_, applied, err := e.Decide(context.Background(), "w", "c", telephony.InboundCallRequest{WorkspaceID: "w", ProviderCallID: "pc"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if applied {
		t.Fatalf("expected not applied")
	}
}
