// internal/context/manager_test.go
package context

import (
	"testing"

	"github.com/Convallariaxhr/convallaria/internal/llm"
)

func TestManager_EstimateTokens(t *testing.T) {
	mgr := New(Config{MaxTokens: 64000, Threshold: 0.8})

	msgs := []llm.Message{
		{Role: "system", Content: "You are a coding agent."},
		{Role: "user", Content: "Write a hello world program."},
	}
	tokens := mgr.EstimateTokens(msgs)
	if tokens <= 0 {
		t.Errorf("expected positive token count, got %d", tokens)
	}
	// Rough estimate: ~20 tokens for these messages
	if tokens < 10 || tokens > 100 {
		t.Errorf("expected ~10-100 tokens, got %d", tokens)
	}
}

func TestManager_NeedsCompression(t *testing.T) {
	mgr := New(Config{MaxTokens: 64000, Threshold: 0.8})

	// Small messages should not need compression
	msgs := []llm.Message{
		{Role: "user", Content: "Hello"},
	}
	if mgr.NeedsCompression(msgs) {
		t.Error("expected no compression needed for small messages")
	}
}

func TestManager_Compress(t *testing.T) {
	mgr := New(Config{MaxTokens: 64000, Threshold: 0.8})

	// Create many messages to simulate long conversation
	msgs := []llm.Message{
		{Role: "system", Content: "System prompt"},
		{Role: "user", Content: "Message 1"},
		{Role: "assistant", Content: "Reply 1"},
		{Role: "user", Content: "Message 2"},
		{Role: "assistant", Content: "Reply 2"},
		{Role: "user", Content: "Message 3"},
	}

	compressed := mgr.Compress(msgs, 3) // Keep last 3
	// Should keep system prompt + last 3 messages
	if len(compressed) < 4 || len(compressed) > 6 {
		t.Errorf("expected 4-6 messages after compression, got %d", len(compressed))
	}
	// First message should still be system prompt
	if compressed[0].Role != "system" {
		t.Error("expected system prompt to be preserved")
	}
}