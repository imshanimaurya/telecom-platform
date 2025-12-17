package telephony

import (
	"context"
	"time"
)

// TelephonyProvider defines the provider-agnostic interface used by business logic.
//
// Rules:
// - No provider SDK calls outside telephony adapters.
// - All requests must be workspace-scoped (workspace_id required).
// - Keep request/response types provider-agnostic; store provider raw payloads in metadata if needed.
type TelephonyProvider interface {
	Name() string
	HealthCheck(ctx context.Context) error

	HandleInboundCall(ctx context.Context, req InboundCallRequest) (InboundCallResult, error)

	BuyNumber(ctx context.Context, req BuyNumberRequest) (BuyNumberResult, error)
	ReleaseNumber(ctx context.Context, req ReleaseNumberRequest) (ReleaseNumberResult, error)

	StartRecording(ctx context.Context, req StartRecordingRequest) (StartRecordingResult, error)
	FetchCDR(ctx context.Context, req FetchCDRRequest) (FetchCDRResult, error)
}

// InboundCallRequest represents an inbound call event received from a provider.
type InboundCallRequest struct {
	WorkspaceID string `json:"workspace_id"`

	// ProviderCallID is the provider's unique identifier for this call.
	ProviderCallID string `json:"provider_call_id"`

	// From and To are E.164 where possible.
	From string `json:"from"`
	To   string `json:"to"`

	// OccurredAt is the provider event time.
	OccurredAt time.Time `json:"occurred_at"`

	// RawPayload is optional for debugging/audit; store as JSON string.
	RawPayload string `json:"raw_payload,omitempty"`
}

// InboundCallResult is the provider adapter response used to drive next steps.
type InboundCallResult struct {
	WorkspaceID string `json:"workspace_id"`
	CallID      string `json:"call_id"` // internal call identifier if created

	// Action describes what should happen next at the provider boundary.
	Action InboundCallAction `json:"action"`

	// ConnectTo is used when Action == "connect".
	ConnectTo string `json:"connect_to,omitempty"`
}

type InboundCallAction string

const (
	InboundCallActionReject  InboundCallAction = "reject"
	InboundCallActionConnect InboundCallAction = "connect"
	InboundCallActionHangup  InboundCallAction = "hangup"
)

type BuyNumberRequest struct {
	WorkspaceID string `json:"workspace_id"`

	CountryISO2 string `json:"country_iso2"`
	NumberType  string `json:"number_type"`

	// DesiredNumber is optional; if empty, provider selects best available.
	DesiredNumber string `json:"desired_number,omitempty"`

	// Metadata is optional JSON.
	Metadata string `json:"metadata,omitempty"`
}

type BuyNumberResult struct {
	WorkspaceID string `json:"workspace_id"`

	// Number is the purchased number (E.164).
	Number string `json:"number"`

	ProviderNumberID string `json:"provider_number_id"`
}

type ReleaseNumberRequest struct {
	WorkspaceID string `json:"workspace_id"`

	// Identify number to release.
	Number           string `json:"number"`
	ProviderNumberID string `json:"provider_number_id,omitempty"`
}

type ReleaseNumberResult struct {
	WorkspaceID string `json:"workspace_id"`
	Released    bool   `json:"released"`
}

type StartRecordingRequest struct {
	WorkspaceID string `json:"workspace_id"`

	// ProviderCallID identifies the call at the provider.
	ProviderCallID string `json:"provider_call_id"`
	// CallID is the internal call id if already created.
	CallID string `json:"call_id,omitempty"`

	// Metadata is optional JSON.
	Metadata string `json:"metadata,omitempty"`
}

type StartRecordingResult struct {
	WorkspaceID string `json:"workspace_id"`

	ProviderRecordingID string `json:"provider_recording_id"`
	Started            bool   `json:"started"`
}

type FetchCDRRequest struct {
	WorkspaceID string `json:"workspace_id"`

	// Query by time window and/or call IDs.
	From time.Time `json:"from"`
	To   time.Time `json:"to"`

	// Optional filters.
	ProviderCallID string `json:"provider_call_id,omitempty"`
}

type FetchCDRResult struct {
	WorkspaceID string `json:"workspace_id"`
	Records     []CDR  `json:"records"`
}

// CDR is a provider-agnostic call detail record.
type CDR struct {
	ProviderCallID string `json:"provider_call_id"`
	From           string `json:"from"`
	To             string `json:"to"`

	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`

	DurationSeconds int `json:"duration_seconds"`

	// CostMinor is optional if provider provides it; internal pricing is preferred.
	CostMinor int64  `json:"cost_minor,omitempty"`
	Currency  string `json:"currency,omitempty"`

	// Raw is optional JSON.
	Raw string `json:"raw,omitempty"`
}
