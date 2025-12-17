package wallet

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// NOTE: This repository assumes the following tables exist:
// - wallets
// - wallet_ledger (immutable append-only)
// - wallet_balances (projection)
// - admin_wallet_actions
//
// It also assumes an idempotency constraint, e.g.:
// UNIQUE (wallet_id, idempotency_key)

func lockWallet(ctx context.Context, tx *sql.Tx, workspaceID, walletID string) (Wallet, error) {
	// Lock the wallet row to serialize concurrent money operations per wallet.
	const q = `
SELECT id, workspace_id, currency, status, created_at, updated_at
FROM wallets
WHERE workspace_id = $1 AND id = $2
FOR UPDATE
`
	var w Wallet
	if err := tx.QueryRowContext(ctx, q, workspaceID, walletID).Scan(
		&w.ID,
		&w.WorkspaceID,
		&w.Currency,
		&w.Status,
		&w.CreatedAt,
		&w.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Wallet{}, ErrNotFound
		}
		return Wallet{}, err
	}
	return w, nil
}

func getBalance(ctx context.Context, db *sql.DB, workspaceID, walletID string) (Balance, error) {
	const q = `
SELECT workspace_id, wallet_id, currency, balance_minor, updated_at
FROM wallet_balances
WHERE workspace_id = $1 AND wallet_id = $2
`
	var b Balance
	if err := db.QueryRowContext(ctx, q, workspaceID, walletID).Scan(
		&b.WorkspaceID,
		&b.WalletID,
		&b.Currency,
		&b.BalanceMinor,
		&b.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Balance{}, ErrNotFound
		}
		return Balance{}, err
	}
	return b, nil
}

func getBalanceTx(ctx context.Context, tx *sql.Tx, workspaceID, walletID string) (Balance, error) {
	const q = `
SELECT workspace_id, wallet_id, currency, balance_minor, updated_at
FROM wallet_balances
WHERE workspace_id = $1 AND wallet_id = $2
`
	var b Balance
	if err := tx.QueryRowContext(ctx, q, workspaceID, walletID).Scan(
		&b.WorkspaceID,
		&b.WalletID,
		&b.Currency,
		&b.BalanceMinor,
		&b.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Balance{}, ErrNotFound
		}
		return Balance{}, err
	}
	return b, nil
}

func getBalanceForUpdate(ctx context.Context, tx *sql.Tx, workspaceID, walletID string) (Balance, error) {
	const q = `
SELECT workspace_id, wallet_id, currency, balance_minor, updated_at
FROM wallet_balances
WHERE workspace_id = $1 AND wallet_id = $2
FOR UPDATE
`
	var b Balance
	if err := tx.QueryRowContext(ctx, q, workspaceID, walletID).Scan(
		&b.WorkspaceID,
		&b.WalletID,
		&b.Currency,
		&b.BalanceMinor,
		&b.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Balance{}, ErrNotFound
		}
		return Balance{}, err
	}
	return b, nil
}

func findLedgerByIdempotency(ctx context.Context, tx *sql.Tx, workspaceID, walletID, key string) (WalletLedger, bool, error) {
	const q = `
SELECT id, workspace_id, wallet_id, type, amount_minor, currency, external_ref, idempotency_key, metadata, created_at
FROM wallet_ledger
WHERE workspace_id = $1 AND wallet_id = $2 AND idempotency_key = $3
LIMIT 1
`
	var e WalletLedger
	err := tx.QueryRowContext(ctx, q, workspaceID, walletID, key).Scan(
		&e.ID,
		&e.WorkspaceID,
		&e.WalletID,
		&e.Type,
		&e.AmountMinor,
		&e.Currency,
		&e.ExternalRef,
		&e.IdempotencyKey,
		&e.Metadata,
		&e.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return WalletLedger{}, false, nil
		}
		return WalletLedger{}, false, err
	}
	return e, true, nil
}

func insertLedger(ctx context.Context, tx *sql.Tx, e WalletLedger) error {
	const q = `
INSERT INTO wallet_ledger (
  id, workspace_id, wallet_id, type, amount_minor, currency, external_ref, idempotency_key, metadata, created_at
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,$8,$9,$10
)
`
	_, err := tx.ExecContext(ctx, q,
		e.ID,
		e.WorkspaceID,
		e.WalletID,
		e.Type,
		e.AmountMinor,
		e.Currency,
		e.ExternalRef,
		e.IdempotencyKey,
		e.Metadata,
		e.CreatedAt,
	)
	return err
}

func applyBalanceDelta(ctx context.Context, tx *sql.Tx, workspaceID, walletID, currency string, deltaMinor int64, now time.Time) (Balance, error) {
	// Upsert the balance row. We keep currency stable. If currency mismatch happens,
	// the wallet lock + service-level currency check should prevent inconsistencies.
	const q = `
INSERT INTO wallet_balances (workspace_id, wallet_id, currency, balance_minor, updated_at)
VALUES ($1,$2,$3,$4,$5)
ON CONFLICT (workspace_id, wallet_id)
DO UPDATE SET balance_minor = wallet_balances.balance_minor + EXCLUDED.balance_minor,
              updated_at = EXCLUDED.updated_at
RETURNING workspace_id, wallet_id, currency, balance_minor, updated_at
`
	var b Balance
	if err := tx.QueryRowContext(ctx, q, workspaceID, walletID, currency, deltaMinor, now).Scan(
		&b.WorkspaceID,
		&b.WalletID,
		&b.Currency,
		&b.BalanceMinor,
		&b.UpdatedAt,
	); err != nil {
		return Balance{}, err
	}
	return b, nil
}

func insertAdminAction(ctx context.Context, tx *sql.Tx, a AdminWalletAction) error {
	const q = `
INSERT INTO admin_wallet_actions (
  id, workspace_id, wallet_id, admin_user_id, admin_role, action, reason,
  amount_minor, currency, related_ledger_id, metadata, created_at
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12
)
`
	_, err := tx.ExecContext(ctx, q,
		a.ID,
		a.WorkspaceID,
		a.WalletID,
		a.AdminUserID,
		a.AdminRole,
		a.Action,
		a.Reason,
		a.AmountMinor,
		a.Currency,
		a.RelatedLedgerID,
		a.Metadata,
		a.CreatedAt,
	)
	return err
}

func findAdminActionByLedger(ctx context.Context, tx *sql.Tx, workspaceID, walletID, ledgerID string) (AdminWalletAction, bool, error) {
	const q = `
SELECT id, workspace_id, wallet_id, admin_user_id, admin_role, action, reason,
       amount_minor, currency, related_ledger_id, metadata, created_at
FROM admin_wallet_actions
WHERE workspace_id = $1 AND wallet_id = $2 AND related_ledger_id = $3
LIMIT 1
`
	var a AdminWalletAction
	err := tx.QueryRowContext(ctx, q, workspaceID, walletID, ledgerID).Scan(
		&a.ID,
		&a.WorkspaceID,
		&a.WalletID,
		&a.AdminUserID,
		&a.AdminRole,
		&a.Action,
		&a.Reason,
		&a.AmountMinor,
		&a.Currency,
		&a.RelatedLedgerID,
		&a.Metadata,
		&a.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AdminWalletAction{}, false, nil
		}
		return AdminWalletAction{}, false, err
	}
	return a, true, nil
}
