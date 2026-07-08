// internal/server/handler.go
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Convallariaxhr/convallaria/internal/agent"
	"github.com/Convallariaxhr/convallaria/internal/llm"
	"github.com/Convallariaxhr/convallaria/internal/session"
)

// Config configures the HTTP server.
type Config struct {
	Port      int
	StaticDir string
}

// Server is the HTTP/SSE server for Convallaria.
type Server struct {
	config   Config
	agent    *agent.Agent
	sessions *session.Manager
	mux      *http.ServeMux
}

// New creates a new Server.
func New(config Config, ag *agent.Agent, sessMgr *session.Manager) *Server {
	s := &Server{
		config:   config,
		agent:    ag,
		sessions: sessMgr,
		mux:      http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("/api/chat", s.handleChat)
	s.mux.HandleFunc("/api/sessions", s.handleSessions)
	s.mux.HandleFunc("/api/sessions/", s.handleSessionByID)
	if s.config.StaticDir != "" {
		s.mux.Handle("/", http.FileServer(http.Dir(s.config.StaticDir)))
	}
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// Start begins listening on the configured port.
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Port)
	if s.config.Port == 0 {
		addr = ":8080"
	}
	return http.ListenAndServe(addr, s.mux)
}

type chatRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	sessID := req.SessionID
	if sessID == "" {
		sess, err := s.sessions.Create("New Chat", "")
		if err != nil {
			http.Error(w, "Failed to create session", http.StatusInternalServerError)
			return
		}
		sessID = sess.ID
	}

	s.sessions.AddMessage(sessID, llm.Message{Role: "user", Content: req.Message})

	sse := NewSSEWriter(w)
	sse.WriteEvent("session", fmt.Sprintf(`{"id":"%s"}`, sessID))

	resp, err := s.agent.Run(r.Context(), req.Message)
	if err != nil {
		sse.WriteEvent("error", fmt.Sprintf(`{"message":"%s"}`, err.Error()))
		return
	}

	for _, r := range resp {
		sse.WriteEvent("token", fmt.Sprintf(`{"token":"%s"}`, string(r)))
	}
	sse.WriteEvent("done", `{}`)

	s.sessions.AddMessage(sessID, llm.Message{Role: "assistant", Content: resp})
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessions := s.sessions.List()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func (s *Server) handleSessionByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/sessions/")

	switch r.Method {
	case http.MethodGet:
		msgs, err := s.sessions.GetMessages(id)
		if err != nil {
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(msgs)

	case http.MethodDelete:
		if err := s.sessions.Delete(id); err != nil {
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}