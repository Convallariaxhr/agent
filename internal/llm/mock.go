// internal/llm/mock.go
package llm

import (
	"context"
	"sync"
)

// MockProvider returns preset responses for deterministic testing.
// It is safe for concurrent use.
type MockProvider struct {
	mu sync.Mutex

	// Responses is a queue of responses to return.
	Responses []*Response
	// Events is a queue of event sequences for streaming.
	Events [][]StreamEvent
	// Err, if non-nil, is returned by Chat and ChatSync instead of a response.
	Err       error
	callCount int
}

// NewMockProvider creates a new MockProvider.
func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

// SetTools is a no-op for MockProvider (responses are pre-configured).
func (m *MockProvider) SetTools(tools []ToolDef) {}

// AddResponse appends a sync response to the queue.
func (m *MockProvider) AddResponse(r *Response) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Responses = append(m.Responses, r)
}

// AddEvents appends a streaming event sequence to the queue.
func (m *MockProvider) AddEvents(events []StreamEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Events = append(m.Events, events)
}

// Chat sends messages and returns a channel of streaming events.
func (m *MockProvider) Chat(ctx context.Context, messages []Message) (<-chan StreamEvent, error) {
	m.mu.Lock()
	if m.Err != nil {
		err := m.Err
		m.mu.Unlock()
		return nil, err
	}

	count := m.callCount
	var events []StreamEvent
	var resp *Response
	hasEvents := count < len(m.Events)
	hasResponses := count < len(m.Responses)

	if hasEvents {
		events = m.Events[count]
	} else if hasResponses {
		resp = m.Responses[count]
	}
	m.callCount++
	m.mu.Unlock()

	ch := make(chan StreamEvent, 100)
	go func() {
		defer close(ch)
		if hasEvents {
			for _, ev := range events {
				select {
				case <-ctx.Done():
					return
				case ch <- ev:
				}
			}
		} else if hasResponses {
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
	}()
	return ch, nil
}

// ChatSync sends messages and returns a complete response (no streaming).
func (m *MockProvider) ChatSync(ctx context.Context, messages []Message) (*Response, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Err != nil {
		return nil, m.Err
	}

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
	m.mu.Lock()
	defer m.mu.Unlock()
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