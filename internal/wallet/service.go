package wallet

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"telecom-platform/pkg/utils"

	"github.com/google/uuid"
)

// Service provides wallet operations.
//
// Money invariants:
// - No balance updates without a ledger entry
// - Ledger is append-only (immutable)
// - All money operations must be executed in a DB transaction
//
// Tenancy invariant:
// - workspace_id is required and enforced in all queries
//
// Balance strategy:
// - Balance is stored in a projection table (wallet_balances) updated atomically
//   alongside ledger inserts.
type Service struct {
	db *sql.DB
	// clock is injectable for deterministic tests.
	clock func() time.Time
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db, clock: time.Now}
}

type Balance struct {
	WorkspaceID  string `json:"workspace_id"`
	WalletID     string `json:"wallet_id"`
	Currency     string `json:"currency"`
	BalanceMinor int64  `json:"balance_minor"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreditRequest struct {
	AmountMinor     int64  `json:"amount_minor"`
	Currency        string `json:"currency"`
	ExternalRef     string `json:"external_ref,omitempty"`
	IdempotencyKey  string `json:"idempotency_key"`
	Metadata        string `json:"metadata,omitempty"`
}

type DebitRequest struct {
	AmountMinor     int64  `json:"amount_minor"`
	Currency        string `json:"currency"`
	ExternalRef     string `json:"external_ref,omitempty"`
	IdempotencyKey  string `json:"idempotency_key"`
	Metadata        string `json:"metadata,omitempty"`
}

type AdminCreditRequest struct {
	AmountMinor     int64  `json:"amount_minor"`
	Currency        string `json:"currency"`
	Reason          string `json:"reason"`
	IdempotencyKey  string `json:"idempotency_key"`
	Metadata        string `json:"metadata,omitempty"`
}

var (
	ErrNotFound         = errors.New("not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrInvalidArgument  = errors.New("invalid argument")
)

func (s *Service) GetBalance(ctx context.Context, workspaceID, walletID string) (Balance, error) {
	if workspaceID == "" || walletID == "" {
		return Balance{}, ErrInvalidArgument
	}
	return getBalance(ctx, s.db, workspaceID, walletID)
}

func (s *Service) Credit(ctx context.Context, workspaceID, walletID string, req CreditRequest) (WalletLedger, Balance, error) {
	if err := validateMoneyReq(workspaceID, walletID, req.AmountMinor, req.Currency, req.IdempotencyKey); err != nil {
		return WalletLedger{}, Balance{}, err
	}
	if req.AmountMinor <= 0 {
		return WalletLedger{}, Balance{}, ErrInvalidArgument
	}

	now := s.clock().UTC()
	ledgerID := uuid.NewString()

	var outLedger WalletLedger
	var outBal Balance

	err := utils.WithTx(ctx, s.db, &sql.TxOptions{}, func(ctx context.Context, tx *sql.Tx) error {
		// Ensure wallet exists + currency matches.
		w, err := lockWallet(ctx, tx, workspaceID, walletID)
		if err != nil {
			return err
		}
		if w.Currency != req.Currency {
			return ErrInvalidArgument
		}

		// Idempotency: if a ledger entry already exists for this wallet+key, return it and the balance.
		if existing, ok, err := findLedgerByIdempotency(ctx, tx, workspaceID, walletID, req.IdempotencyKey); err != nil {
			return err
		} else if ok {
			outLedger = existing
			b, err := getBalanceTx(ctx, tx, workspaceID, walletID)
			if err != nil {
				return err
			}
			outBal = b
			return nil
		}

		entry := WalletLedger{
			ID:             ledgerID,
			WorkspaceID:    workspaceID,
			WalletID:       walletID,
			Type:           LedgerEntryTypeCredit,
			AmountMinor:    req.AmountMinor,
			Currency:       req.Currency,
			ExternalRef:    req.ExternalRef,
			IdempotencyKey: req.IdempotencyKey,
			Metadata:       req.Metadata,
			CreatedAt:      now,
		}
		if err := insertLedger(ctx, tx, entry); err != nil {
			return err
		}

		// Projection update.
		b, err := applyBalanceDelta(ctx, tx, workspaceID, walletID, req.Currency, req.AmountMinor, now)
		if err != nil {
			return err
		}
		outLedger = entry
		outBal = b
		return nil
	})

	return outLedger, outBal, err
}

func (s *Service) Debit(ctx context.Context, workspaceID, walletID string, req DebitRequest) (WalletLedger, Balance, error) {
	if err := validateMoneyReq(workspaceID, walletID, req.AmountMinor, req.Currency, req.IdempotencyKey); err != nil {
		return WalletLedger{}, Balance{}, err
	}
	if req.AmountMinor <= 0 {
		return WalletLedger{}, Balance{}, ErrInvalidArgument
	}

	now := s.clock().UTC()
	ledgerID := uuid.NewString()

	var outLedger WalletLedger
	var outBal Balance

	err := utils.WithTx(ctx, s.db, &sql.TxOptions{}, func(ctx context.Context, tx *sql.Tx) error {
		w, err := lockWallet(ctx, tx, workspaceID, walletID)
		if err != nil {
			return err
		}
		if w.Currency != req.Currency {
			return ErrInvalidArgument
		}

		if existing, ok, err := findLedgerByIdempotency(ctx, tx, workspaceID, walletID, req.IdempotencyKey); err != nil {
			return err
		} else if ok {
			outLedger = existing
			b, err := getBalanceTx(ctx, tx, workspaceID, walletID)
			if err != nil {
				return err
			}
			outBal = b
			return nil
		}

		// Ensure sufficient funds using the projection row and lock it.
		b, err := getBalanceForUpdate(ctx, tx, workspaceID, walletID)
		if err != nil {
			return err
		}
		if b.Currency != req.Currency {
			return ErrInvalidArgument
		}
		if b.BalanceMinor < req.AmountMinor {
			return ErrInsufficientFunds
		}

		entry := WalletLedger{
			ID:             ledgerID,
			WorkspaceID:    workspaceID,
			WalletID:       walletID,
			Type:           LedgerEntryTypeDebit,
			AmountMinor:    -req.AmountMinor,
			Currency:       req.Currency,
			ExternalRef:    req.ExternalRef,
			IdempotencyKey: req.IdempotencyKey,
			Metadata:       req.Metadata,
			CreatedAt:      now,
		}
		if err := insertLedger(ctx, tx, entry); err != nil {
			return err
		}

		out, err := applyBalanceDelta(ctx, tx, workspaceID, walletID, req.Currency, -req.AmountMinor, now)
		if err != nil {
			return err
		}
		outLedger = entry
		outBal = out
		return nil
	})

	return outLedger, outBal, err
}

func (s *Service) AdminManualCredit(ctx context.Context, workspaceID, walletID, adminUserID, adminRole string, req AdminCreditRequest) (AdminWalletAction, WalletLedger, Balance, error) {
	if adminUserID == "" || adminRole == "" {
		return AdminWalletAction{}, WalletLedger{}, Balance{}, ErrInvalidArgument
	}
	if req.Reason == "" {
		return AdminWalletAction{}, WalletLedger{}, Balance{}, ErrInvalidArgument
	}
	if err := validateMoneyReq(workspaceID, walletID, req.AmountMinor, req.Currency, req.IdempotencyKey); err != nil {
		return AdminWalletAction{}, WalletLedger{}, Balance{}, err
	}
	if req.AmountMinor <= 0 {
		return AdminWalletAction{}, WalletLedger{}, Balance{}, ErrInvalidArgument
	}

	now := s.clock().UTC()
	actionID := uuid.NewString()
	ledgerID := uuid.NewString()

	var outAction AdminWalletAction
	var outLedger WalletLedger
	var outBal Balance

	err := utils.WithTx(ctx, s.db, &sql.TxOptions{}, func(ctx context.Context, tx *sql.Tx) error {
		w, err := lockWallet(ctx, tx, workspaceID, walletID)
		if err != nil {
			return err
		}
		if w.Currency != req.Currency {
			return ErrInvalidArgument
		}

		// Idempotency based on ledger idempotency key; admin action will be derived.
		if existing, ok, err := findLedgerByIdempotency(ctx, tx, workspaceID, walletID, req.IdempotencyKey); err != nil {
			return err
		} else if ok {
			outLedger = existing
			// Best-effort: look up admin action by related_ledger_id.
			act, ok, err := findAdminActionByLedger(ctx, tx, workspaceID, walletID, existing.ID)
			if err != nil {
				return err
			}
			if ok {
				outAction = act
			}
			b, err := getBalanceTx(ctx, tx, workspaceID, walletID)
			if err != nil {
				return err
			}
			outBal = b
			return nil
		}

		entry := WalletLedger{
			ID:             ledgerID,
			WorkspaceID:    workspaceID,
			WalletID:       walletID,
			Type:           LedgerEntryTypeCredit,
			AmountMinor:    req.AmountMinor,
			Currency:       req.Currency,
			ExternalRef:    "admin_manual_credit",
			IdempotencyKey: req.IdempotencyKey,
			Metadata:       req.Metadata,
			CreatedAt:      now,
		}
		if err := insertLedger(ctx, tx, entry); err != nil {
			return err
		}

		b, err := applyBalanceDelta(ctx, tx, workspaceID, walletID, req.Currency, req.AmountMinor, now)
		if err != nil {
			return err
		}

		action := AdminWalletAction{
			ID:              actionID,
			WorkspaceID:     workspaceID,
			WalletID:        walletID,
			AdminUserID:     adminUserID,
			AdminRole:       adminRole,
			Action:          AdminWalletActionTypeAdjustBalance,
			Reason:          req.Reason,
			AmountMinor:     req.AmountMinor,
			Currency:        req.Currency,
			RelatedLedgerID: entry.ID,
			Metadata:        req.Metadata,
			CreatedAt:       now,
		}
		if err := insertAdminAction(ctx, tx, action); err != nil {
			return err
		}

		outAction = action
		outLedger = entry
		outBal = b
		return nil
	})

	return outAction, outLedger, outBal, err
}

func validateMoneyReq(workspaceID, walletID string, amountMinor int64, currency, idempotencyKey string) error {
	if workspaceID == "" || walletID == "" {
		return ErrInvalidArgument
	}
	if currency == "" {
		return ErrInvalidArgument
	}
	if idempotencyKey == "" {
		return ErrInvalidArgument
	}
	if amountMinor == 0 {
		return ErrInvalidArgument
	}
	return nil
}
