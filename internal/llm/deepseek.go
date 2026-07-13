// internal/llm/deepseek.go
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
	"time"
)

const defaultDeepSeekBaseURL = "https://api.deepseek.com/v1"

// DeepSeekProvider implements Provider using the DeepSeek API (OpenAI-compatible).
type DeepSeekProvider struct {
	apiKey      string
	model       string
	baseURL     string
	client      *http.Client
	tools       []ToolDef
	forceTool   bool
}

// NewDeepSeek creates a new DeepSeek provider.
func NewDeepSeek(apiKey, model string) *DeepSeekProvider {
	return NewDeepSeekWithURL(apiKey, model, defaultDeepSeekBaseURL)
}

// NewDeepSeekWithURL creates a new DeepSeek provider with a custom base URL.
func NewDeepSeekWithURL(apiKey, model, baseURL string) *DeepSeekProvider {
	if model == "" {
		model = "deepseek-chat"
	}
	return &DeepSeekProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

// SetTools configures the tools available to the LLM.
func (d *DeepSeekProvider) SetTools(tools []ToolDef) {
	d.tools = tools
}

// ForceToolUse forces the next ChatSync call to require at least one tool call.
func (d *DeepSeekProvider) ForceToolUse() {
	d.forceTool = true
}

// chatRequest is the OpenAI-compatible request body.
type chatRequest struct {
	Model      string       `json:"model"`
	Messages   []Message    `json:"messages"`
	Stream     bool         `json:"stream"`
	Tools      []openAITool `json:"tools,omitempty"`
	ToolChoice string       `json:"tool_choice,omitempty"`
}

// chatResponse is the OpenAI-compatible sync response.
type chatResponse struct {
	Choices []struct {
		Message struct {
			Role      string     `json:"role"`
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// streamChunk is a single SSE chunk from the streaming API.
type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// ChatSync sends a synchronous chat request and returns the full response.
func (d *DeepSeekProvider) ChatSync(ctx context.Context, messages []Message) (*Response, error) {
	toolChoice := ""
	if d.forceTool && len(d.tools) > 0 {
		toolChoice = "required"
		d.forceTool = false
	}
	body := chatRequest{
		Model:      d.model,
		Messages:   messages,
		Stream:     false,
		Tools:      toOpenAITools(d.tools),
		ToolChoice: toolChoice,
	}

	reqBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", d.baseURL+"/chat/completions", bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.apiKey)

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var cr chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(cr.Choices) == 0 {
		return &Response{Text: "", StopReason: "stop"}, nil
	}

	choice := cr.Choices[0]
	return &Response{
		Text:       choice.Message.Content,
		ToolCalls:  choice.Message.ToolCalls,
		StopReason: choice.FinishReason,
	}, nil
}

// Chat sends a streaming chat request and returns a channel of events.
func (d *DeepSeekProvider) Chat(ctx context.Context, messages []Message) (<-chan StreamEvent, error) {
	toolChoice := ""
	if d.forceTool && len(d.tools) > 0 {
		toolChoice = "required"
		d.forceTool = false
	}
	body := chatRequest{
		Model:      d.model,
		Messages:   messages,
		Stream:     true,
		Tools:      toOpenAITools(d.tools),
		ToolChoice: toolChoice,
	}

	reqBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", d.baseURL+"/chat/completions", bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.apiKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := d.client.Do(req)
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
			if line == "" || line == "data: [DONE]" {
				continue
			}
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			var chunk streamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				select {
				case <-ctx.Done():
					return
				case ch <- StreamEvent{Type: "error", Error: err}:
				}
				continue
			}

			if len(chunk.Choices) == 0 {
				continue
			}

			delta := chunk.Choices[0].Delta
			if delta.Content != "" {
				select {
				case <-ctx.Done():
					return
				case ch <- StreamEvent{Type: "token", Token: delta.Content}:
				}
			}

			for _, tc := range delta.ToolCalls {
				select {
				case <-ctx.Done():
					return
				case ch <- StreamEvent{Type: "tool_call", ToolCall: &tc}:
				}
			}

			if chunk.Choices[0].FinishReason != "" {
				select {
				case <-ctx.Done():
					return
				case ch <- StreamEvent{Type: "done"}:
				}
			}
		}

		// Ensure done event is sent even if stream ends without explicit finish_reason
		select {
		case <-ctx.Done():
			return
		case ch <- StreamEvent{Type: "done"}:
		}
	}()

	return ch, nil
}