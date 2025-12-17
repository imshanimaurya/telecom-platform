package pricing

import "time"

// Pricing models are tenant-scoped (workspace_id required everywhere).
// Amounts are expressed in minor units (e.g., cents) using int64.

// NumberPricing defines the price for purchasing/renewing or using a phone number.
type NumberPricing struct {
	ID          string `json:"id" db:"id"`
	WorkspaceID string `json:"workspace_id" db:"workspace_id"`

	// Provider is optional for provider-specific pricing, but business logic must remain provider-agnostic.
	Provider string `json:"provider,omitempty" db:"provider"`

	// CountryISO2 is the country of the phone number (e.g., "US", "IN").
	CountryISO2 string `json:"country_iso2" db:"country_iso2"`

	// NumberType examples: local, mobile, toll_free.
	NumberType string `json:"number_type" db:"number_type"`

	Currency string `json:"currency" db:"currency"`

	// SetupFeeMinor is a one-time purchase/activation fee.
	SetupFeeMinor int64 `json:"setup_fee_minor" db:"setup_fee_minor"`

	// MonthlyFeeMinor is a recurring rental fee.
	MonthlyFeeMinor int64 `json:"monthly_fee_minor" db:"monthly_fee_minor"`

	// Effective window for pricing.
	EffectiveFrom time.Time  `json:"effective_from" db:"effective_from"`
	EffectiveTo   *time.Time `json:"effective_to,omitempty" db:"effective_to"`

	Status PricingStatus `json:"status" db:"status"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// MinutePricing defines per-minute charges for calls.
type MinutePricing struct {
	ID          string `json:"id" db:"id"`
	WorkspaceID string `json:"workspace_id" db:"workspace_id"`

	Provider string `json:"provider,omitempty" db:"provider"`

	// Direction is inbound/outbound.
	Direction CallDirection `json:"direction" db:"direction"`

	// Destination identifies pricing region/route (e.g., country, prefix bucket, destination_id).
	Destination string `json:"destination" db:"destination"`

	Currency string `json:"currency" db:"currency"`

	// RatePerMinuteMinor is the price per started minute.
	RatePerMinuteMinor int64 `json:"rate_per_minute_minor" db:"rate_per_minute_minor"`

	// BillingIncrementSeconds (e.g., 60 for per-minute, 1 for per-second billing).
	BillingIncrementSeconds int `json:"billing_increment_seconds" db:"billing_increment_seconds"`

	// MinimumBillableSeconds enforces a minimum charge duration.
	MinimumBillableSeconds int `json:"minimum_billable_seconds" db:"minimum_billable_seconds"`

	EffectiveFrom time.Time  `json:"effective_from" db:"effective_from"`
	EffectiveTo   *time.Time `json:"effective_to,omitempty" db:"effective_to"`

	Status PricingStatus `json:"status" db:"status"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// RecordingPricing defines charges for call recordings (storage and/or processing).
type RecordingPricing struct {
	ID          string `json:"id" db:"id"`
	WorkspaceID string `json:"workspace_id" db:"workspace_id"`

	Provider string `json:"provider,omitempty" db:"provider"`

	Currency string `json:"currency" db:"currency"`

	// PricePerMinuteMinor is the recording cost per minute of recorded audio.
	PricePerMinuteMinor int64 `json:"price_per_minute_minor" db:"price_per_minute_minor"`

	// StoragePerGBMonthMinor is optional: recurring storage charges.
	StoragePerGBMonthMinor int64 `json:"storage_per_gb_month_minor" db:"storage_per_gb_month_minor"`

	EffectiveFrom time.Time  `json:"effective_from" db:"effective_from"`
	EffectiveTo   *time.Time `json:"effective_to,omitempty" db:"effective_to"`

	Status PricingStatus `json:"status" db:"status"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// TrackingPricing defines charges for call tracking features (e.g., tracking sessions/events).
type TrackingPricing struct {
	ID          string `json:"id" db:"id"`
	WorkspaceID string `json:"workspace_id" db:"workspace_id"`

	Currency string `json:"currency" db:"currency"`

	// PricePerEventMinor could apply to webhook events, tracking pings, analytics events, etc.
	PricePerEventMinor int64 `json:"price_per_event_minor" db:"price_per_event_minor"`

	// MonthlyFeeMinor is an optional flat fee for tracking add-on.
	MonthlyFeeMinor int64 `json:"monthly_fee_minor" db:"monthly_fee_minor"`

	EffectiveFrom time.Time  `json:"effective_from" db:"effective_from"`
	EffectiveTo   *time.Time `json:"effective_to,omitempty" db:"effective_to"`

	Status PricingStatus `json:"status" db:"status"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type PricingStatus string

const (
	PricingStatusActive   PricingStatus = "active"
	PricingStatusInactive PricingStatus = "inactive"
)

type CallDirection string

const (
	CallDirectionInbound  CallDirection = "inbound"
	CallDirectionOutbound CallDirection = "outbound"
)
