// internal/server/server_test.go
package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Convallariaxhr/convallaria/internal/agent"
	"github.com/Convallariaxhr/convallaria/internal/llm"
	"github.com/Convallariaxhr/convallaria/internal/session"
)

func TestServer_ChatEndpoint_SSE(t *testing.T) {
	mock := llm.NewMockProvider()
	mock.AddResponse(llm.MockTextResponse("Hello! I'm Convallaria."))

	ag := agent.New(agent.Config{
		MaxTurns:     5,
		Provider:     mock,
		Workspace:    "/tmp/test",
		SystemPrompt: "You are a helpful coding agent.",
	})

	sessMgr := session.NewManager()
	srv := New(Config{Port: 0}, ag, sessMgr)

	body := `{"session_id":"","message":"Hello"}`
	req := httptest.NewRequest("POST", "/api/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	// Verify SSE content type
	ct := w.Header().Get("Content-Type")
	if ct != "text/event-stream" {
		t.Errorf("expected text/event-stream, got %q", ct)
	}
}

func TestServer_SessionListEndpoint(t *testing.T) {
	sessMgr := session.NewManager()
	sessMgr.Create("Test session", "/tmp/test")

	srv := New(Config{Port: 0}, nil, sessMgr)

	req := httptest.NewRequest("GET", "/api/sessions", nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var sessions []map[string]any
	json.Unmarshal(w.Body.Bytes(), &sessions)
	if len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions))
	}
}

func TestSSEWriter_WriteEvent(t *testing.T) {
	w := httptest.NewRecorder()
	sse := NewSSEWriter(w)

	sse.WriteEvent("token", `{"token":"H"}`)
	sse.WriteEvent("done", `{}`)

	body := w.Body.String()
	if !strings.Contains(body, "event: token") {
		t.Error("expected 'event: token' in SSE output")
	}
	if !strings.Contains(body, "event: done") {
		t.Error("expected 'event: done' in SSE output")
	}
}