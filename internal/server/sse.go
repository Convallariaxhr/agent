// internal/server/sse.go
package server

import (
	"fmt"
	"net/http"
)

// SSEWriter writes Server-Sent Events to an http.ResponseWriter.
type SSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

// NewSSEWriter creates a new SSE writer and sends the initial headers.
func NewSSEWriter(w http.ResponseWriter) *SSEWriter {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		panic("http.ResponseWriter does not implement http.Flusher")
	}

	return &SSEWriter{w: w, flusher: flusher}
}

// WriteEvent writes an SSE event with the given type and data.
func (s *SSEWriter) WriteEvent(eventType, data string) {
	fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", eventType, data)
	s.flusher.Flush()
}