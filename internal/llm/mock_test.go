// internal/llm/mock_test.go
package llm

import (
	"context"
	"testing"
)

func TestMockProvider_ChatSync_ReturnsPresetResponse(t *testing.T) {
	mock := NewMockProvider()
	mock.AddResponse(MockTextResponse("Hello, world!"))

	resp, err := mock.ChatSync(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Text != "Hello, world!" {
		t.Errorf("expected 'Hello, world!', got %q", resp.Text)
	}
	if resp.StopReason != "stop" {
		t.Errorf("expected stop reason 'stop', got %q", resp.StopReason)
	}
}

func TestMockProvider_ChatSync_ReturnsToolCall(t *testing.T) {
	mock := NewMockProvider()
	mock.AddResponse(MockToolCallResponse("call_1", "file_write", `{"path":"main.go","content":"package main"}`))

	resp, err := mock.ChatSync(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Function.Name != "file_write" {
		t.Errorf("expected tool 'file_write', got %q", resp.ToolCalls[0].Function.Name)
	}
}

func TestMockProvider_Chat_StreamsTokens(t *testing.T) {
	mock := NewMockProvider()
	mock.AddResponse(MockTextResponse("Hi!"))

	ch, err := mock.Chat(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var tokens []string
	for ev := range ch {
		if ev.Type == "token" {
			tokens = append(tokens, ev.Token)
		}
	}
	result := ""
	for _, tok := range tokens {
		result += tok
	}
	if result != "Hi!" {
		t.Errorf("expected 'Hi!', got %q", result)
	}
}

func TestMockProvider_CallCount(t *testing.T) {
	mock := NewMockProvider()
	mock.AddResponse(MockTextResponse("a"))
	mock.AddResponse(MockTextResponse("b"))

	mock.ChatSync(context.Background(), nil)
	if mock.CallCount() != 1 {
		t.Errorf("expected call count 1, got %d", mock.CallCount())
	}
	mock.ChatSync(context.Background(), nil)
	if mock.CallCount() != 2 {
		t.Errorf("expected call count 2, got %d", mock.CallCount())
	}
}