// internal/agent/loop.go
package agent

import (
	"context"
	"errors"
	"fmt"

	"github.com/Convallariaxhr/convallaria/internal/feedback"
	"github.com/Convallariaxhr/convallaria/internal/guardrail"
	"github.com/Convallariaxhr/convallaria/internal/llm"
	"github.com/Convallariaxhr/convallaria/internal/parser"
	"github.com/Convallariaxhr/convallaria/internal/tools"
)

var ErrMaxTurnsExceeded = errors.New("max turns exceeded")

// Config configures the agent.
type Config struct {
	MaxTurns     int
	Provider     llm.Provider
	Workspace    string
	SystemPrompt string
}

// Agent is the core harness that runs the agent loop.
type Agent struct {
	config    Config
	tools     *tools.Registry
	guardrail *guardrail.Guardrail
	feedback  *feedback.Loop
	messages  []llm.Message
}

// New creates a new Agent with default tools and mechanisms.
func New(config Config) *Agent {
	reg := tools.NewRegistry()
	reg.Register("file_read", &tools.FileReader{})
	reg.Register("file_write", &tools.FileWriter{})
	reg.Register("shell_run", &tools.ShellRunner{})
	reg.Register("search", &tools.Searcher{})
	reg.Register("test_run", &tools.TestRunner{})
	reg.Register("git", &tools.GitOps{})

	g := guardrail.New(guardrail.Config{
		DangerousCommands: true,
		FileScope:         true,
		GitDangerousOps:   true,
		Workspace:         config.Workspace,
	})

	return &Agent{
		config:    config,
		tools:     reg,
		guardrail: g,
		feedback:  feedback.NewLoop(),
	}
}

// Run executes the agent main loop for a single user input.
func (a *Agent) Run(ctx context.Context, userInput string) (string, error) {
	// 1. Build initial context
	a.messages = []llm.Message{
		{Role: "system", Content: a.systemPrompt()},
		{Role: "user", Content: userInput},
	}

	var finalText string

	for turn := 0; turn < a.config.MaxTurns; turn++ {
		// 2. Call LLM
		resp, err := a.config.Provider.ChatSync(ctx, a.messages)
		if err != nil {
			return "", fmt.Errorf("llm call: %w", err)
		}

		// 3. Parse actions
		actions := parser.Parse(resp)

		// 4. Stop condition: pure text response
		if actions.IsStop() {
			finalText = resp.Text
			a.messages = append(a.messages, llm.Message{
				Role:    "assistant",
				Content: resp.Text,
			})
			return finalText, nil
		}

		// 5. Append assistant message with tool calls
		a.messages = append(a.messages, llm.Message{
			Role:      "assistant",
			Content:   resp.Text,
			ToolCalls: resp.ToolCalls,
		})

		// 6. Execute each action
		codeModified := false
		for _, action := range actions {
			// 6a. Guardrail check
			if reason := a.guardrail.Check(action.Tool, action.Params); reason != nil {
				// Blocked: inject rejection as tool result
				a.messages = append(a.messages, llm.Message{
					Role:       "tool",
					Content:    fmt.Sprintf("BLOCKED: %s - %s", reason.Level, reason.Message),
					ToolCallID: action.ToolCallID,
				})
				continue
			}

			// 6b. Execute tool
			result, err := a.tools.Execute(ctx, action.Tool, action.Params)
			if err != nil {
				// Execution error
				a.messages = append(a.messages, llm.Message{
					Role:       "tool",
					Content:    fmt.Sprintf("ERROR: %v", err),
					ToolCallID: action.ToolCallID,
				})
				continue
			}

			// 6c. Append tool result
			content := result.Output
			if !result.Success {
				content = "ERROR: " + result.Error + "\n" + result.Output
			}
			a.messages = append(a.messages, llm.Message{
				Role:       "tool",
				Content:    content,
				ToolCallID: action.ToolCallID,
			})

			// Track if code files were modified
			if action.Tool == "file_write" || action.Tool == "shell_run" {
				codeModified = true
			}
		}

		// 6d. Feedback loop: run once per turn after all actions
		if codeModified {
			fbResult := a.feedback.Run(ctx, a.config.Workspace)
			if fbResult.Status == "failed" {
				fb := &feedback.Feedback{
					Stage:   fbResult.Stage,
					Status:  "failed",
					Errors:  fbResult.Errors,
					Summary: fmt.Sprintf("%s failed: %d error(s)", fbResult.Stage, len(fbResult.Errors)),
				}
				a.messages = append(a.messages, fb.ToMessage())
			}
		}
	}

	return "", ErrMaxTurnsExceeded
}

func (a *Agent) systemPrompt() string {
	if a.config.SystemPrompt != "" {
		return a.config.SystemPrompt
	}
	return `You are Convallaria, a coding agent. You help users write, modify, and test code.
You have access to the following tools:
- file_read(path): Read a file
- file_write(path, content): Write a file
- shell_run(command): Run a shell command
- search(pattern, path): Search for a pattern in files
- test_run(path): Run tests
- git(operation, ...): Git operations

Always think step by step. When you write code, you will receive automated feedback
from the build system, linter, and test runner. Use this feedback to fix issues.`
}