package telephony

import (
	"context"
	"errors"

	"telecom-platform/internal/routing"
)

// TwilioProvider is a placeholder implementation.
// TODO: wire in Twilio REST client + credentials from config.
type TwilioProvider struct {
	router routing.Engine
}

func NewTwilioProvider(router routing.Engine) *TwilioProvider {
	return &TwilioProvider{router: router}
}

func (p *TwilioProvider) Name() string { return "twilio" }

func (p *TwilioProvider) HealthCheck(ctx context.Context) error {
	// TODO: call a lightweight Twilio endpoint.
	return nil
}

func (p *TwilioProvider) HandleInboundCall(ctx context.Context, req InboundCallRequest) (InboundCallResult, error) {
	if p.router == nil {
		return InboundCallResult{}, errors.New("telephony: twilio router is nil")
	}
	return p.router.RouteInboundCall(ctx, req)
}

func (p *TwilioProvider) BuyNumber(ctx context.Context, req BuyNumberRequest) (BuyNumberResult, error) {
	return BuyNumberResult{}, errors.New("telephony: twilio BuyNumber not implemented")
}

func (p *TwilioProvider) ReleaseNumber(ctx context.Context, req ReleaseNumberRequest) (ReleaseNumberResult, error) {
	return ReleaseNumberResult{}, errors.New("telephony: twilio ReleaseNumber not implemented")
}

func (p *TwilioProvider) StartRecording(ctx context.Context, req StartRecordingRequest) (StartRecordingResult, error) {
	return StartRecordingResult{}, errors.New("telephony: twilio StartRecording not implemented")
}

func (p *TwilioProvider) FetchCDR(ctx context.Context, req FetchCDRRequest) (FetchCDRResult, error) {
	return FetchCDRResult{}, errors.New("telephony: twilio FetchCDR not implemented")
}
