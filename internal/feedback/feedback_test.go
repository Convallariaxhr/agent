// internal/feedback/feedback_test.go
package feedback

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initGoModule creates a minimal go.mod in the given directory.
func initGoModule(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("go", "mod", "init", "test")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to init go module: %v\n%s", err, out)
	}
}

func TestBuildValidator_ValidGoFile(t *testing.T) {
	dir := t.TempDir()
	initGoModule(t, dir)
	goFile := filepath.Join(dir, "main.go")
	os.WriteFile(goFile, []byte("package main\n\nfunc main() { println(\"hello\") }\n"), 0644)

	v := &BuildValidator{}
	fb := v.Validate(context.Background(), dir)
	if fb.Status != "passed" {
		t.Errorf("expected passed, got %s: %s", fb.Status, fb.Summary)
	}
}

func TestBuildValidator_InvalidGoFile(t *testing.T) {
	dir := t.TempDir()
	initGoModule(t, dir)
	goFile := filepath.Join(dir, "main.go")
	os.WriteFile(goFile, []byte("package main\n\nfunc main() {\n\tundefinedVar\n}\n"), 0644)

	v := &BuildValidator{}
	fb := v.Validate(context.Background(), dir)
	if fb.Status == "passed" {
		t.Error("expected build failure for invalid Go file")
	}
	if len(fb.Errors) == 0 {
		t.Error("expected at least one error")
	}
}

func TestFeedbackLoop_AllPass(t *testing.T) {
	dir := t.TempDir()
	initGoModule(t, dir)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() { println(\"ok\") }\n"), 0644)

	loop := NewLoop()
	result := loop.Run(context.Background(), dir)
	if result.Status != "passed" {
		t.Errorf("expected all pass, got %s", result.Status)
	}
}

func TestFeedbackLoop_BuildFailure(t *testing.T) {
	dir := t.TempDir()
	initGoModule(t, dir)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {\n\tbroken\n}\n"), 0644)

	loop := NewLoop()
	result := loop.Run(context.Background(), dir)
	if result.Status != "failed" {
		t.Errorf("expected failed, got %s", result.Status)
	}
	if result.Stage != "build" {
		t.Errorf("expected failure at build stage, got %s", result.Stage)
	}
}

func TestFeedbackToMessage(t *testing.T) {
	fb := &Feedback{
		Stage:  "build",
		Status: "failed",
		Errors: []FeedbackError{
			{File: "main.go", Line: 3, Column: 2, Message: "undefined: broken"},
		},
		Summary: "Build failed: 1 error",
	}
	msg := fb.ToMessage()
	if msg.Role != "tool" {
		t.Errorf("expected role 'tool', got %q", msg.Role)
	}
	if msg.Name != "feedback" {
		t.Errorf("expected name 'feedback', got %q", msg.Name)
	}
}