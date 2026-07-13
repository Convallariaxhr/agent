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

	toolsExecuted := 0
	for turn := 0; turn < a.config.MaxTurns; turn++ {
		// 2. Call LLM
		resp, err := a.config.Provider.ChatSync(ctx, messages)
		if err != nil {
			return "", fmt.Errorf("llm call: %w", err)
		}

		// 3. Parse actions
		actions := parser.Parse(resp)

		// 4. Stop condition: pure text response — but check for hallucination
		if actions.IsStop() {
			// Only flag as hallucination if NO tools have been executed at all in this Run
			if toolsExecuted == 0 && a.detectHallucination(userInput, resp.Text) && turn < a.config.MaxTurns/2 {
				if fp, ok := a.config.Provider.(interface{ ForceToolUse() }); ok {
					fp.ForceToolUse()
				}
				messages = append(messages,
					llm.Message{Role: "assistant", Content: resp.Text},
					llm.Message{Role: "user", Content: "WARNING: You claimed to complete the task but did NOT call any tools. The user can see you are lying. You MUST use the tools now to actually perform the requested operation. Call file_write, shell_run, or the appropriate tool right now — do NOT apologize or explain, just do it."},
				)
				continue
			}
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
					resp, err := a.approvalHandler(ctx, ApprovalRequest{
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
				toolsExecuted++
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
			toolsExecuted++

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

// detectHallucination checks if the LLM claimed to have done something in its response.
// Only checks the response text — no user input keyword matching needed.
func (a *Agent) detectHallucination(_, response string) bool {
	lower := strings.ToLower(response)
	claimPatterns := []string{
		"done!", "created", "I've written", "I've made", "I've added",
		"I have written", "I have created", "I've changed", "I've updated",
		"I've modified", "has been created", "has been written",
		"搞定", "完成", "弄好", "改好", "写好", "建好", "做好了",
		"已经创建", "已经写", "已创建", "已写", "已经生成", "已生成",
		"已经修改", "已经完成", "已经更新", "已经改",
		"我已经", "我帮你", "我替",
	}
	for _, p := range claimPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
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
	return fmt.Sprintf(`You are Convallaria, a friendly and capable coding agent. Greet users warmly and help them with their programming tasks.

Current workspace: %s

You have access to these tools:
%s

CRITICAL — READ THIS:
- The ONLY way to create, read, modify, delete files is to call file_read / file_write.
- The ONLY way to run commands is to call shell_run.
- If you describe doing these things without calling the tools, you are LYING.
- EVERY file operation requires a tool call. Saying "I've created the file" without calling file_write is lying.
- The user can DETECT when you lie. Your response is checked for false claims.
- After writing a file, call file_read to verify it exists with the correct content.

Rules:
- Chat naturally. If the user says hello, greet them back.
- When asked to do something involving files or commands, call the tool FIRST, then report the result.
- Before operating on files, check what's in the directory using shell_run with "dir" (Windows) or "ls -la" (Unix).
- If a command fails, read the error and fix it.`, a.config.Workspace, strings.Join(toolDescs, "\n"))
}