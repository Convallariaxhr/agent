// internal/recovery/recovery.go
package recovery

import (
	"context"
	"errors"
	"fmt"

	"github.com/Convallariaxhr/convallaria/internal/llm"
)

var ErrMaxRetriesExceeded = errors.New("max retries exceeded")

// Config configures error recovery behavior.
type Config struct {
	MaxRetries     int
	MaxCorrections int
}

// Handler is a function that processes an LLM response and returns a result.
type Handler func(resp *llm.Response) (string, error)

// WithRecovery wraps a handler with retry and correction logic.
func WithRecovery(ctx context.Context, provider llm.Provider, cfg Config, handler Handler) (string, error) {
	var messages []llm.Message

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		resp, err := provider.ChatSync(ctx, messages)
		if err != nil {
			if attempt >= cfg.MaxRetries {
				return "", fmt.Errorf("llm error after %d retries: %w", attempt, err)
			}
			messages = append(messages, llm.Message{
				Role:    "system",
				Content: fmt.Sprintf("Error: %v. Please try again with a valid response.", err),
			})
			continue
		}

		result, err := handler(resp)
		if err == nil {
			return result, nil
		}

		if attempt >= cfg.MaxRetries {
			return "", ErrMaxRetriesExceeded
		}

		// Inject error feedback and retry
		messages = append(messages,
			llm.Message{Role: "assistant", Content: fmt.Sprintf("%v", resp)},
			llm.Message{Role: "system", Content: fmt.Sprintf("Error processing your response: %v. Please correct and retry.", err)},
		)
	}

	return "", ErrMaxRetriesExceeded
}