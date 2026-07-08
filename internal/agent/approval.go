// internal/agent/approval.go
package agent

import "context"

// ApprovalRequest is sent when a dangerous action needs user approval.
type ApprovalRequest struct {
	Tool    string
	Command string
	Reason  string
}

// ApprovalResponse is the user's decision.
type ApprovalResponse struct {
	Allowed    bool
	AlwaysAllow bool
}

// ApprovalHandler is called when an action needs HITL approval.
// Returns the user's decision. The handler should block until the user responds.
type ApprovalHandler func(ctx context.Context, req ApprovalRequest) (ApprovalResponse, error)