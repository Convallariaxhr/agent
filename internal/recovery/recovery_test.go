// internal/recovery/recovery_test.go
package recovery

import (
	"context"
	"errors"
	"testing"

	"github.com/Convallariaxhr/convallaria/internal/llm"
)

func TestRecovery_RetryOnParseError(t *testing.T) {
	mock := llm.NewMockProvider()
	// First response: malformed
	mock.AddResponse(&llm.Response{
		ToolCalls: []llm.ToolCall{
			{ID: "call_1", Function: llm.FunctionCall{Name: "shell_run", Arguments: `{bad json}`}},
		},
		StopReason: "tool_calls",
	})
	// Second response: corrected
	mock.AddResponse(llm.MockTextResponse("OK, fixed."))

	cfg := Config{MaxRetries: 3, MaxCorrections: 2}
	result, err := WithRecovery(context.Background(), mock, cfg, func(resp *llm.Response) (string, error) {
		if len(resp.ToolCalls) > 0 && resp.ToolCalls[0].Function.Arguments == `{bad json}` {
			return "", errors.New("parse error")
		}
		return resp.Text, nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "OK, fixed." {
		t.Errorf("expected 'OK, fixed.', got %q", result)
	}
	if mock.CallCount() < 2 {
		t.Error("expected at least 2 LLM calls (retry)")
	}
}

func TestRecovery_DegradeOnMaxRetries(t *testing.T) {
	mock := llm.NewMockProvider()
	for i := 0; i < 5; i++ {
		mock.AddResponse(&llm.Response{
			ToolCalls: []llm.ToolCall{
				{ID: "call_1", Function: llm.FunctionCall{Name: "shell_run", Arguments: `{bad}`}},
			},
			StopReason: "tool_calls",
		})
	}

	cfg := Config{MaxRetries: 2, MaxCorrections: 1}
	_, err := WithRecovery(context.Background(), mock, cfg, func(resp *llm.Response) (string, error) {
		return "", errors.New("parse error")
	})

	if err != ErrMaxRetriesExceeded {
		t.Errorf("expected ErrMaxRetriesExceeded, got %v", err)
	}
}