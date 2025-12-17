package wallet

import (
	"context"
	"database/sql"
	"testing"
)

// These are true unit tests for wallet.Service input validation behavior.
//
// The money operations (Credit/Debit/AdminManualCredit) are implemented with
// Postgres-specific SQL (notably SELECT ... FOR UPDATE). That means end-to-end
// behavior tests (balance changes, insufficient funds, ledger/admin action inserts)
// are best covered via integration tests against Postgres.
//
// What we *can* safely unit-test without a DB:
// - request validation (
//   workspace_id / wallet_id presence,
//   currency presence,
//   idempotency key presence,
//   amount > 0,
//   adminUserID/adminRole/reason presence for admin credit
// )
//
// See also: TestValidateMoneyReq in service_test.go.

func TestWalletService_Credit_RejectsInvalidArgs(t *testing.T) {
	svc := NewService((*sql.DB)(nil))

	_, _, err := svc.Credit(context.Background(), "", "w", CreditRequest{AmountMinor: 100, Currency: "USD", IdempotencyKey: "k"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if err != ErrInvalidArgument {
		t.Fatalf("expected ErrInvalidArgument, got %v", err)
	}

	_, _, err = svc.Credit(context.Background(), "ws", "w", CreditRequest{AmountMinor: 0, Currency: "USD", IdempotencyKey: "k"})
	if err != ErrInvalidArgument {
		t.Fatalf("expected ErrInvalidArgument, got %v", err)
	}

	_, _, err = svc.Credit(context.Background(), "ws", "w", CreditRequest{AmountMinor: 100, Currency: "", IdempotencyKey: "k"})
	if err != ErrInvalidArgument {
		t.Fatalf("expected ErrInvalidArgument, got %v", err)
	}

	_, _, err = svc.Credit(context.Background(), "ws", "w", CreditRequest{AmountMinor: 100, Currency: "USD", IdempotencyKey: ""})
	if err != ErrInvalidArgument {
		t.Fatalf("expected ErrInvalidArgument, got %v", err)
	}
}

func TestWalletService_Debit_RejectsInvalidArgs(t *testing.T) {
	svc := NewService((*sql.DB)(nil))

	_, _, err := svc.Debit(context.Background(), "", "w", DebitRequest{AmountMinor: 100, Currency: "USD", IdempotencyKey: "k"})
	if err != ErrInvalidArgument {
		t.Fatalf("expected ErrInvalidArgument, got %v", err)
	}

	_, _, err = svc.Debit(context.Background(), "ws", "w", DebitRequest{AmountMinor: -1, Currency: "USD", IdempotencyKey: "k"})
	if err != ErrInvalidArgument {
		t.Fatalf("expected ErrInvalidArgument, got %v", err)
	}
}

func TestWalletService_AdminManualCredit_RejectsInvalidArgs(t *testing.T) {
	svc := NewService((*sql.DB)(nil))

	_, _, _, err := svc.AdminManualCredit(context.Background(), "ws", "w", "", "owner", AdminCreditRequest{
		AmountMinor:    100,
		Currency:       "USD",
		Reason:         "refund",
		IdempotencyKey: "k",
	})
	if err != ErrInvalidArgument {
		t.Fatalf("expected ErrInvalidArgument (missing admin user), got %v", err)
	}

	_, _, _, err = svc.AdminManualCredit(context.Background(), "ws", "w", "admin", "", AdminCreditRequest{
		AmountMinor:    100,
		Currency:       "USD",
		Reason:         "refund",
		IdempotencyKey: "k",
	})
	if err != ErrInvalidArgument {
		t.Fatalf("expected ErrInvalidArgument (missing admin role), got %v", err)
	}

	_, _, _, err = svc.AdminManualCredit(context.Background(), "ws", "w", "admin", "owner", AdminCreditRequest{
		AmountMinor:    100,
		Currency:       "USD",
		Reason:         "",
		IdempotencyKey: "k",
	})
	if err != ErrInvalidArgument {
		t.Fatalf("expected ErrInvalidArgument (missing reason), got %v", err)
	}

	_, _, _, err = svc.AdminManualCredit(context.Background(), "ws", "w", "admin", "owner", AdminCreditRequest{
		AmountMinor:    0,
		Currency:       "USD",
		Reason:         "refund",
		IdempotencyKey: "k",
	})
	if err != ErrInvalidArgument {
		t.Fatalf("expected ErrInvalidArgument (amount <=0), got %v", err)
	}
}
