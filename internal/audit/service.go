package audit

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Repository is the persistence contract for audit events.
//
// It MUST be append-only.
// No Update/Delete methods are provided by design.

type Repository interface {
	Append(ctx context.Context, e Event) error
}

// Service logs internal audit information.
//
// IMPORTANT:
// - Audit is internal-only. Do not expose these records to tenant users by default.
// - Callers should treat audit logging as best-effort.

type Service struct {
	repo  Repository
	clock func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, clock: time.Now}
}

var ErrInvalidEvent = errors.New("audit: invalid event")

func (s *Service) Append(ctx context.Context, e Event) error {
	if s.repo == nil {
		return errors.New("audit: repository not configured")
	}
	if e.WorkspaceID == "" {
		return ErrInvalidEvent
	}
	if e.Type == "" {
		return ErrInvalidEvent
	}

	now := s.clock().UTC()
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	return s.repo.Append(ctx, e)
}

// LogAdminAction records an admin action (including hidden roles).
func (s *Service) LogAdminAction(ctx context.Context, workspaceID, actorUserID, actorRole, ip, message, walletID string, metadata string) error {
	return s.Append(ctx, Event{
		WorkspaceID:  workspaceID,
		Type:         EventTypeAdminAction,
		ActorUserID:  actorUserID,
		ActorRole:    actorRole,
		IPAddress:    ip,
		WalletID:     walletID,
		Message:      message,
		Metadata:     metadata,
	})
}

// LogOverride records an internal override usage.
func (s *Service) LogOverride(ctx context.Context, workspaceID, actorUserID, actorRole, ip, campaignID, callID, overrideID, connectTo, metadata string) error {
	return s.Append(ctx, Event{
		WorkspaceID: workspaceID,
		Type:        EventTypeOverride,
		ActorUserID: actorUserID,
		ActorRole:   actorRole,
		IPAddress:   ip,
		CampaignID:  campaignID,
		CallID:      callID,
		OverrideID:  overrideID,
		Message:     "override applied",
		Metadata:    metadata,
	})
}
