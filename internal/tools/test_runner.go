// internal/tools/test_runner.go
package tools

import (
	"context"
	"os/exec"
)

type TestRunner struct{}

func (t *TestRunner) Name() string        { return "test_run" }
func (t *TestRunner) Description() string { return "Run tests in the project" }

func (t *TestRunner) Execute(ctx context.Context, params map[string]any) (*Result, error) {
	testPath, _ := params["path"].(string)
	args := []string{"test", "-json"}
	if testPath != "" {
		args = append(args, testPath)
	} else {
		args = append(args, "./...")
	}

	cmd := exec.CommandContext(ctx, "go", args...)
	// Set working directory if provided
	if ws, ok := params["workspace"].(string); ok && ws != "" {
		cmd.Dir = ws
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &Result{
			Output:  string(output),
			Success: false,
			Error:   "tests failed",
		}, nil
	}
	return &Result{Output: string(output), Success: true}, nil
}