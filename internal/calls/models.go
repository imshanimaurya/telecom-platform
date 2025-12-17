package calls

import "time"

// Call represents a tenant-scoped phone call.
//
// Multi-tenant invariant: WorkspaceID is required on every row.
//
// NOTE: This is a domain model only. Provider-specific fields (like Twilio CallSid)
// should be stored as separate columns (e.g., provider_call_id) or metadata,
// not mixed into this provider-agnostic core model.
//
// Money invariant reminder: usage charging should reference call_id in the wallet ledger
// (external_ref) rather than mutating money fields here.

type Call struct {
	CallID      string `json:"call_id" db:"call_id"`
	WorkspaceID string `json:"workspace_id" db:"workspace_id"`
	CampaignID  string `json:"campaign_id,omitempty" db:"campaign_id"`

	From string `json:"from" db:"from"`
	To   string `json:"to" db:"to"`

	Status CallStatus `json:"status" db:"status"`

	// Duration is the call duration in seconds.
	// Keep as an int for JSON friendliness; store as INT in Postgres.
	DurationSeconds int `json:"duration" db:"duration"`

	RecordingURL string `json:"recording_url,omitempty" db:"recording_url"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type CallStatus string

const (
	CallStatusQueued    CallStatus = "queued"
	CallStatusRinging   CallStatus = "ringing"
	CallStatusInProgress CallStatus = "in_progress"
	CallStatusCompleted CallStatus = "completed"
	CallStatusFailed    CallStatus = "failed"
	CallStatusNoAnswer  CallStatus = "no_answer"
	CallStatusBusy      CallStatus = "busy"
	CallStatusCanceled  CallStatus = "canceled"
)
