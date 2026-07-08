// internal/feedback/build_validator.go
package feedback

import (
	"context"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type BuildValidator struct{}

func (v *BuildValidator) Validate(ctx context.Context, workspace string) *Feedback {
	fb := &Feedback{Stage: "build", Status: "passed"}

	cmd := exec.CommandContext(ctx, "go", "build", "./...")
	cmd.Dir = workspace
	output, err := cmd.CombinedOutput()

	if err == nil {
		fb.Summary = "Build passed"
		return fb
	}

	fb.Status = "failed"
	fb.Summary = "Build failed"

	// Parse Go build errors: file:line:col: message
	errStr := string(output)
	re := regexp.MustCompile(`(.+?):(\d+):(\d+):\s*(.+)`)
	for _, line := range strings.Split(errStr, "\n") {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 5 {
			lineNo, _ := strconv.Atoi(matches[2])
			colNo, _ := strconv.Atoi(matches[3])
			fb.Errors = append(fb.Errors, FeedbackError{
				File:    matches[1],
				Line:    lineNo,
				Column:  colNo,
				Message: strings.TrimSpace(matches[4]),
			})
		}
	}
	return fb
}