// internal/feedback/feedback.go
package feedback

import (
	"context"
	"encoding/json"

	"github.com/Convallariaxhr/convallaria/internal/llm"
)

// Feedback is the structured result of running validators.
type Feedback struct {
	Stage   string          `json:"stage"`   // build, vet, test
	Status  string          `json:"status"`  // passed, failed
	Errors  []FeedbackError `json:"errors"`
	Summary string          `json:"summary"`
}

// FeedbackError describes a single validation error.
type FeedbackError struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
}

// Validator checks code quality and returns structured feedback.
type Validator interface {
	Validate(ctx context.Context, workspace string) *Feedback
}

// LoopResult is the aggregated result of running all validators.
type LoopResult struct {
	Status string
	Stage  string
	Errors []FeedbackError
}

// Loop runs validators in order and returns the first failure.
type Loop struct {
	validators []Validator
}

func NewLoop() *Loop {
	return &Loop{
		validators: []Validator{
			&BuildValidator{},
			&VetValidator{},
			&TestValidator{},
		},
	}
}

// Run executes all validators in sequence. Returns the first failure.
func (l *Loop) Run(ctx context.Context, workspace string) *LoopResult {
	for _, v := range l.validators {
		fb := v.Validate(ctx, workspace)
		if fb.Status == "failed" {
			return &LoopResult{
				Status: "failed",
				Stage:  fb.Stage,
				Errors: fb.Errors,
			}
		}
	}
	return &LoopResult{Status: "passed"}
}

// ToMessage converts feedback into an LLM message for context injection.
func (fb *Feedback) ToMessage() llm.Message {
	data, _ := json.Marshal(fb)
	return llm.Message{
		Role:    "tool",
		Name:    "feedback",
		Content: string(data),
	}
}