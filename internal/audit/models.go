package audit

import "time"

// Event is an immutable, append-only audit log record.
//
// Invariants:
// - Events are never updated or deleted.
// - workspace_id is required for tenancy isolation.
// - actor and ip capture are best-effort; do not block critical flows on audit failures.
//
// Storage recommendation (Postgres):
// - Table audit_events with an INSERT-only policy.
// - Optional: trigger to prevent UPDATE/DELETE.
// - Optional: partition by time for retention.

type Event struct {
	ID          string   `json:"id" db:"id"`
	WorkspaceID string   `json:"workspace_id" db:"workspace_id"`

	// Type indicates the business category of the audit record.
	Type EventType `json:"type" db:"type"`

	// ActorUserID is the authenticated user causing the event (if applicable).
	ActorUserID string `json:"actor_user_id,omitempty" db:"actor_user_id"`
	// ActorRole may include hidden roles.
	ActorRole string `json:"actor_role,omitempty" db:"actor_role"`

	// IPAddress should capture the original client IP when available.
	// Prefer X-Forwarded-For processing at the edge; store the resolved client IP here.
	IPAddress string `json:"ip_address,omitempty" db:"ip_address"`

	// Target identifiers (optional, depending on the event type).
	WalletID     string `json:"wallet_id,omitempty" db:"wallet_id"`
	CampaignID   string `json:"campaign_id,omitempty" db:"campaign_id"`
	CallID       string `json:"call_id,omitempty" db:"call_id"`
	OverrideID   string `json:"override_id,omitempty" db:"override_id"`

	// Message is a short human-readable description for internal ops.
	Message string `json:"message,omitempty" db:"message"`

	// Metadata is optional JSON for full details.
	Metadata string `json:"metadata,omitempty" db:"metadata"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type EventType string

const (
	EventTypeAdminAction EventType = "admin_action"
	EventTypeOverride    EventType = "routing_override"
)
