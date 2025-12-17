package pricing

import "testing"

func TestBillableSeconds(t *testing.T) {
	// 60s increment, 0 min
	if got := billableSeconds(1, 0, 60); got != 60 {
		t.Fatalf("expected 60, got %d", got)
	}
	if got := billableSeconds(60, 0, 60); got != 60 {
		t.Fatalf("expected 60, got %d", got)
	}
	if got := billableSeconds(61, 0, 60); got != 120 {
		t.Fatalf("expected 120, got %d", got)
	}

	// min billable seconds
	if got := billableSeconds(5, 30, 60); got != 60 {
		t.Fatalf("expected 60, got %d", got)
	}
}

func TestBillableMinutesFromSeconds(t *testing.T) {
	if got := billableMinutesFromSeconds(1); got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
	if got := billableMinutesFromSeconds(60); got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
	if got := billableMinutesFromSeconds(61); got != 2 {
		t.Fatalf("expected 2, got %d", got)
	}
}
