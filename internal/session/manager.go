// internal/session/manager.go
package session

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Convallariaxhr/convallaria/internal/llm"
)

var ErrSessionNotFound = errors.New("session not found")

// Session represents a chat session.
type Session struct {
	ID         string
	Title      string
	ProjectDir string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Store defines the persistence interface for sessions.
type Store interface {
	CreateSession(id, title, projectDir string) error
	GetSession(id string) (*Session, error)
	ListSessions() ([]*Session, error)
	DeleteSession(id string) error
	AddMessage(sessionID string, msg llm.Message) error
	GetMessages(sessionID string) ([]llm.Message, error)
}

// Manager manages sessions and their messages.
type Manager struct {
	store  Store
	nextID int
	mu     sync.Mutex
}

// NewManager creates a Manager with an in-memory store.
func NewManager() *Manager {
	return &Manager{store: newMemoryStore()}
}

// NewSQLiteManager creates a Manager backed by SQLite.
func NewSQLiteManager(dbPath string) (*Manager, error) {
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		return nil, err
	}
	m := &Manager{store: store}
	// Initialize nextID from existing sessions to avoid collisions after restart
	sessions, _ := store.ListSessions()
	for _, s := range sessions {
		// Session IDs are "sess_<number>"
		var n int
		if _, err := fmt.Sscanf(s.ID, "sess_%d", &n); err == nil && n >= m.nextID {
			m.nextID = n
		}
	}
	return m, nil
}

func (m *Manager) Create(title, projectDir string) (*Session, error) {
	m.mu.Lock()
	m.nextID++
	id := fmt.Sprintf("sess_%d", m.nextID)
	m.mu.Unlock()

	// Retry with incremented ID on collision (e.g., after server restart)
	for {
		err := m.store.CreateSession(id, title, projectDir)
		if err == nil {
			return m.store.GetSession(id)
		}
		// If collision, try next ID
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			m.mu.Lock()
			m.nextID++
			id = fmt.Sprintf("sess_%d", m.nextID)
			m.mu.Unlock()
			continue
		}
		return nil, err
	}
}

func (m *Manager) Get(id string) (*Session, error) {
	return m.store.GetSession(id)
}

func (m *Manager) List() []*Session {
	sessions, _ := m.store.ListSessions()
	if sessions == nil {
		sessions = []*Session{}
	}
	return sessions
}

func (m *Manager) Delete(id string) error {
	return m.store.DeleteSession(id)
}

func (m *Manager) AddMessage(sessionID string, msg llm.Message) error {
	return m.store.AddMessage(sessionID, msg)
}

func (m *Manager) GetMessages(sessionID string) ([]llm.Message, error) {
	return m.store.GetMessages(sessionID)
}

// Export returns the session's messages as a formatted string.
func (m *Manager) Export(sessionID string) (string, error) {
	msgs, err := m.store.GetMessages(sessionID)
	if err != nil {
		return "", err
	}
	var result string
	for _, msg := range msgs {
		result += fmt.Sprintf("## %s\n\n%s\n\n", msg.Role, msg.Content)
	}
	return result, nil
}

// memoryStore is the in-memory implementation of Store.
type memoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	messages map[string][]llm.Message
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		sessions: make(map[string]*Session),
		messages: make(map[string][]llm.Message),
	}
}

func (s *memoryStore) CreateSession(id, title, projectDir string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	s.sessions[id] = &Session{
		ID: id, Title: title, ProjectDir: projectDir,
		CreatedAt: now, UpdatedAt: now,
	}
	s.messages[id] = make([]llm.Message, 0)
	return nil
}

func (s *memoryStore) GetSession(id string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[id]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return sess, nil
}

func (s *memoryStore) ListSessions() ([]*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sessions := make([]*Session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		sessions = append(sessions, sess)
	}
	return sessions, nil
}

func (s *memoryStore) DeleteSession(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[id]; !ok {
		return ErrSessionNotFound
	}
	delete(s.sessions, id)
	delete(s.messages, id)
	return nil
}

func (s *memoryStore) AddMessage(sessionID string, msg llm.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[sessionID]; !ok {
		return ErrSessionNotFound
	}
	s.messages[sessionID] = append(s.messages[sessionID], msg)
	s.sessions[sessionID].UpdatedAt = time.Now()
	return nil
}

func (s *memoryStore) GetMessages(sessionID string) ([]llm.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.sessions[sessionID]; !ok {
		return nil, ErrSessionNotFound
	}
	msgs := make([]llm.Message, len(s.messages[sessionID]))
	copy(msgs, s.messages[sessionID])
	return msgs, nil
}