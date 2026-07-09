// internal/llm/provider.go
package llm

import "context"

// Message represents a single chat message.
type Message struct {
	Role    string `json:"role"` // system, user, assistant, tool
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
	// ToolCalls is populated for assistant messages that request tool execution.
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	// ToolCallID links tool result messages to the originating call.
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// ToolCall represents a function call requested by the LLM.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall contains the name and arguments of a tool invocation.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON-encoded
}

// StreamEvent is emitted during streaming responses.
type StreamEvent struct {
	Type     string    `json:"type"` // "token", "tool_call", "done", "error"
	Token    string    `json:"token,omitempty"`
	ToolCall *ToolCall `json:"tool_call,omitempty"`
	Error    error     `json:"-"`
}

// Response is the final complete response from the LLM.
type Response struct {
	Text       string     `json:"text"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	StopReason string     `json:"stop_reason"` // "stop", "tool_calls", "max_tokens"
}

// ToolDef describes a tool available to the LLM.
type ToolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"` // JSON Schema for the tool's arguments
}

// openAITool is the OpenAI-compatible wrapper format for function calling.
type openAITool struct {
	Type     string  `json:"type"`
	Function ToolDef `json:"function"`
}

// toOpenAITools converts ToolDefs to the OpenAI function-calling format.
func toOpenAITools(tools []ToolDef) []openAITool {
	result := make([]openAITool, len(tools))
	for i, t := range tools {
		result[i] = openAITool{Type: "function", Function: t}
	}
	return result
}

// Provider defines the interface for LLM interactions.
type Provider interface {
	// Chat sends messages and returns a channel of streaming events.
	Chat(ctx context.Context, messages []Message) (<-chan StreamEvent, error)
	// ChatSync sends messages and returns a complete response (no streaming).
	ChatSync(ctx context.Context, messages []Message) (*Response, error)
	// SetTools configures the tools available to the LLM.
	SetTools(tools []ToolDef)
}