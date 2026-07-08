// internal/tools/file_reader.go
package tools

import (
	"context"
	"os"
)

type FileReader struct{}

func (f *FileReader) Name() string        { return "file_read" }
func (f *FileReader) Description() string { return "Read the contents of a file" }

func (f *FileReader) Execute(ctx context.Context, params map[string]any) (*Result, error) {
	path, _ := params["path"].(string)
	data, err := os.ReadFile(path)
	if err != nil {
		return &Result{Success: false, Error: err.Error()}, nil
	}
	return &Result{Output: string(data), Success: true}, nil
}