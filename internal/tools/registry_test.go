// internal/tools/registry_test.go
package tools

import (
	"context"
	"testing"
)

func TestRegistry_RegisterAndExecute(t *testing.T) {
	reg := NewRegistry()
	reg.Register("echo", &EchoTool{})

	result, err := reg.Execute(context.Background(), "echo", map[string]any{"message": "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Output != "hello" {
		t.Errorf("expected 'hello', got %q", result.Output)
	}
}

func TestRegistry_UnknownTool(t *testing.T) {
	reg := NewRegistry()
	_, err := reg.Execute(context.Background(), "nonexistent", nil)
	if err != ErrUnknownTool {
		t.Errorf("expected ErrUnknownTool, got %v", err)
	}
}

func TestRegistry_ListTools(t *testing.T) {
	reg := NewRegistry()
	reg.Register("tool_a", &EchoTool{})
	reg.Register("tool_b", &EchoTool{})

	tools := reg.List()
	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}
}

// EchoTool is a simple test tool.
type EchoTool struct{}

func (e *EchoTool) Execute(ctx context.Context, params map[string]any) (*Result, error) {
	msg, _ := params["message"].(string)
	return &Result{Output: msg, Success: true}, nil
}

func (e *EchoTool) Name() string        { return "echo" }
func (e *EchoTool) Description() string { return "Echoes the message parameter" }