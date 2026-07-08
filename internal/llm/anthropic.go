// internal/llm/anthropic.go
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const anthropicBaseURL = "https://api.anthropic.com/v1"

// anthropicMessage is the Anthropic API message format.
type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicRequest is the Anthropic API request body.
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
	System    string             `json:"system,omitempty"`
	Stream    bool               `json:"stream"`
}

// anthropicContent is a content block in the response.
type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// anthropicResponse is the sync response.
type anthropicResponse struct {
	Content    []anthropicContent `json:"content"`
	StopReason string             `json:"stop_reason"`
}

// anthropicStreamEvent is a streaming SSE event.
type anthropicStreamEvent struct {
	Type       string             `json:"type"`
	Index      int                `json:"index,omitempty"`
	ContentBlock *anthropicContent `json:"content_block,omitempty"`
	Delta      *struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
	} `json:"delta,omitempty"`
}

// AnthropicProvider implements Provider using the Anthropic Messages API.
type AnthropicProvider struct {
	apiKey  string
	model   string
	client  *http.Client
}

// NewAnthropic creates a new Anthropic provider.
func NewAnthropic(apiKey, model string) *AnthropicProvider {
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	return &AnthropicProvider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{},
	}
}

// ChatSync sends a synchronous chat request to Anthropic.
func (p *AnthropicProvider) ChatSync(ctx context.Context, messages []Message) (*Response, error) {
	req := anthropicRequest{
		Model:     p.model,
		MaxTokens: 4096,
		Stream:    false,
	}

	// Convert messages: extract system prompt, convert rest
	for _, m := range messages {
		if m.Role == "system" {
			req.System = m.Content
			continue
		}
		role := m.Role
		if role == "assistant" {
			role = "assistant"
		}
		req.Messages = append(req.Messages, anthropicMessage{Role: role, Content: m.Content})
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", anthropicBaseURL+"/messages", bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var ar anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Extract text and tool calls from content blocks
	var text string
	var toolCalls []ToolCall
	for _, block := range ar.Content {
		switch block.Type {
		case "text":
			text += block.Text
		case "tool_use":
			toolCalls = append(toolCalls, ToolCall{
				ID: block.ID,
				Function: FunctionCall{
					Name:      block.Name,
					Arguments: string(block.Input),
				},
			})
		}
	}

	stopReason := ar.StopReason
	if stopReason == "end_turn" {
		stopReason = "stop"
	}
	if stopReason == "tool_use" {
		stopReason = "tool_calls"
	}

	return &Response{
		Text:       text,
		ToolCalls:  toolCalls,
		StopReason: stopReason,
	}, nil
}

// Chat sends a streaming chat request to Anthropic.
func (p *AnthropicProvider) Chat(ctx context.Context, messages []Message) (<-chan StreamEvent, error) {
	req := anthropicRequest{
		Model:     p.model,
		MaxTokens: 4096,
		Stream:    true,
	}

	for _, m := range messages {
		if m.Role == "system" {
			req.System = m.Content
			continue
		}
		req.Messages = append(req.Messages, anthropicMessage{Role: m.Role, Content: m.Content})
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", anthropicBaseURL+"/messages", bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("api call: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("api error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	ch := make(chan StreamEvent, 100)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}

			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			var ev anthropicStreamEvent
			if err := json.Unmarshal([]byte(data), &ev); err != nil {
				continue
			}

			switch ev.Type {
			case "content_block_delta":
				if ev.Delta != nil && ev.Delta.Type == "text_delta" {
					select {
					case <-ctx.Done():
						return
					case ch <- StreamEvent{Type: "token", Token: ev.Delta.Text}:
					}
				}
			case "message_stop":
				select {
				case <-ctx.Done():
					return
				case ch <- StreamEvent{Type: "done"}:
				}
			}
		}

		select {
		case <-ctx.Done():
			return
		case ch <- StreamEvent{Type: "done"}:
		}
	}()

	return ch, nil
}