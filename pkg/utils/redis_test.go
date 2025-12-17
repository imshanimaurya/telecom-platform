package utils

import "testing"

func TestConcurrencyScriptsCompile(t *testing.T) {
	// Compile-time smoke test: scripts should be initialized.
	if concurrencyAcquireScript == nil || concurrencyReleaseScript == nil {
		t.Fatalf("expected scripts to be initialized")
	}
}
