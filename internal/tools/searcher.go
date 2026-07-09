// internal/tools/searcher.go
package tools

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Searcher struct{}

func (s *Searcher) Name() string { return "search" }
func (s *Searcher) Description() string {
	return "Search for a pattern in files using recursive directory scan"
}
func (s *Searcher) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{"type": "string", "description": "The text pattern to search for"},
			"path":    map[string]any{"type": "string", "description": "Directory to search in (defaults to workspace root)"},
		},
		"required": []string{"pattern"},
	}
}

func (s *Searcher) Execute(ctx context.Context, params map[string]any) (*Result, error) {
	pattern, _ := params["pattern"].(string)
	searchPath, _ := params["path"].(string)
	if searchPath == "" {
		searchPath = "."
	}
	// Resolve relative to workspace if provided
	if ws, ok := params["workspace"].(string); ok && ws != "" && !filepath.IsAbs(searchPath) {
		searchPath = filepath.Join(ws, searchPath)
	}

	var matches []string
	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable files
		}
		if info.IsDir() {
			// Skip hidden directories and .git
			name := info.Name()
			if strings.HasPrefix(name, ".") && name != "." {
				return filepath.SkipDir
			}
			return nil
		}
		// Skip binary files by extension
		ext := filepath.Ext(path)
		if isBinaryExt(ext) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if strings.Contains(line, pattern) {
				matches = append(matches, path+":"+strconv.Itoa(i+1)+":"+line)
			}
		}
		return nil
	})

	if err != nil {
		return &Result{Success: false, Error: err.Error()}, nil
	}
	if len(matches) == 0 {
		return &Result{Output: "No matches found", Success: true}, nil
	}
	return &Result{Output: strings.Join(matches, "\n"), Success: true}, nil
}

func isBinaryExt(ext string) bool {
	binary := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".o": true,
		".png": true, ".jpg": true, ".gif": true, ".zip": true,
		".tar": true, ".gz": true, ".pdf": true,
	}
	return binary[ext]
}