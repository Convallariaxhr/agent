// internal/agent/loop.go
package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

// Workspace returns the agent's configured workspace directory.
func (a *Agent) Workspace() string {
	return a.config.Workspace
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
// history contains previous messages in the conversation (may be nil for first turn).
// It is safe for concurrent use: all state is local to the call.
func (a *Agent) Run(ctx context.Context, userInput string, history []llm.Message) (string, error) {
	// 0. Build tool definitions and register with provider
	toolDefs := a.buildToolDefs()
	a.config.Provider.SetTools(toolDefs)

	// 1. Build context: system prompt + history + current user input
	messages := make([]llm.Message, 0, len(history)+2)
	messages = append(messages, llm.Message{Role: "system", Content: a.systemPrompt()})
	messages = append(messages, history...)
	messages = append(messages, llm.Message{Role: "user", Content: userInput})

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

			// 6c. Inject workspace into params for all path-based tools
			params := action.Params
			if params == nil {
				params = make(map[string]any)
			}
			if _, ok := params["workspace"]; !ok {
				params["workspace"] = a.config.Workspace
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

func (a *Agent) buildToolDefs() []llm.ToolDef {
	tools := a.tools.List()
	defs := make([]llm.ToolDef, len(tools))
	for i, t := range tools {
		defs[i] = llm.ToolDef{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Schema(),
		}
	}
	return defs
}

func (a *Agent) systemPrompt() string {
	if a.config.SystemPrompt != "" {
		return a.config.SystemPrompt
	}
	toolList := a.tools.List()
	toolDescs := make([]string, len(toolList))
	for i, t := range toolList {
		toolDescs[i] = fmt.Sprintf("- %s: %s", t.Name(), t.Description())
	}
	return fmt.Sprintf(`You are Convallaria, a coding agent. You help users write, modify, and test code.

Current workspace: %s

You have access to the following tools:
%s

Important rules:
- Always think step by step before acting.
- You CAN read and write files in the workspace and its subdirectories.
- You CAN run shell commands to explore directories, build, and test code.
- When you write code, you will receive automated feedback from the build system and test runner. Use this feedback to fix issues.
- Be persistent: if a command fails, try to understand why and fix it.
- Remember context from earlier in the conversation — the user may refer to things they mentioned before.`, a.config.Workspace, strings.Join(toolDescs, "\n"))
}