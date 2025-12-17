package audit

import (
	"context"
	"testing"
)

func TestService_AppendRequiresWorkspaceAndType(t *testing.T) {
	repo := NewMemoryRepo()
	svc := NewService(repo)

	if err := svc.Append(context.Background(), Event{Type: EventTypeAdminAction}); err == nil {
		t.Fatalf("expected error")
	}
	if err := svc.Append(context.Background(), Event{WorkspaceID: "w"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestService_AppendsImmutableEvents(t *testing.T) {
	repo := NewMemoryRepo()
	svc := NewService(repo)

	if err := svc.LogAdminAction(context.Background(), "w", "u", "super_admin", "1.2.3.4", "did something", "wallet1", "{}"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	evs := repo.Events()
	if len(evs) != 1 {
		t.Fatalf("expected 1 event")
	}
	if evs[0].IPAddress != "1.2.3.4" {
		t.Fatalf("expected ip captured")
	}
	if evs[0].Type != EventTypeAdminAction {
		t.Fatalf("expected admin_action")
	}
}
