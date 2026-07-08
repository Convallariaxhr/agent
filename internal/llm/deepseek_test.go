// internal/llm/deepseek_test.go
package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeepSeek_ChatSync_ReturnsText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Error("expected POST")
		}
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Error("expected Authorization header")
		}

		resp := chatResponse{
			Choices: []struct {
				Message struct {
					Role      string     `json:"role"`
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{FinishReason: "stop"},
			},
		}
		resp.Choices[0].Message.Content = "Hello from DeepSeek!"
		resp.Choices[0].Message.Role = "assistant"
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := &DeepSeekProvider{
		apiKey:  "sk-test",
		model:   "deepseek-chat",
		baseURL: server.URL,
		client:  server.Client(),
	}

	result, err := provider.ChatSync(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "Hello from DeepSeek!" {
		t.Errorf("expected 'Hello from DeepSeek!', got %q", result.Text)
	}
	if result.StopReason != "stop" {
		t.Errorf("expected stop reason 'stop', got %q", result.StopReason)
	}
}

func TestDeepSeek_ChatSync_ReturnsToolCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := chatResponse{
			Choices: []struct {
				Message struct {
					Role      string     `json:"role"`
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{FinishReason: "tool_calls"},
			},
		}
		resp.Choices[0].Message.Role = "assistant"
		resp.Choices[0].Message.ToolCalls = []ToolCall{
			{ID: "call_1", Function: FunctionCall{Name: "file_read", Arguments: `{"path":"main.go"}`}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := &DeepSeekProvider{
		apiKey:  "sk-test",
		model:   "deepseek-chat",
		baseURL: server.URL,
		client:  server.Client(),
	}

	result, err := provider.ChatSync(context.Background(), []Message{
		{Role: "user", Content: "Read main.go"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(result.ToolCalls))
	}
	if result.ToolCalls[0].Function.Name != "file_read" {
		t.Errorf("expected tool 'file_read', got %q", result.ToolCalls[0].Function.Name)
	}
}

func TestDeepSeek_Chat_StreamsTokens(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected Flusher")
		}
		chunks := []string{
			`data: {"choices":[{"delta":{"content":"Hello"},"finish_reason":""}]}` + "\n\n",
			`data: {"choices":[{"delta":{"content":" world"},"finish_reason":""}]}` + "\n\n",
			`data: {"choices":[{"delta":{"content":"!"},"finish_reason":"stop"}]}` + "\n\n",
			`data: [DONE]` + "\n\n",
		}
		for _, chunk := range chunks {
			w.Write([]byte(chunk))
			flusher.Flush()
		}
	}))
	defer server.Close()

	provider := &DeepSeekProvider{
		apiKey:  "sk-test",
		model:   "deepseek-chat",
		baseURL: server.URL,
		client:  server.Client(),
	}

	ch, err := provider.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	})
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
	if result != "Hello world!" {
		t.Errorf("expected 'Hello world!', got %q", result)
	}
}

func TestDeepSeek_Chat_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"Invalid API key"}}`))
	}))
	defer server.Close()

	provider := &DeepSeekProvider{
		apiKey:  "sk-bad",
		model:   "deepseek-chat",
		baseURL: server.URL,
		client:  server.Client(),
	}

	_, err := provider.ChatSync(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}