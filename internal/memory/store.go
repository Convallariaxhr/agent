// internal/memory/store.go
package memory

import (
	"fmt"
	"strings"
	"sync"
)

// MemoryEntry represents a stored memory.
type MemoryEntry struct {
	ID        string
	Content   string
	Category  string
	FilePath  string
	CreatedAt int64
}

// Store is an in-memory memory store with keyword-based search.
// In production, this is replaced by SQLite + vector search.
type Store struct {
	mu      sync.RWMutex
	entries []MemoryEntry
	nextID  int
}

func NewStore() *Store {
	return &Store{}
}

// Insert adds a new memory entry.
func (s *Store) Insert(entry MemoryEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	entry.ID = fmt.Sprintf("mem_%d", s.nextID)
	s.entries = append(s.entries, entry)
	return nil
}

// Search performs a simple keyword-based search over stored memories.
// In production, this would use embedding vectors + cosine similarity.
func (s *Store) Search(query string, topK int) ([]MemoryEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query = strings.ToLower(query)
	queryWords := strings.Fields(query)

	type scoredEntry struct {
		entry MemoryEntry
		score int
	}
	var scored []scoredEntry

	for _, entry := range s.entries {
		content := strings.ToLower(entry.Content)
		score := 0
		for _, word := range queryWords {
			if strings.Contains(content, word) {
				score++
			}
		}
		if score > 0 {
			scored = append(scored, scoredEntry{entry: entry, score: score})
		}
	}

	// Simple sort by score descending
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	if len(scored) > topK {
		scored = scored[:topK]
	}

	results := make([]MemoryEntry, len(scored))
	for i, s := range scored {
		results[i] = s.entry
	}
	return results, nil
}