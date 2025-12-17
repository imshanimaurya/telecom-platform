package telephony

import "testing"

func TestRenderTwiMLReject(t *testing.T) {
	xml, err := RenderTwiML(InboundCallResult{WorkspaceID: "w", Action: InboundCallActionReject})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if xml == "" {
		t.Fatalf("expected xml")
	}
	if want := "<Reject"; !contains(xml, want) {
		t.Fatalf("expected %q in xml: %s", want, xml)
	}
}

func TestRenderTwiMLConnectRequiresTarget(t *testing.T) {
	_, err := RenderTwiML(InboundCallResult{WorkspaceID: "w", Action: InboundCallActionConnect})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (func() bool { return indexOf(s, sub) >= 0 })())
}

func indexOf(s, sub string) int {
	// tiny helper to avoid importing strings in this small test file
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
