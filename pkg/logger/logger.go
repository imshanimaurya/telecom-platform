package logger

import (
	"context"
	"log/slog"
	"os"
	"time"
)

// New returns a production-friendly structured logger.
// No business logic should depend on logging implementation details.
func New(appEnv string) *slog.Logger {
	level := slog.LevelInfo
	if appEnv == "local" || appEnv == "dev" {
		level = slog.LevelDebug
	}

	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(h)
}

type ctxKey struct{}

// With stores a logger in context.
func With(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// From gets a logger from context, falling back to slog.Default().
func From(ctx context.Context) *slog.Logger {
	if v := ctx.Value(ctxKey{}); v != nil {
		if l, ok := v.(*slog.Logger); ok && l != nil {
			return l
		}
	}
	return slog.Default()
}

// ShutdownFlush is a placeholder for future log flushing (if a buffered logger is used).
func ShutdownFlush(_ context.Context, _ time.Duration) error { return nil }
