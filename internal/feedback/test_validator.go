// internal/feedback/test_validator.go
package feedback

import (
	"context"
	"encoding/json"
	"os/exec"
)

type TestValidator struct{}

func (v *TestValidator) Validate(ctx context.Context, workspace string) *Feedback {
	fb := &Feedback{Stage: "test", Status: "passed"}

	cmd := exec.CommandContext(ctx, "go", "test", "-json", "./...")
	cmd.Dir = workspace
	output, err := cmd.CombinedOutput()

	if err == nil {
		fb.Summary = "Tests passed"
		return fb
	}

	fb.Status = "failed"
	fb.Summary = "Tests failed"

	// Parse go test -json output
	type testEvent struct {
		Action  string `json:"Action"`
		Test    string `json:"Test"`
		Package string `json:"Package"`
		Output  string `json:"Output"`
	}

	for _, line := range bytesToLines(output) {
		var ev testEvent
		if json.Unmarshal([]byte(line), &ev) == nil {
			if ev.Action == "fail" && ev.Test != "" {
				fb.Errors = append(fb.Errors, FeedbackError{
					File:    ev.Package,
					Message: "Test failed: " + ev.Test,
				})
			}
		}
	}
	return fb
}

func bytesToLines(data []byte) []string {
	var lines []string
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, string(data[start:i]))
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, string(data[start:]))
	}
	return lines
}