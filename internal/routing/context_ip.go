package routing

import (
	"context"
)

// clientIPKey is an unexported context key for passing client IP through internal layers.
//
// Provider HTTP handlers (Gin) should resolve the real client IP and attach it to
// the request context using WithClientIP.

type clientIPKey struct{}

func WithClientIP(ctx context.Context, ip string) context.Context {
	if ip == "" {
		return ctx
	}
	return context.WithValue(ctx, clientIPKey{}, ip)
}

func ClientIPFromContext(ctx context.Context) string {
	v := ctx.Value(clientIPKey{})
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
