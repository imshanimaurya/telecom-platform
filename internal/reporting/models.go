package reporting

import "time"

// Common filtering inputs.

type TimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// CallsSummaryRequest requests aggregated call metrics.
// Workspace isolation: WorkspaceID is required.

type CallsSummaryRequest struct {
	WorkspaceID string    `json:"workspace_id"`
	Range       TimeRange `json:"range"`
	CampaignID  string    `json:"campaign_id,omitempty"`
}

type CallsSummary struct {
	WorkspaceID string `json:"workspace_id"`
	CampaignID  string `json:"campaign_id,omitempty"`

	TotalCalls      int `json:"total_calls"`
	CompletedCalls  int `json:"completed_calls"`
	FailedCalls     int `json:"failed_calls"`
	NoAnswerCalls   int `json:"no_answer_calls"`
	BusyCalls       int `json:"busy_calls"`
	CanceledCalls   int `json:"canceled_calls"`
	InProgressCalls int `json:"in_progress_calls"`

	TotalDurationSeconds int `json:"total_duration_seconds"`
	AverageDurationSeconds int `json:"average_duration_seconds"`

	RecordedCalls int `json:"recorded_calls"`
}

// SpendSummaryRequest requests aggregated spend metrics.
// Spend is derived from immutable wallet ledger entries (debits) scoped to workspace.

type SpendSummaryRequest struct {
	WorkspaceID string    `json:"workspace_id"`
	Range       TimeRange `json:"range"`
	WalletID    string    `json:"wallet_id,omitempty"`
	Currency    string    `json:"currency,omitempty"`
}

type SpendSummary struct {
	WorkspaceID string `json:"workspace_id"`
	WalletID    string `json:"wallet_id,omitempty"`
	Currency    string `json:"currency"`

	TotalDebitMinor int64 `json:"total_debit_minor"`
	TotalCreditMinor int64 `json:"total_credit_minor"`
	NetDeltaMinor   int64 `json:"net_delta_minor"`

	UsageDebitMinor int64 `json:"usage_debit_minor"`
	AdminAdjustMinor int64 `json:"admin_adjust_minor"`
}

// ConversionMetricsRequest captures simple campaign conversion metrics.
// Since campaigns module is not implemented yet, this is intentionally minimal.

type ConversionMetricsRequest struct {
	WorkspaceID string    `json:"workspace_id"`
	Range       TimeRange `json:"range"`
	CampaignID  string    `json:"campaign_id"`
}

type ConversionMetrics struct {
	WorkspaceID string `json:"workspace_id"`
	CampaignID  string `json:"campaign_id"`

	CallsAttempted int `json:"calls_attempted"`
	CallsConnected int `json:"calls_connected"`
	Conversions    int `json:"conversions"`

	ConnectionRate float64 `json:"connection_rate"`
	ConversionRate float64 `json:"conversion_rate"`
}
