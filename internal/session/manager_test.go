// internal/session/manager_test.go
package session

import (
	"testing"

	"github.com/Convallariaxhr/convallaria/internal/llm"
)

func TestManager_CreateAndGetSession(t *testing.T) {
	mgr := NewManager()
	sess, err := mgr.Create("Test session", "/tmp/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess.Title != "Test session" {
		t.Errorf("expected title 'Test session', got %q", sess.Title)
	}

	got, err := mgr.Get(sess.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != sess.ID {
		t.Errorf("expected session ID %q, got %q", sess.ID, got.ID)
	}
}

func TestManager_ListSessions(t *testing.T) {
	mgr := NewManager()
	mgr.Create("Session A", "/tmp/a")
	mgr.Create("Session B", "/tmp/b")

	sessions := mgr.List()
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestManager_DeleteSession(t *testing.T) {
	mgr := NewManager()
	sess, _ := mgr.Create("To delete", "/tmp/test")

	err := mgr.Delete(sess.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = mgr.Get(sess.ID)
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestManager_AddMessage(t *testing.T) {
	mgr := NewManager()
	sess, _ := mgr.Create("Test", "/tmp/test")

	err := mgr.AddMessage(sess.ID, llm.Message{Role: "user", Content: "Hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = mgr.AddMessage(sess.ID, llm.Message{Role: "assistant", Content: "Hi!"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msgs, err := mgr.GetMessages(sess.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages, got %d", len(msgs))
	}
}

func TestManager_ExportSession(t *testing.T) {
	mgr := NewManager()
	sess, _ := mgr.Create("Export test", "/tmp/test")
	mgr.AddMessage(sess.ID, llm.Message{Role: "user", Content: "Hello"})
	mgr.AddMessage(sess.ID, llm.Message{Role: "assistant", Content: "World"})

	exported, err := mgr.Export(sess.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exported) == 0 {
		t.Error("expected non-empty export")
	}
}