// internal/memory/rules.go
package memory

import (
	"os"
	"path/filepath"
)

// LoadRules reads project-level rule files from the given directory.
// It looks for CONVALLARIA.md (and optionally CLAUDE.md for compatibility).
func LoadRules(projectDir string) (string, error) {
	var rules string

	for _, name := range []string{"CONVALLARIA.md", "CLAUDE.md"} {
		path := filepath.Join(projectDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if len(rules) > 0 {
			rules += "\n\n"
		}
		rules += string(data)
	}

	return rules, nil
}