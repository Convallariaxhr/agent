// internal/server/sse.go
package server

import (
	"fmt"
	"net/http"
	"time"
)

// SSEWriter writes Server-Sent Events to an http.ResponseWriter.
type SSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
	done    chan struct{}
}

// NewSSEWriter creates a new SSE writer, sends initial headers, and starts a keepalive heartbeat.
func NewSSEWriter(w http.ResponseWriter) *SSEWriter {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		panic("http.ResponseWriter does not implement http.Flusher")
	}

	s := &SSEWriter{w: w, flusher: flusher, done: make(chan struct{})}

	// Keepalive heartbeat every 15 seconds to prevent browser/proxy timeout
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-s.done:
				return
			case <-ticker.C:
				fmt.Fprintf(s.w, ": heartbeat\n\n")
				s.flusher.Flush()
			}
		}
	}()

	return s
}

// WriteEvent writes an SSE event with the given type and data.
func (s *SSEWriter) WriteEvent(eventType, data string) {
	fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", eventType, data)
	s.flusher.Flush()
}

// Close stops the keepalive heartbeat.
func (s *SSEWriter) Close() {
	close(s.done)
}