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
	MaxTurns        int
	Provider        llm.Provider
	Workspace       string
	SystemPrompt    string
	ApprovalHandler ApprovalHandler
}

// Agent is the core harness that runs the agent loop.
type Agent struct {
	config          Config
	tools           *tools.Registry
	guardrail       *guardrail.Guardrail
	feedback        *feedback.Loop
	approvalHandler ApprovalHandler
}

// SetApprovalHandler sets the per-request approval handler.
func (a *Agent) SetApprovalHandler(h ApprovalHandler) {
	a.approvalHandler = h
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
// It is safe for concurrent use: all state is local to the call.
func (a *Agent) Run(ctx context.Context, userInput string) (string, error) {
	// 1. Build initial context (local to this call — no shared state)
	messages := []llm.Message{
		{Role: "system", Content: a.systemPrompt()},
		{Role: "user", Content: userInput},
	}

	for turn := 0; turn < a.config.MaxTurns; turn++ {
		// 2. Call LLM
		resp, err := a.config.Provider.ChatSync(ctx, messages)
		if err != nil {
			return "", fmt.Errorf("llm call: %w", err)
		}

		// 3. Parse actions
		actions := parser.Parse(resp)

		// 4. Stop condition: pure text response
		if actions.IsStop() {
			messages = append(messages, llm.Message{
				Role:    "assistant",
				Content: resp.Text,
			})
			return resp.Text, nil
		}

		// 5. Append assistant message with tool calls
		messages = append(messages, llm.Message{
			Role:      "assistant",
			Content:   resp.Text,
			ToolCalls: resp.ToolCalls,
		})

		// 6. Execute each action
		codeModified := false
		for _, action := range actions {
			// 6a. Check for parse errors
			if action.ParseError != nil {
				messages = append(messages, llm.Message{
					Role:       "tool",
					Content:    fmt.Sprintf("ERROR: failed to parse arguments: %v", action.ParseError),
					ToolCallID: action.ToolCallID,
				})
				continue
			}

			// 6b. Guardrail check
			if reason := a.guardrail.Check(action.Tool, action.Params); reason != nil {
				// Try HITL approval if handler is configured
				if a.approvalHandler != nil {
					cmd := ""
					if c, ok := action.Params["command"].(string); ok {
						cmd = c
					}
					resp, err := a.config.ApprovalHandler(ctx, ApprovalRequest{
						Tool:    action.Tool,
						Command: cmd,
						Reason:  reason.Message,
					})
					if err == nil && resp.Allowed {
						// User approved — proceed to execution
						goto execute
					}
				}
				// Blocked: inject rejection as tool result
				messages = append(messages, llm.Message{
					Role:       "tool",
					Content:    fmt.Sprintf("BLOCKED: %s - %s", reason.Level, reason.Message),
					ToolCallID: action.ToolCallID,
				})
				continue
			}

		execute:

			// 6c. Inject workspace into params for shell/git tools
			params := action.Params
			if action.Tool == "shell_run" || action.Tool == "git" || action.Tool == "test_run" {
				if params == nil {
					params = make(map[string]any)
				}
				if _, ok := params["workspace"]; !ok {
					params["workspace"] = a.config.Workspace
				}
			}

			// 6d. Execute tool
			result, err := a.tools.Execute(ctx, action.Tool, params)
			if err != nil {
				messages = append(messages, llm.Message{
					Role:       "tool",
					Content:    fmt.Sprintf("ERROR: %v", err),
					ToolCallID: action.ToolCallID,
				})
				continue
			}

			// 6e. Append tool result
			content := result.Output
			if !result.Success {
				content = "ERROR: " + result.Error + "\n" + result.Output
			}
			messages = append(messages, llm.Message{
				Role:       "tool",
				Content:    content,
				ToolCallID: action.ToolCallID,
			})

			// Track if code files were modified
			if action.Tool == "file_write" {
				codeModified = true
			}
		}

		// 6f. Feedback loop: run once per turn, only when code was modified
		if codeModified {
			fbResult := a.feedback.Run(ctx, a.config.Workspace)
			if fbResult.Status == "failed" {
				fb := &feedback.Feedback{
					Stage:   fbResult.Stage,
					Status:  "failed",
					Errors:  fbResult.Errors,
					Summary: fmt.Sprintf("%s failed: %d error(s)", fbResult.Stage, len(fbResult.Errors)),
				}
				messages = append(messages, fb.ToMessage())
			}
		}
	}

	return "", ErrMaxTurnsExceeded
}

func (a *Agent) systemPrompt() string {
	if a.config.SystemPrompt != "" {
		return a.config.SystemPrompt
	}
	return "You are Convallaria, a coding agent. You help users write, modify, and test code.\n" +
		"You have access to the following tools:\n" +
		"- file_read(path): Read a file\n" +
		"- file_write(path, content): Write a file\n" +
		"- shell_run(command): Run a shell command\n" +
		"- search(pattern, path): Search for a pattern in files\n" +
		"- test_run(path): Run tests\n" +
		"- git(operation, ...): Git operations\n" +
		"\n" +
		"Always think step by step. When you write code, you will receive automated feedback\n" +
		"from the build system, linter, and test runner. Use this feedback to fix issues."
}