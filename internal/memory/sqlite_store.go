// internal/memory/sqlite_store.go
package memory

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

// SQLiteStore persists memory entries to SQLite with keyword search.
type SQLiteStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewSQLiteStore opens (or creates) a SQLite database for memory storage.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA foreign_keys=ON")

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS memories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			content TEXT NOT NULL,
			category TEXT DEFAULT '',
			file_path TEXT DEFAULT '',
			created_at INTEGER NOT NULL DEFAULT (unixepoch())
		)
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// Insert adds a new memory entry.
func (s *SQLiteStore) Insert(entry MemoryEntry) error {
	_, err := s.db.Exec(
		`INSERT INTO memories (content, category, file_path) VALUES (?, ?, ?)`,
		entry.Content, entry.Category, entry.FilePath,
	)
	return err
}

// Search performs keyword-based search over stored memories.
func (s *SQLiteStore) Search(query string, topK int) ([]MemoryEntry, error) {
	query = strings.ToLower(query)
	words := strings.Fields(query)
	if len(words) == 0 {
		return nil, nil
	}

	// Build LIKE conditions for each word
	conditions := make([]string, len(words))
	args := make([]any, len(words))
	for i, w := range words {
		conditions[i] = "LOWER(content) LIKE ?"
		args[i] = "%" + w + "%"
	}
	where := strings.Join(conditions, " OR ")

	rows, err := s.db.Query(
		fmt.Sprintf(`SELECT id, content, category, file_path, created_at FROM memories WHERE %s ORDER BY id DESC LIMIT ?`, where),
		append(args, topK)...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []MemoryEntry
	for rows.Next() {
		var e MemoryEntry
		if err := rows.Scan(&e.ID, &e.Content, &e.Category, &e.FilePath, &e.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, e)
	}
	return results, rows.Err()
}