package routing

// Decision is the provider-agnostic output of the routing engine.
//
// It must contain *only* information required for the provider adapter boundary
// (e.g., Twilio TwiML builder) to execute the decision.
//
// No provider identity and no provider-specific fields belong here.

type Decision struct {
	WorkspaceID string `json:"workspace_id"`
	CampaignID  string `json:"campaign_id,omitempty"`

	Action    Action `json:"action"`
	ConnectTo string `json:"connect_to,omitempty"`

	// Reason is optional and intended for internal logs/metrics.
	Reason string `json:"reason,omitempty"`
}

type Action string

const (
	ActionReject  Action = "reject"
	ActionConnect Action = "connect"
	ActionHangup  Action = "hangup"
)
