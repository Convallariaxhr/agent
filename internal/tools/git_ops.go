// internal/tools/git_ops.go
package tools

import (
	"context"
	"os/exec"
	"strings"
)

type GitOps struct{}

func (g *GitOps) Name() string        { return "git" }
func (g *GitOps) Description() string { return "Execute git operations (status, commit, branch, diff)" }
func (g *GitOps) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"operation": map[string]any{"type": "string", "description": "Git operation: status, commit, diff, branch"},
			"message":   map[string]any{"type": "string", "description": "Commit message (required for commit)"},
		},
		"required": []string{"operation"},
	}
}

func (g *GitOps) Execute(ctx context.Context, params map[string]any) (*Result, error) {
	operation, _ := params["operation"].(string)
	args := []string{operation}

	switch operation {
	case "status":
		args = append(args, "--short")
	case "commit":
		message, _ := params["message"].(string)
		args = append(args, "-m", message)
	case "diff":
		// no extra args
	case "branch":
		// list branches
	default:
		return &Result{Success: false, Error: "unsupported git operation: " + operation}, nil
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	// Set working directory if provided
	if ws, ok := params["workspace"].(string); ok && ws != "" {
		cmd.Dir = ws
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &Result{
			Output:  string(output),
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	return &Result{Output: strings.TrimSpace(string(output)), Success: true}, nil
}