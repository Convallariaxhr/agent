// internal/tools/shell_runner.go
package tools

import (
	"context"
	"os/exec"
	"runtime"
	"time"
)

type ShellRunner struct {
	Timeout time.Duration
}

func (s *ShellRunner) Name() string        { return "shell_run" }
func (s *ShellRunner) Description() string { return "Execute a shell command" }

func (s *ShellRunner) Execute(ctx context.Context, params map[string]any) (*Result, error) {
	command, _ := params["command"].(string)
	timeout := s.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/c", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}
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
	return &Result{Output: string(output), Success: true}, nil
}