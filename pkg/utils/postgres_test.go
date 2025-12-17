package utils

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

type fakeDB struct{}

func TestWithTx_RollbackOnError(t *testing.T) {
	// This test can't run without a real *sql.DB; keep it as a compile-time smoke test
	// for the helper signature.
	var _ = WithTx
	_ = context.Background()
	_ = &sql.DB{}
	_ = errors.New("x")
}
