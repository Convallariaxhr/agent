// internal/server/handler.go
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

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
	config    Config
	agent     *agent.Agent
	sessions  *session.Manager
	mux       *http.ServeMux
	approvals map[string]chan agent.ApprovalResponse
	appMu     sync.Mutex
	nextAppID int
}

// New creates a new Server.
func New(config Config, ag *agent.Agent, sessMgr *session.Manager) *Server {
	s := &Server{
		config:    config,
		agent:     ag,
		sessions:  sessMgr,
		mux:       http.NewServeMux(),
		approvals: make(map[string]chan agent.ApprovalResponse),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("/api/chat", s.handleChat)
	s.mux.HandleFunc("/api/approve", s.handleApprove)
	s.mux.HandleFunc("/api/files", s.handleFiles)
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
	// Handle CORS preflight
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.Message) > 32000 {
		http.Error(w, "Message too long (max 32000 chars)", http.StatusBadRequest)
		return
	}
	if req.Message == "" {
		http.Error(w, "Message is empty", http.StatusBadRequest)
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

	// Load conversation history for context
	history, _ := s.sessions.GetMessages(sessID)
	// Exclude the user message we just added (it will be passed separately)
	if len(history) > 0 {
		history = history[:len(history)-1]
	}

	sse := NewSSEWriter(w)
	sse.WriteEvent("session", jsonEncode(map[string]string{"id": sessID}))

	// Set up HITL approval for this request
	approvalCh := make(chan agent.ApprovalResponse, 1)
	appID := s.registerApproval(approvalCh)
	defer s.unregisterApproval(appID)

	s.agent.SetApprovalHandler(func(ctx context.Context, req agent.ApprovalRequest) (agent.ApprovalResponse, error) {
		// Send approval request to frontend via SSE
		sse.WriteEvent("approval_required", jsonEncode(map[string]any{
			"id":      appID,
			"tool":    req.Tool,
			"command": req.Command,
			"reason":  req.Reason,
		}))
		// Wait for user response
		select {
		case <-ctx.Done():
			return agent.ApprovalResponse{Allowed: false}, ctx.Err()
		case resp := <-approvalCh:
			return resp, nil
		}
	})

	resp, err := s.agent.Run(r.Context(), req.Message, history)
	if err != nil {
		sse.WriteEvent("error", jsonEncode(map[string]string{"message": err.Error()}))
		return
	}

	// Send entire response as a single token event (not per-character)
	sse.WriteEvent("token", jsonEncode(map[string]string{"token": resp}))
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

	case http.MethodPut:
		var req struct {
			Title string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		if err := s.sessions.Rename(id, req.Title); err != nil {
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listDir returns a list of files/dirs in the given directory.
func listDir(dir string) ([]map[string]any, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var result []map[string]any
	for _, e := range entries {
		result = append(result, map[string]any{
			"name":  e.Name(),
			"isDir": e.IsDir(),
		})
	}
	return result, nil
}
func jsonEncode(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return `{"error":"json encode failed"}`
	}
	return string(data)
}

// approval management

func (s *Server) registerApproval(ch chan agent.ApprovalResponse) string {
	s.appMu.Lock()
	defer s.appMu.Unlock()
	s.nextAppID++
	id := fmt.Sprintf("app_%d", s.nextAppID)
	s.approvals[id] = ch
	return id
}

func (s *Server) unregisterApproval(id string) {
	s.appMu.Lock()
	defer s.appMu.Unlock()
	delete(s.approvals, id)
}

func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID          string `json:"id"`
		Allowed     bool   `json:"allowed"`
		AlwaysAllow bool   `json:"always_allow"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	s.appMu.Lock()
	ch, ok := s.approvals[req.ID]
	s.appMu.Unlock()

	if !ok {
		http.Error(w, "Approval not found", http.StatusNotFound)
		return
	}

	ch <- agent.ApprovalResponse{Allowed: req.Allowed, AlwaysAllow: req.AlwaysAllow}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// File content endpoint
	if r.URL.Query().Has("path") {
		path := r.URL.Query().Get("path")
		// Restrict to workspace: resolve path and check with filepath.Rel
		absPath, err := filepath.Abs(path)
		absWorkspace, _ := filepath.Abs(s.agent.Workspace())
		rel, relErr := filepath.Rel(absWorkspace, absPath)
		if err != nil || relErr != nil || strings.HasPrefix(rel, "..") {
			http.Error(w, "Forbidden: path outside workspace", http.StatusForbidden)
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(data)
		return
	}

	dir := r.URL.Query().Get("dir")
	if dir == "" || dir == "." {
		dir = s.agent.Workspace()
	}

	entries, err := listDir(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}