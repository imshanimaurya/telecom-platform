package telephony

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseTwilioInboundCall(t *testing.T) {
	body := strings.NewReader("CallSid=CA123&From=%2B15551234567&To=%2B15557654321")
	r := httptest.NewRequest(http.MethodPost, "/webhooks/twilio/voice", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	form, err := ParseTwilioInboundCall(r)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if form.CallSid != "CA123" {
		t.Fatalf("expected CallSid")
	}
	if form.From != "+15551234567" || form.To != "+15557654321" {
		t.Fatalf("unexpected from/to: %q %q", form.From, form.To)
	}

	req := form.ToInboundCallRequest("w1", time.Unix(1700000000, 0).UTC())
	if req.WorkspaceID != "w1" {
		t.Fatalf("expected workspace_id")
	}
	if req.ProviderCallID != "CA123" {
		t.Fatalf("expected provider call id")
	}
	if req.From == "" || req.To == "" {
		t.Fatalf("expected from/to")
	}
}
