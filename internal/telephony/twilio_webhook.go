package telephony

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// TwilioInboundForm captures the subset of voice webhook fields we care about.
// Twilio sends application/x-www-form-urlencoded by default.
// Ref: https://www.twilio.com/docs/voice/twiml
//
// Keep it minimal and provider-adapter-only.
// Business logic (routing decisions) is not made here.

type TwilioInboundForm struct {
	CallSid      string
	AccountSid   string
	From         string
	To           string
	Direction    string
	CallStatus   string
	ApiVersion   string
	Timestamp    string
	CallerName   string
	FromCity     string
	FromState    string
	FromZip      string
	FromCountry  string
	ToCity       string
	ToState      string
	ToZip        string
	ToCountry    string
	ForwardedFrom string
}

func ParseTwilioInboundCall(r *http.Request) (TwilioInboundForm, error) {
	if err := r.ParseForm(); err != nil {
		return TwilioInboundForm{}, err
	}
	f := TwilioInboundForm{
		CallSid:       r.PostFormValue("CallSid"),
		AccountSid:    r.PostFormValue("AccountSid"),
		From:          normalizePhone(r.PostFormValue("From")),
		To:            normalizePhone(r.PostFormValue("To")),
		Direction:     r.PostFormValue("Direction"),
		CallStatus:    r.PostFormValue("CallStatus"),
		ApiVersion:    r.PostFormValue("ApiVersion"),
		Timestamp:     r.PostFormValue("Timestamp"),
		CallerName:    r.PostFormValue("CallerName"),
		FromCity:      r.PostFormValue("FromCity"),
		FromState:     r.PostFormValue("FromState"),
		FromZip:       r.PostFormValue("FromZip"),
		FromCountry:   r.PostFormValue("FromCountry"),
		ToCity:        r.PostFormValue("ToCity"),
		ToState:       r.PostFormValue("ToState"),
		ToZip:         r.PostFormValue("ToZip"),
		ToCountry:     r.PostFormValue("ToCountry"),
		ForwardedFrom: normalizePhone(r.PostFormValue("ForwardedFrom")),
	}
	return f, nil
}

func normalizePhone(s string) string {
	s = strings.TrimSpace(s)
	// Twilio sometimes sends "anonymous" or empty; keep as-is.
	return s
}

func (f TwilioInboundForm) ToInboundCallRequest(workspaceID string, occurredAt time.Time) InboundCallRequest {
	raw, _ := json.Marshal(f)
	return InboundCallRequest{
		WorkspaceID:     workspaceID,
		ProviderCallID:  f.CallSid,
		From:           f.From,
		To:             f.To,
		OccurredAt:     occurredAt,
		RawPayload:     string(raw),
	}
}
