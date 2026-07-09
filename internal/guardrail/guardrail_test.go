// internal/guardrail/guardrail_test.go
package guardrail

import (
	"testing"
)

func TestGuardrail_BlocksDangerousCommand(t *testing.T) {
	g := New(Config{
		DangerousCommands: true,
		FileScope:         true,
		GitDangerousOps:   true,
		Workspace:         "/tmp/test",
	})

	reason := g.Check("shell_run", map[string]any{"command": "rm -rf /"})
	if reason == nil {
		t.Fatal("expected block for 'rm -rf /'")
	}
	if reason.Level != "dangerous_command" {
		t.Errorf("expected level 'dangerous_command', got %q", reason.Level)
	}
}

func TestGuardrail_BlocksFileOutsideWorkspace(t *testing.T) {
	g := New(Config{
		DangerousCommands: true,
		FileScope:         true,
		GitDangerousOps:   true,
		Workspace:         "/tmp/test",
	})

	reason := g.Check("file_write", map[string]any{"path": "../../etc/passwd", "content": "x"})
	if reason == nil {
		t.Fatal("expected block for writing outside workspace")
	}
	if reason.Level != "file_scope" {
		t.Errorf("expected level 'file_scope', got %q", reason.Level)
	}
}

func TestGuardrail_AllowsFileInsideWorkspace(t *testing.T) {
	g := New(Config{
		DangerousCommands: true,
		FileScope:         true,
		GitDangerousOps:   true,
		Workspace:         "/tmp/test",
	})

	reason := g.Check("file_write", map[string]any{"path": "/tmp/test/main.go", "content": "x"})
	if reason != nil {
		t.Errorf("expected no block for workspace file, got %v", reason)
	}
}

func TestGuardrail_BlocksGitForcePush(t *testing.T) {
	g := New(Config{
		DangerousCommands: true,
		FileScope:         true,
		GitDangerousOps:   true,
		Workspace:         "/tmp/test",
	})

	reason := g.Check("git", map[string]any{"operation": "push", "force": true})
	if reason == nil {
		t.Fatal("expected block for git push --force")
	}
	if reason.Level != "git_dangerous" {
		t.Errorf("expected level 'git_dangerous', got %q", reason.Level)
	}
}

func TestGuardrail_AllowsSafeCommand(t *testing.T) {
	g := New(Config{
		DangerousCommands: true,
		FileScope:         true,
		GitDangerousOps:   true,
		Workspace:         "/tmp/test",
	})

	reason := g.Check("shell_run", map[string]any{"command": "go build ./..."})
	if reason != nil {
		t.Errorf("expected no block for 'go build', got %v", reason)
	}
}

func TestGuardrail_DisabledGuardrails(t *testing.T) {
	g := New(Config{
		DangerousCommands: false,
		FileScope:         false,
		GitDangerousOps:   false,
		Workspace:         "/tmp/test",
	})

	reason := g.Check("shell_run", map[string]any{"command": "rm -rf /"})
	if reason != nil {
		t.Errorf("expected no block when guardrails disabled, got %v", reason)
	}
}