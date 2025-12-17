package auth

import (
	"context"
	"errors"
)

type ctxKey int

const (
	ctxUserID ctxKey = iota
	ctxWorkspaceID
	ctxRole
)

func WithIdentity(ctx context.Context, userID, workspaceID, role string) context.Context {
	ctx = context.WithValue(ctx, ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxWorkspaceID, workspaceID)
	ctx = context.WithValue(ctx, ctxRole, role)
	return ctx
}

func UserID(ctx context.Context) (string, error) {
	v := ctx.Value(ctxUserID)
	if s, ok := v.(string); ok && s != "" {
		return s, nil
	}
	return "", errors.New("user_id not in context")
}

func WorkspaceID(ctx context.Context) (string, error) {
	v := ctx.Value(ctxWorkspaceID)
	if s, ok := v.(string); ok && s != "" {
		return s, nil
	}
	return "", errors.New("workspace_id not in context")
}

func Role(ctx context.Context) (string, error) {
	v := ctx.Value(ctxRole)
	if s, ok := v.(string); ok && s != "" {
		return s, nil
	}
	return "", errors.New("role not in context")
}
