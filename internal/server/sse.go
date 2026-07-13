// internal/server/sse.go
package server

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// SSEWriter writes Server-Sent Events to an http.ResponseWriter.
type SSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

// NewSSEWriter creates a new SSE writer and sends the initial headers.
// If ctx is provided, a keepalive heartbeat is started that stops when ctx is cancelled.
func NewSSEWriter(w http.ResponseWriter, ctx context.Context) *SSEWriter {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		panic("http.ResponseWriter does not implement http.Flusher")
	}

	// Send initial comment to establish connection
	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	s := &SSEWriter{w: w, flusher: flusher}

	// Keepalive heartbeat every 15 seconds, stopped by context cancellation
	if ctx != nil {
		go func() {
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					fmt.Fprintf(s.w, ": heartbeat\n\n")
					s.flusher.Flush()
				}
			}
		}()
	}

	return s
}

// WriteEvent writes an SSE event with the given type and data.
func (s *SSEWriter) WriteEvent(eventType, data string) {
	fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", eventType, data)
	s.flusher.Flush()
}
