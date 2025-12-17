package telephony

import (
	"context"
)

// SIPProvider is a stub adapter for SIP trunk / gateway integrations.
//
// Future FreeSWITCH integration (planned):
// - Inbound calls will arrive via FreeSWITCH ESL events or HTTP hooks from a gateway.
// - Outbound call control will be done via ESL (originate, bridge, hangup) or via a mediabroker.
// - Recordings can be started/stopped via FreeSWITCH APIs and then persisted to object storage.
// - CDRs should be sourced from FreeSWITCH CDR exports (e.g., XML/JSON CDR, event socket) and normalized.
//
// IMPORTANT:
// - Keep this adapter free of business logic.
// - It should only translate SIP/FreeSWITCH boundary events into internal types and delegate decisions
//   to internal/routing and internal/calls.
type SIPProvider struct{}

func (p *SIPProvider) Name() string { return "sip" }

func (p *SIPProvider) HealthCheck(ctx context.Context) error {
	return nil
}

func (p *SIPProvider) HandleInboundCall(ctx context.Context, req InboundCallRequest) (InboundCallResult, error) {
	return InboundCallResult{}, nil
}

func (p *SIPProvider) BuyNumber(ctx context.Context, req BuyNumberRequest) (BuyNumberResult, error) {
	return BuyNumberResult{}, nil
}

func (p *SIPProvider) ReleaseNumber(ctx context.Context, req ReleaseNumberRequest) (ReleaseNumberResult, error) {
	return ReleaseNumberResult{}, nil
}

func (p *SIPProvider) StartRecording(ctx context.Context, req StartRecordingRequest) (StartRecordingResult, error) {
	return StartRecordingResult{}, nil
}

func (p *SIPProvider) FetchCDR(ctx context.Context, req FetchCDRRequest) (FetchCDRResult, error) {
	return FetchCDRResult{}, nil
}
