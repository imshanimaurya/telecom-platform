package utils

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// PostgresPoolConfig controls database/sql pool behavior.
// Keep it config-driven; defaults should be safe and conservative.
type PostgresPoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	PingTimeout     time.Duration
}

func (c PostgresPoolConfig) withDefaults() PostgresPoolConfig {
	out := c
	if out.MaxOpenConns <= 0 {
		out.MaxOpenConns = 25
	}
	if out.MaxIdleConns <= 0 {
		out.MaxIdleConns = 25
	}
	if out.ConnMaxLifetime <= 0 {
		out.ConnMaxLifetime = 30 * time.Minute
	}
	if out.ConnMaxIdleTime <= 0 {
		out.ConnMaxIdleTime = 5 * time.Minute
	}
	if out.PingTimeout <= 0 {
		out.PingTimeout = 5 * time.Second
	}
	return out
}

// OpenPostgres opens a Postgres connection using database/sql.
// driverName should typically be "pgx" (pgx stdlib).
// dsn must not be logged; it contains secrets.
func OpenPostgres(ctx context.Context, driverName, dsn string, pool PostgresPoolConfig) (*sql.DB, error) {
	pool = pool.withDefaults()

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(pool.MaxOpenConns)
	db.SetMaxIdleConns(pool.MaxIdleConns)
	db.SetConnMaxLifetime(pool.ConnMaxLifetime)
	db.SetConnMaxIdleTime(pool.ConnMaxIdleTime)

	if err := HealthCheck(ctx, db, pool.PingTimeout); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// HealthCheck pings the DB with a timeout.
func HealthCheck(ctx context.Context, db *sql.DB, timeout time.Duration) error {
	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		return fmt.Errorf("db ping failed: %w", err)
	}
	return nil
}

// TxFunc is the unit of work executed inside a transaction.
type TxFunc func(ctx context.Context, tx *sql.Tx) error

// WithTx runs fn inside a transaction.
// - If fn returns error: tx is rolled back and the error is returned.
// - If fn panics: tx is rolled back and the panic is re-thrown.
// - If commit fails: commit error is returned.
func WithTx(ctx context.Context, db *sql.DB, opts *sql.TxOptions, fn TxFunc) (err error) {
	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	err = fn(ctx, tx)
	return err
}
