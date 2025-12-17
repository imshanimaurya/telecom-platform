package wallet

import "time"

// Wallet represents a tenant-scoped wallet.
// Invariant: available balance must be derived from immutable ledger entries.
// No code should ever mutate a "balance" without writing a corresponding ledger entry.
type Wallet struct {
	ID          string    `json:"id" db:"id"`
	WorkspaceID string    `json:"workspace_id" db:"workspace_id"`
	Currency    string    `json:"currency" db:"currency"`

	// Optional operational flags (do not encode money state here).
	Status WalletStatus `json:"status" db:"status"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type WalletStatus string

const (
	WalletStatusActive   WalletStatus = "active"
	WalletStatusDisabled WalletStatus = "disabled"
)

// WalletLedger is an immutable append-only entry.
// Each row represents a credit/debit posted to the wallet.
//
// Multi-tenant invariant: workspace_id required.
// Money invariant: any balance change MUST have a corresponding ledger entry.
type WalletLedger struct {
	ID          string `json:"id" db:"id"`
	WorkspaceID string `json:"workspace_id" db:"workspace_id"`
	WalletID    string `json:"wallet_id" db:"wallet_id"`

	// Type categorizes the ledger entry. Keep stable.
	Type LedgerEntryType `json:"type" db:"type"`

	// AmountMinor is the signed amount in minor units (e.g., cents).
	// Credits are positive, debits are negative.
	AmountMinor int64 `json:"amount_minor" db:"amount_minor"`
	Currency    string `json:"currency" db:"currency"`

	// ExternalRef is optional: call_id, invoice_id, provider_event_id, etc.
	ExternalRef string `json:"external_ref,omitempty" db:"external_ref"`

	// IdempotencyKey is required for safe retries of money-posting operations.
	IdempotencyKey string `json:"idempotency_key" db:"idempotency_key"`

	// Metadata is optional JSON for audit/debug (store as JSONB in Postgres).
	Metadata string `json:"metadata,omitempty" db:"metadata"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type LedgerEntryType string

const (
	LedgerEntryTypeCredit LedgerEntryType = "credit" // top-up, adjustment, etc.
	LedgerEntryTypeDebit  LedgerEntryType = "debit"  // usage charge, fee, etc.
	LedgerEntryTypeHold   LedgerEntryType = "hold"   // reservation (optional future)
	LedgerEntryTypeRelease LedgerEntryType = "release" // release reservation (optional future)
)

// AdminWalletAction tracks privileged/manual actions performed by admins.
// This is required for auditability (especially for hidden override capabilities).
//
// Note: This is not the ledger itself. Any admin mutation of money must also create
// a WalletLedger entry (or a pair of entries) to preserve money invariants.
type AdminWalletAction struct {
	ID          string `json:"id" db:"id"`
	WorkspaceID string `json:"workspace_id" db:"workspace_id"`
	WalletID    string `json:"wallet_id" db:"wallet_id"`

	AdminUserID string `json:"admin_user_id" db:"admin_user_id"`
	// AdminRole records the role at the time of action (may include hidden roles).
	AdminRole string `json:"admin_role" db:"admin_role"`

	Action AdminWalletActionType `json:"action" db:"action"`
	Reason string               `json:"reason,omitempty" db:"reason"`

	// AmountMinor is optional depending on the action.
	AmountMinor int64  `json:"amount_minor" db:"amount_minor"`
	Currency    string `json:"currency" db:"currency"`

	// RelatedLedgerID links to the ledger entry created by the action (if applicable).
	RelatedLedgerID string `json:"related_ledger_id,omitempty" db:"related_ledger_id"`

	// Metadata is optional JSON (store as JSONB).
	Metadata string `json:"metadata,omitempty" db:"metadata"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type AdminWalletActionType string

const (
	AdminWalletActionTypeAdjustBalance AdminWalletActionType = "adjust_balance"
	AdminWalletActionTypeFreeze        AdminWalletActionType = "freeze"
	AdminWalletActionTypeUnfreeze      AdminWalletActionType = "unfreeze"
)
