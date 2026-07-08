// internal/guardrail/guardrail.go
package guardrail

import (
	"path/filepath"
	"regexp"
	"strings"
)

// Config configures guardrail behavior.
type Config struct {
	DangerousCommands bool
	FileScope         bool
	GitDangerousOps   bool
	Workspace         string
}

// BlockReason describes why an action was blocked.
type BlockReason struct {
	Level   string // "dangerous_command", "file_scope", "git_dangerous"
	Message string
}

// Guardrail checks actions against safety rules.
type Guardrail struct {
	config            Config
	dangerousPatterns []*regexp.Regexp
	gitDangerousOps   map[string]bool
}

// dangerous command patterns to block.
var defaultDangerousPatterns = []string{
	`rm\s+-rf\s+/`,
	`rm\s+-rf\s+/\*`,
	`mkfs\.`,
	`dd\s+if=`,
	`:\s*\(\s*\)\s*\{`,
	`chmod\s+777\s+/`,
	`>\s*/dev/sda`,
	`format\s+[a-zA-Z]:`,
	`shutdown`,
	`reboot`,
	`curl\s+.*\s*\|\s*(ba)?sh`,
	`wget\s+.*\s*-O\s*-?\s*\|\s*(ba)?sh`,
}

var gitDangerousOps = map[string]bool{
	"push --force": true,
	"reset --hard": true,
	"clean -fdx":   true,
}

func New(config Config) *Guardrail {
	g := &Guardrail{
		config:          config,
		gitDangerousOps: gitDangerousOps,
	}
	for _, p := range defaultDangerousPatterns {
		g.dangerousPatterns = append(g.dangerousPatterns, regexp.MustCompile(p))
	}
	return g
}

// Check evaluates an action and returns a BlockReason if it should be blocked.
// Returns nil if the action is safe.
func (g *Guardrail) Check(toolName string, params map[string]any) *BlockReason {
	// Layer 1: Dangerous commands
	if g.config.DangerousCommands {
		if cmd, ok := params["command"].(string); ok {
			for _, pattern := range g.dangerousPatterns {
				if pattern.MatchString(cmd) {
					return &BlockReason{
						Level:   "dangerous_command",
						Message: "Dangerous command blocked: " + cmd,
					}
				}
			}
		}
	}

	// Layer 2: File scope
	if g.config.FileScope {
		if path, ok := params["path"].(string); ok {
			if toolName == "file_write" || toolName == "file_read" {
				absPath, err := filepath.Abs(path)
				if err == nil {
					absWorkspace, _ := filepath.Abs(g.config.Workspace)
					rel, err := filepath.Rel(absWorkspace, absPath)
					if err != nil || strings.HasPrefix(rel, "..") {
						return &BlockReason{
							Level:   "file_scope",
							Message: "File outside workspace: " + path,
						}
					}
				}
			}
		}
	}

	// Layer 3: Git dangerous operations
	if g.config.GitDangerousOps {
		if toolName == "git" {
			op, _ := params["operation"].(string)
			force, _ := params["force"].(bool)
			if op == "push" && force {
				return &BlockReason{
					Level:   "git_dangerous",
					Message: "Dangerous git operation: push --force",
				}
			}
			if g.gitDangerousOps[op] {
				return &BlockReason{
					Level:   "git_dangerous",
					Message: "Dangerous git operation: " + op,
				}
			}
		}
	}

	return nil
}