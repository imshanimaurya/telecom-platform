package audit

import (
	"context"
	"sync"
)

// MemoryRepo is a simple in-memory append-only repository useful for tests.
// It is not intended for production use.

type MemoryRepo struct {
	mu     sync.Mutex
	events []Event
}

func NewMemoryRepo() *MemoryRepo { return &MemoryRepo{} }

func (r *MemoryRepo) Append(ctx context.Context, e Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
	return nil
}

func (r *MemoryRepo) Events() []Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Event, len(r.events))
	copy(out, r.events)
	return out
}
