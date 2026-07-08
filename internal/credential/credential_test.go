// internal/credential/credential_test.go
package credential

import (
	"testing"
)

func TestMemoryStore_SetAndGet(t *testing.T) {
	store := NewMemoryStore()
	err := store.Set("deepseek", "sk-test-key-12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	key, err := store.Get("deepseek")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "sk-test-key-12345" {
		t.Errorf("expected 'sk-test-key-12345', got %q", key)
	}
}

func TestMemoryStore_GetMissing(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.Get("nonexistent")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStore_List(t *testing.T) {
	store := NewMemoryStore()
	store.Set("deepseek", "sk-abc")
	store.Set("openai", "sk-xyz")

	providers := store.List()
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(providers))
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore()
	store.Set("deepseek", "sk-abc")
	err := store.Delete("deepseek")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = store.Get("deepseek")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestMaskKey(t *testing.T) {
	masked := MaskKey("sk-abcdefghijklmnop12345")
	if masked != "sk-****2345" {
		t.Errorf("expected 'sk-****2345', got %q", masked)
	}
}

func TestMaskKey_Short(t *testing.T) {
	masked := MaskKey("abc")
	if masked != "***" {
		t.Errorf("expected '***', got %q", masked)
	}
}