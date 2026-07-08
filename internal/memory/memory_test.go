// internal/memory/memory_test.go
package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRulesLoader_LoadsCONVALLARIAMD(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CONVALLARIA.md"), []byte("# Project Rules\n\n- Use tabs for indentation"), 0644)

	rules, err := LoadRules(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rules == "" {
		t.Error("expected non-empty rules")
	}
	if !contains(rules, "Use tabs for indentation") {
		t.Errorf("expected rules to contain 'Use tabs for indentation', got %q", rules)
	}
}

func TestRulesLoader_NoFile(t *testing.T) {
	dir := t.TempDir()
	rules, err := LoadRules(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rules != "" {
		t.Errorf("expected empty rules, got %q", rules)
	}
}

func TestMemoryStore_InsertAndSearch(t *testing.T) {
	store := NewStore()
	err := store.Insert(MemoryEntry{
		Content:  "We decided to use Go modules for dependency management",
		Category: "decision",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	results, err := store.Search("Go dependency management", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected at least one search result")
	}
}

func TestMemoryStore_SearchEmpty(t *testing.T) {
	store := NewStore()
	results, err := store.Search("nonexistent query", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}