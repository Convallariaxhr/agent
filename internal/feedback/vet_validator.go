// internal/feedback/vet_validator.go
package feedback

import (
	"context"
	"os/exec"
	"strings"
)

type VetValidator struct{}

func (v *VetValidator) Validate(ctx context.Context, workspace string) *Feedback {
	fb := &Feedback{Stage: "vet", Status: "passed"}

	cmd := exec.CommandContext(ctx, "go", "vet", "./...")
	cmd.Dir = workspace
	output, err := cmd.CombinedOutput()

	if err == nil {
		fb.Summary = "Vet passed"
		return fb
	}

	fb.Status = "failed"
	fb.Summary = "Vet found issues"
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line != "" {
			fb.Errors = append(fb.Errors, FeedbackError{
				Message: line,
			})
		}
	}
	return fb
}