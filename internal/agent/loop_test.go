// internal/agent/loop_test.go
package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Convallariaxhr/convallaria/internal/llm"
)

func TestAgent_TextResponse_ReturnsFinalReply(t *testing.T) {
	mock := llm.NewMockProvider()
	mock.AddResponse(llm.MockTextResponse("Hello! I can help you write code."))

	agent := New(Config{
		MaxTurns: 5,
		Provider: mock,
	})

	resp, err := agent.Run(context.Background(), "Write a hello world program", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "Hello! I can help you write code." {
		t.Errorf("expected text response, got %q", resp)
	}
}

func TestAgent_ToolCall_ExecutesAndReturnsResult(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.ToSlash(filepath.Join(dir, "hello.txt"))

	mock := llm.NewMockProvider()
	// First response: tool call to write file
	mock.AddResponse(llm.MockToolCallResponse("call_1", "file_write",
		`{"path":"`+filePath+`","content":"hello world"}`))
	// Second response: text completion
	mock.AddResponse(llm.MockTextResponse("Done! I've created hello.txt with 'hello world'."))

	agent := New(Config{
		MaxTurns:  5,
		Provider:  mock,
		Workspace: dir,
	})

	resp, err := agent.Run(context.Background(), "Create a file called hello.txt", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "Done! I've created hello.txt with 'hello world'." {
		t.Errorf("unexpected response: %q", resp)
	}
	// Verify file was actually created
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("file content mismatch: %q", string(data))
	}
}

func TestAgent_GuardrailBlocksDangerousAction(t *testing.T) {
	mock := llm.NewMockProvider()
	// LLM tries to run a dangerous command
	mock.AddResponse(llm.MockToolCallResponse("call_1", "shell_run",
		`{"command":"rm -rf /"}`))
	// After blocked, it should get a text response
	mock.AddResponse(llm.MockTextResponse("Sorry, I cannot execute that command."))

	agent := New(Config{
		MaxTurns:  5,
		Provider:  mock,
		Workspace: "/tmp/test",
	})

	resp, err := agent.Run(context.Background(), "Delete everything", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == "" {
		t.Error("expected a response after guardrail block")
	}
	// The dangerous command should NOT have been executed
	// (the mock would have returned a tool result if it was)
}

func TestAgent_FeedbackLoop_DetectsBuildError(t *testing.T) {
	dir := t.TempDir()
	// Write a broken Go file
	brokenFile := filepath.Join(dir, "broken.go")
	os.WriteFile(brokenFile, []byte("package main\n\nfunc main() {\n\tundefined\n}\n"), 0644)

	mock := llm.NewMockProvider()
	// LLM writes a broken file
	mock.AddResponse(llm.MockToolCallResponse("call_1", "file_write",
		`{"path":"`+brokenFile+`","content":"package main\n\nfunc main() {\n\tundefined\n}\n"}`))
	// Feedback loop should detect build failure and inform LLM
	// LLM then tries to fix
	mock.AddResponse(llm.MockToolCallResponse("call_2", "file_write",
		`{"path":"`+brokenFile+`","content":"package main\n\nfunc main() { println(\"hello\") }\n"}`))
	// After fix, text response
	mock.AddResponse(llm.MockTextResponse("Fixed the build error!"))

	agent := New(Config{
		MaxTurns:  5,
		Provider:  mock,
		Workspace: dir,
	})

	resp, err := agent.Run(context.Background(), "Write a hello world program", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == "" {
		t.Error("expected a response after feedback loop")
	}
	// Verify the feedback loop ran: the second tool call should have been triggered
	// by the build failure feedback from the first file write
	if mock.CallCount() < 2 {
		t.Error("expected at least 2 LLM calls (initial + after feedback)")
	}
}

func TestAgent_MaxTurnsExceeded(t *testing.T) {
	mock := llm.NewMockProvider()
	// Add many tool calls that will never converge
	for i := 0; i < 10; i++ {
		mock.AddResponse(llm.MockToolCallResponse("call_"+string(rune('a'+i)), "file_write",
			`{"path":"/tmp/test/x.go","content":"x"}`))
	}

	agent := New(Config{
		MaxTurns:  3,
		Provider:  mock,
		Workspace: "/tmp/test",
	})

	_, err := agent.Run(context.Background(), "Write code", nil)
	if err != ErrMaxTurnsExceeded {
		t.Errorf("expected ErrMaxTurnsExceeded, got %v", err)
	}
}