// internal/credential/credential.go
package credential

import (
	"errors"
	"sync"
)

var ErrNotFound = errors.New("credential not found")

// Store is the interface for secure credential storage.
type Store interface {
	Set(provider string, key string) error
	Get(provider string) (string, error)
	List() []string
	Delete(provider string) error
}

// MemoryStore is an in-memory credential store for testing.
// In production, this is replaced by OS keychain.
type MemoryStore struct {
	mu   sync.RWMutex
	keys map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{keys: make(map[string]string)}
}

func (s *MemoryStore) Set(provider string, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[provider] = key
	return nil
}

func (s *MemoryStore) Get(provider string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key, ok := s.keys[provider]
	if !ok {
		return "", ErrNotFound
	}
	return key, nil
}

func (s *MemoryStore) List() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	providers := make([]string, 0, len(s.keys))
	for p := range s.keys {
		providers = append(providers, p)
	}
	return providers
}

func (s *MemoryStore) Delete(provider string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.keys, provider)
	return nil
}

// MaskKey returns a masked version of an API key showing only the last 4 chars.
func MaskKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	// key[:3] includes the prefix (e.g. "sk-"), then "****" +
	// the last 4 characters of the key
	return key[:3] + "****" + key[len(key)-4:]
}