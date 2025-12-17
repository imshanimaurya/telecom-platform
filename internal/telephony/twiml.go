package telephony

import (
	"bytes"
	"encoding/xml"
	"errors"
	"strings"
)

// TwiML is a minimal Twilio Markup Language response builder.
// It intentionally avoids any provider SDK dependency.
//
// Only include primitives we need at the adapter boundary.

type twimlResponse struct {
	XMLName xml.Name     `xml:"Response"`
	Verbs   []any        `xml:",any"`
}

type twimlReject struct {
	XMLName xml.Name `xml:"Reject"`
	Reason  string   `xml:"reason,attr,omitempty"`
}

type twimlHangup struct {
	XMLName xml.Name `xml:"Hangup"`
}

type twimlDial struct {
	XMLName xml.Name `xml:"Dial"`
	Number  string   `xml:"Number,omitempty"`
	Sip     *twimlSip `xml:"Sip,omitempty"`
}

type twimlSip struct {
	URI string `xml:",chardata"`
}

// RenderTwiML maps an InboundCallResult to TwiML.
func RenderTwiML(res InboundCallResult) (string, error) {
	var r twimlResponse

	switch res.Action {
	case InboundCallActionReject:
		r.Verbs = append(r.Verbs, twimlReject{Reason: "busy"})
	case InboundCallActionHangup:
		r.Verbs = append(r.Verbs, twimlHangup{})
	case InboundCallActionConnect:
		if strings.TrimSpace(res.ConnectTo) == "" {
			return "", errors.New("telephony: connect_to required for connect action")
		}
		d := twimlDial{}
		// Prefer SIP if it looks like sip:... otherwise treat as a PSTN number.
		if strings.HasPrefix(strings.ToLower(res.ConnectTo), "sip:") {
			d.Sip = &twimlSip{URI: res.ConnectTo}
		} else {
			d.Number = res.ConnectTo
		}
		r.Verbs = append(r.Verbs, d)
	default:
		return "", errors.New("telephony: unknown inbound action")
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(r); err != nil {
		return "", err
	}
	if err := enc.Flush(); err != nil {
		return "", err
	}
	return buf.String(), nil
}
