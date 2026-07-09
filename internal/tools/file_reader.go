// internal/tools/file_reader.go
package tools

import (
	"context"
	"os"
	"path/filepath"
)

type FileReader struct{}

func (f *FileReader) Name() string        { return "file_read" }
func (f *FileReader) Description() string { return "Read the contents of a file" }
func (f *FileReader) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string", "description": "Path to the file to read"},
		},
		"required": []string{"path"},
	}
}

func (f *FileReader) Execute(ctx context.Context, params map[string]any) (*Result, error) {
	path, _ := params["path"].(string)
	// Resolve relative to workspace if provided
	if ws, ok := params["workspace"].(string); ok && ws != "" && !filepath.IsAbs(path) {
		path = filepath.Join(ws, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return &Result{Success: false, Error: err.Error()}, nil
	}
	return &Result{Output: string(data), Success: true}, nil
}