// internal/tools/file_writer.go
package tools

import (
	"context"
	"os"
	"path/filepath"
)

type FileWriter struct{}

func (f *FileWriter) Name() string        { return "file_write" }
func (f *FileWriter) Description() string { return "Write content to a file, creating it if necessary" }

func (f *FileWriter) Execute(ctx context.Context, params map[string]any) (*Result, error) {
	path, _ := params["path"].(string)
	content, _ := params["content"].(string)

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &Result{Success: false, Error: err.Error()}, nil
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return &Result{Success: false, Error: err.Error()}, nil
	}
	return &Result{Output: "File written: " + path, Success: true}, nil
}