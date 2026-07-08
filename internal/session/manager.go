// internal/session/manager.go
package session

import (
	"errors"
	"fmt"
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

// Manager manages sessions and their messages in memory.
// In production, this is backed by SQLite.
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	messages map[string][]llm.Message
	nextID   int
}

func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
		messages: make(map[string][]llm.Message),
	}
}

func (m *Manager) Create(title, projectDir string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nextID++
	sess := &Session{
		ID:         fmt.Sprintf("sess_%d", m.nextID),
		Title:      title,
		ProjectDir: projectDir,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	m.sessions[sess.ID] = sess
	m.messages[sess.ID] = make([]llm.Message, 0)
	return sess, nil
}

func (m *Manager) Get(id string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sess, ok := m.sessions[id]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return sess, nil
}

func (m *Manager) List() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sessions := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[id]; !ok {
		return ErrSessionNotFound
	}
	delete(m.sessions, id)
	delete(m.messages, id)
	return nil
}

func (m *Manager) AddMessage(sessionID string, msg llm.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[sessionID]; !ok {
		return ErrSessionNotFound
	}
	m.messages[sessionID] = append(m.messages[sessionID], msg)
	m.sessions[sessionID].UpdatedAt = time.Now()
	return nil
}

func (m *Manager) GetMessages(sessionID string) ([]llm.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, ok := m.sessions[sessionID]; !ok {
		return nil, ErrSessionNotFound
	}
	msgs := make([]llm.Message, len(m.messages[sessionID]))
	copy(msgs, m.messages[sessionID])
	return msgs, nil
}

// Export returns the session's messages as a formatted string.
func (m *Manager) Export(sessionID string) (string, error) {
	msgs, err := m.GetMessages(sessionID)
	if err != nil {
		return "", err
	}
	var result string
	for _, msg := range msgs {
		result += fmt.Sprintf("## %s\n\n%s\n\n", msg.Role, msg.Content)
	}
	return result, nil
}