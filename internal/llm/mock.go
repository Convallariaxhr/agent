// internal/llm/mock.go
package llm

import (
	"context"
)

// MockProvider returns preset responses for deterministic testing.
type MockProvider struct {
	// Responses is a queue of responses to return.
	Responses []*Response
	// Events is a queue of event sequences for streaming.
	Events    [][]StreamEvent
	callCount int
}

// NewMockProvider creates a new MockProvider.
func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

// AddResponse appends a sync response to the queue.
func (m *MockProvider) AddResponse(r *Response) {
	m.Responses = append(m.Responses, r)
}

// AddEvents appends a streaming event sequence to the queue.
func (m *MockProvider) AddEvents(events []StreamEvent) {
	m.Events = append(m.Events, events)
}

// Chat sends messages and returns a channel of streaming events.
func (m *MockProvider) Chat(ctx context.Context, messages []Message) (<-chan StreamEvent, error) {
	ch := make(chan StreamEvent, 100)
	go func() {
		defer close(ch)
		if m.callCount < len(m.Events) {
			for _, ev := range m.Events[m.callCount] {
				select {
				case <-ctx.Done():
					return
				case ch <- ev:
				}
			}
		} else if m.callCount < len(m.Responses) {
			resp := m.Responses[m.callCount]
			for _, r := range resp.Text {
				select {
				case <-ctx.Done():
					return
				case ch <- StreamEvent{Type: "token", Token: string(r)}:
				}
			}
			for _, tc := range resp.ToolCalls {
				select {
				case <-ctx.Done():
					return
				case ch <- StreamEvent{Type: "tool_call", ToolCall: &tc}:
				}
			}
			select {
			case <-ctx.Done():
				return
			case ch <- StreamEvent{Type: "done"}:
			}
		} else {
			select {
			case <-ctx.Done():
				return
			case ch <- StreamEvent{Type: "done"}:
			}
		}
		m.callCount++
	}()
	return ch, nil
}

// ChatSync sends messages and returns a complete response (no streaming).
func (m *MockProvider) ChatSync(ctx context.Context, messages []Message) (*Response, error) {
	if m.callCount < len(m.Responses) {
		resp := m.Responses[m.callCount]
		m.callCount++
		return resp, nil
	}
	m.callCount++
		return &Response{Text: "", StopReason: "stop"}, nil
}

// CallCount returns the number of times Chat/ChatSync has been called.
func (m *MockProvider) CallCount() int {
	return m.callCount
}

// MockToolCallResponse creates a tool call response quickly.
func MockToolCallResponse(id, name, args string) *Response {
	return &Response{
		ToolCalls: []ToolCall{
			{ID: id, Function: FunctionCall{Name: name, Arguments: args}},
		},
		StopReason: "tool_calls",
	}
}

// MockTextResponse creates a text response quickly.
func MockTextResponse(text string) *Response {
	return &Response{Text: text, StopReason: "stop"}
}