package telephony

import (
	"context"
	"testing"
)

func TestSIPProvider_ImplementsTelephonyProvider(t *testing.T) {
	var _ TelephonyProvider = (*SIPProvider)(nil)
}

func TestSIPProvider_EmptyMethods(t *testing.T) {
	p := &SIPProvider{}
	ctx := context.Background()

	if _, err := p.HandleInboundCall(ctx, InboundCallRequest{WorkspaceID: "w", ProviderCallID: "c", From: "+1", To: "+2"}); err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if _, err := p.BuyNumber(ctx, BuyNumberRequest{WorkspaceID: "w", CountryISO2: "US", NumberType: "local"}); err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if _, err := p.ReleaseNumber(ctx, ReleaseNumberRequest{WorkspaceID: "w", Number: "+1555"}); err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if _, err := p.StartRecording(ctx, StartRecordingRequest{WorkspaceID: "w", ProviderCallID: "CA123"}); err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if _, err := p.FetchCDR(ctx, FetchCDRRequest{WorkspaceID: "w"}); err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
}
