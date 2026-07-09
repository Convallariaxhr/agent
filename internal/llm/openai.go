// internal/llm/openai.go
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

const defaultOpenAIBaseURL = "https://api.openai.com/v1"

// OpenAIProvider implements Provider using the OpenAI API.
// Shares the same OpenAI-compatible protocol as DeepSeekProvider.
type OpenAIProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
	tools   []ToolDef
}

// NewOpenAI creates a new OpenAI provider.
func NewOpenAI(apiKey, model string) *OpenAIProvider {
	if model == "" {
		model = "gpt-4o"
	}
	return &OpenAIProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: defaultOpenAIBaseURL,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

// SetTools configures the tools available to the LLM.
func (p *OpenAIProvider) SetTools(tools []ToolDef) {
	p.tools = tools
}

// ChatSync sends a synchronous chat request and returns the full response.
func (p *OpenAIProvider) ChatSync(ctx context.Context, messages []Message) (*Response, error) {
	body := chatRequest{
		Model:    p.model,
		Messages: messages,
		Stream:   false,
		Tools:    toOpenAITools(p.tools),
	}

	reqBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
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
func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message) (<-chan StreamEvent, error) {
	body := chatRequest{
		Model:    p.model,
		Messages: messages,
		Stream:   true,
	}

	reqBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(req)
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

		select {
		case <-ctx.Done():
			return
		case ch <- StreamEvent{Type: "done"}:
		}
	}()

	return ch, nil
}