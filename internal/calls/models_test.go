package calls

import "testing"

func TestCallStatusValuesAreNonEmpty(t *testing.T) {
	statuses := []CallStatus{
		CallStatusQueued,
		CallStatusRinging,
		CallStatusInProgress,
		CallStatusCompleted,
		CallStatusFailed,
		CallStatusNoAnswer,
		CallStatusBusy,
		CallStatusCanceled,
	}
	for _, s := range statuses {
		if s == "" {
			t.Fatalf("expected non-empty status")
		}
	}
}

func TestCall_FieldsCompile(t *testing.T) {
	_ = Call{}
}
