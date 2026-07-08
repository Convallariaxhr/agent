// internal/parser/parser_test.go
package parser

import (
	"testing"

	"github.com/Convallariaxhr/convallaria/internal/llm"
)

func TestParse_PureText_NoToolCalls(t *testing.T) {
	resp := &llm.Response{
		Text:       "Hello, how can I help you?",
		StopReason: "stop",
	}
	actions := Parse(resp)
	if len(actions) != 0 {
		t.Errorf("expected 0 actions, got %d", len(actions))
	}
	if !actions.IsStop() {
		t.Error("expected IsStop() to be true for pure text")
	}
}

func TestParse_SingleToolCall(t *testing.T) {
	resp := &llm.Response{
		ToolCalls: []llm.ToolCall{
			{
				ID: "call_1",
				Function: llm.FunctionCall{
					Name:      "file_read",
					Arguments: `{"path":"main.go"}`,
				},
			},
		},
		StopReason: "tool_calls",
	}
	actions := Parse(resp)
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].Tool != "file_read" {
		t.Errorf("expected tool 'file_read', got %q", actions[0].Tool)
	}
	if actions[0].Params["path"] != "main.go" {
		t.Errorf("expected path 'main.go', got %v", actions[0].Params["path"])
	}
}

func TestParse_MultipleToolCalls(t *testing.T) {
	resp := &llm.Response{
		ToolCalls: []llm.ToolCall{
			{ID: "call_1", Function: llm.FunctionCall{Name: "file_read", Arguments: `{"path":"a.go"}`}},
			{ID: "call_2", Function: llm.FunctionCall{Name: "file_write", Arguments: `{"path":"b.go","content":"x"}`}},
		},
		StopReason: "tool_calls",
	}
	actions := Parse(resp)
	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(actions))
	}
}

func TestParse_MalformedJSON(t *testing.T) {
	resp := &llm.Response{
		ToolCalls: []llm.ToolCall{
			{ID: "call_1", Function: llm.FunctionCall{Name: "shell_run", Arguments: `{not json}`}},
		},
		StopReason: "tool_calls",
	}
	actions := Parse(resp)
	if len(actions) != 1 {
		t.Fatalf("expected 1 action even with malformed JSON, got %d", len(actions))
	}
	if actions[0].ParseError == nil {
		t.Error("expected ParseError for malformed JSON")
	}
}