// internal/llm/mock_test.go
package llm

import (
	"context"
	"errors"
	"testing"
	"time"
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

func TestMockProvider_ChatSync_ContextCancelled(t *testing.T) {
	mock := NewMockProvider()
	mock.AddResponse(MockTextResponse("hello"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	resp, err := mock.ChatSync(ctx, nil)
	if err == nil {
		t.Errorf("expected error for cancelled context, got response: %+v", resp)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestMockProvider_Chat_ContextCancelled(t *testing.T) {
	mock := NewMockProvider()
	// Use many events so the goroutine is still running when we cancel
	events := make([]StreamEvent, 1000)
	for i := range events {
		events[i] = StreamEvent{Type: "token", Token: "x"}
	}
	mock.AddEvents(events)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := mock.Chat(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Cancel after receiving the first event
	var count int
	for ev := range ch {
		count++
		if count == 1 {
			cancel()
		}
		_ = ev
	}
	// We should not receive all 1000 events — the loop should break early
	if count >= 1000 {
		t.Errorf("expected fewer than 1000 events after cancellation, got %d", count)
	}
}

func TestMockProvider_Chat_AddEvents(t *testing.T) {
	mock := NewMockProvider()
	mock.AddEvents([]StreamEvent{
		{Type: "token", Token: "A"},
		{Type: "token", Token: "B"},
		{Type: "done"},
	})

	ch, err := mock.Chat(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var events []StreamEvent
	for ev := range ch {
		events = append(events, ev)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].Type != "token" || events[0].Token != "A" {
		t.Errorf("unexpected first event: %+v", events[0])
	}
	if events[1].Type != "token" || events[1].Token != "B" {
		t.Errorf("unexpected second event: %+v", events[1])
	}
	if events[2].Type != "done" {
		t.Errorf("expected done event, got %+v", events[2])
	}
}

func TestMockProvider_Chat_ToolCallStreaming(t *testing.T) {
	mock := NewMockProvider()
	mock.AddResponse(MockToolCallResponse("call_1", "file_write", `{"path":"main.go"}`))

	ch, err := mock.Chat(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var toolCalls []ToolCall
	for ev := range ch {
		if ev.Type == "tool_call" && ev.ToolCall != nil {
			toolCalls = append(toolCalls, *ev.ToolCall)
		}
	}
	if len(toolCalls) != 1 {
		t.Fatalf("expected 1 tool call event, got %d", len(toolCalls))
	}
	if toolCalls[0].Function.Name != "file_write" {
		t.Errorf("expected tool 'file_write', got %q", toolCalls[0].Function.Name)
	}
}

func TestMockProvider_ChatSync_EmptyQueue(t *testing.T) {
	mock := NewMockProvider()

	resp, err := mock.ChatSync(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Text != "" {
		t.Errorf("expected empty text, got %q", resp.Text)
	}
	if resp.StopReason != "stop" {
		t.Errorf("expected stop reason 'stop', got %q", resp.StopReason)
	}
}

func TestMockProvider_Chat_EmptyQueue(t *testing.T) {
	mock := NewMockProvider()

	ch, err := mock.Chat(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var events []StreamEvent
	for ev := range ch {
		events = append(events, ev)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event (done), got %d", len(events))
	}
	if events[0].Type != "done" {
		t.Errorf("expected done event, got %+v", events[0])
	}
}

func TestMockProvider_ChatSync_ReturnsError(t *testing.T) {
	mock := NewMockProvider()
	mock.Err = errors.New("simulated failure")

	resp, err := mock.ChatSync(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error, got response: %+v", resp)
	}
	if err.Error() != "simulated failure" {
		t.Errorf("expected 'simulated failure', got %q", err.Error())
	}
}

func TestMockProvider_Chat_ReturnsError(t *testing.T) {
	mock := NewMockProvider()
	mock.Err = errors.New("simulated failure")

	ch, err := mock.Chat(context.Background(), nil)
	if err == nil {
		// Drain the channel if one was returned
		for range ch {
		}
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != "simulated failure" {
		t.Errorf("expected 'simulated failure', got %q", err.Error())
	}
}

func TestMockProvider_ConcurrentAccess(t *testing.T) {
	// Verify no data races when Chat and ChatSync are called concurrently.
	mock := NewMockProvider()
	for i := 0; i < 10; i++ {
		mock.AddResponse(MockTextResponse("x"))
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 5; i++ {
			ch, err := mock.Chat(context.Background(), nil)
			if err != nil {
				t.Errorf("Chat error: %v", err)
				return
			}
			for range ch {
			}
		}
	}()

	for i := 0; i < 5; i++ {
		_, err := mock.ChatSync(context.Background(), nil)
		if err != nil {
			t.Errorf("ChatSync error: %v", err)
		}
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for concurrent Chat to finish")
	}
}