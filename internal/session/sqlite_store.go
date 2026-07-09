// internal/session/sqlite_store.go
package session

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Convallariaxhr/convallaria/internal/llm"
	_ "modernc.org/sqlite"
)

// SQLiteStore persists sessions and messages to SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens (or creates) a SQLite database for session storage.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Enable WAL mode for concurrent reads
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA foreign_keys=ON")

	if err := createSessionTables(db); err != nil {
		db.Close()
		return nil, err
	}

	// Run migration for existing databases
	migrateMessagesTable(db)

	return &SQLiteStore{db: db}, nil
}

func createSessionTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			project_dir TEXT DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			tool_call_id TEXT DEFAULT '',
			tool_calls TEXT DEFAULT '',
			FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
		);
	`)
	return err
}

// migrateMessagesTable adds tool_calls column if it doesn't exist (for existing databases).
func migrateMessagesTable(db *sql.DB) {
	db.Exec(`ALTER TABLE messages ADD COLUMN tool_calls TEXT DEFAULT ''`)
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// CreateSession inserts a new session.
func (s *SQLiteStore) CreateSession(id, title, projectDir string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT INTO sessions (id, title, project_dir, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		id, title, projectDir, now, now,
	)
	return err
}

// GetSession retrieves a session by ID.
func (s *SQLiteStore) GetSession(id string) (*Session, error) {
	row := s.db.QueryRow(`SELECT id, title, project_dir, created_at, updated_at FROM sessions WHERE id = ?`, id)
	var sess Session
	var createdAt, updatedAt string
	err := row.Scan(&sess.ID, &sess.Title, &sess.ProjectDir, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	sess.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	sess.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &sess, nil
}

// ListSessions returns all sessions ordered by last update.
func (s *SQLiteStore) ListSessions() ([]*Session, error) {
	rows, err := s.db.Query(`SELECT id, title, project_dir, created_at, updated_at FROM sessions ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var sess Session
		var createdAt, updatedAt string
		if err := rows.Scan(&sess.ID, &sess.Title, &sess.ProjectDir, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		sess.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		sess.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		sessions = append(sessions, &sess)
	}
	return sessions, rows.Err()
}

// DeleteSession removes a session and its messages.
func (s *SQLiteStore) DeleteSession(id string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	return err
}

// RenameSession updates the title of a session.
func (s *SQLiteStore) RenameSession(id, title string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(`UPDATE sessions SET title = ?, updated_at = ? WHERE id = ?`, title, now, id)
	return err
}

// AddMessage inserts a message for a session.
func (s *SQLiteStore) AddMessage(sessionID string, msg llm.Message) error {
	// Serialize ToolCalls to JSON
	toolCallsJSON := ""
	if len(msg.ToolCalls) > 0 {
		data, _ := json.Marshal(msg.ToolCalls)
		toolCallsJSON = string(data)
	}
	_, err := s.db.Exec(
		`INSERT INTO messages (session_id, role, content, tool_call_id, tool_calls) VALUES (?, ?, ?, ?, ?)`,
		sessionID, msg.Role, msg.Content, msg.ToolCallID, toolCallsJSON,
	)
	if err != nil {
		return err
	}
	// Update session timestamp
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.Exec(`UPDATE sessions SET updated_at = ? WHERE id = ?`, now, sessionID)
	return err
}

// GetMessages retrieves all messages for a session.
func (s *SQLiteStore) GetMessages(sessionID string) ([]llm.Message, error) {
	rows, err := s.db.Query(
		`SELECT role, content, tool_call_id, tool_calls FROM messages WHERE session_id = ? ORDER BY id ASC`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []llm.Message
	for rows.Next() {
		var msg llm.Message
		var toolCallsJSON string
		if err := rows.Scan(&msg.Role, &msg.Content, &msg.ToolCallID, &toolCallsJSON); err != nil {
			return nil, err
		}
		if toolCallsJSON != "" {
			json.Unmarshal([]byte(toolCallsJSON), &msg.ToolCalls)
		}
		msgs = append(msgs, msg)
	}
	return msgs, rows.Err()
}