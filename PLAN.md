# Convallaria Coding Agent Harness · Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a complete coding agent harness (Go backend + Material Design 3 Web UI) with mock-LLM-driven deterministic unit tests for all core mechanisms.

**Architecture:** Go backend provides HTTP/SSE server, agent main loop, tool execution, guardrails, feedback loop, memory, and session management. Web UI communicates via SSE for streaming responses. LLM abstraction layer supports multiple providers (DeepSeek/OpenAI/Anthropic) with mock implementation for testing.

**Tech Stack:** Go 1.22+, SQLite (go-sqlite3), Material Design 3 + Open Design, SSE (Server-Sent Events)

## Implementation Status (All Complete ✅)

| Phase | Task | Status | Key Commit |
|-------|------|--------|------------|
| 1 | 1.1 Go module init | ✅ | `6bc06d6` |
| 1 | 1.2 LLM Provider + Mock | ✅ | `437c9f2` |
| 2 | 2.1 Config system | ✅ | `7889f4e` |
| 2 | 2.2 Credential management | ✅ | `e864579` |
| 3 | 3.1 Action Parser | ✅ | `c99d266` |
| 3 | 3.2 Tool registry + 6 tools | ✅ | `1c0957d` |
| 4 | 4.1 Guardrail (3-layer) | ✅ | `1a2e207` |
| 5 | 5.1 Feedback loop | ✅ | `3e1dcac` |
| 6 | 6.1 Agent main loop | ✅ | `d3af108` |
| 7 | 7.1 Context window manager | ✅ | `8b89014` |
| 7 | 7.2 Error recovery | ✅ | `8b89014` |
| 8 | 8.1 Memory system | ✅ | `46a4e44` |
| 9 | 9.1 Session management | ✅ | `51d1ffe` |
| 10 | 10.1 HTTP/SSE server | ✅ | `f42cfe9` |
| 11 | 11.1 Web UI | ✅ | `213b6e0` |
| 12 | 12.1 CLI entry point | ✅ | `213b6e0` |
| 13 | CI/CD + Docker | ✅ | `f52e06a`, `8017960` |
| - | Code review fixes | ✅ | `9e33029` |
| - | DeepSeek provider | ✅ | `3cdac3f` |
| - | SQLite persistence | ✅ | `da04eef` |
| - | HITL approval | ✅ | `ea88223` |
| - | File browser + config | ✅ | `d21d3f7` |
| - | Multi-provider support | ✅ | `8017960` |
| - | Anti-hallucination | ✅ | `200b269`, `3472191` |
| - | OpenCode review fixes | ✅ | `bb029de`, `3344796` |
| - | Function calling format | ✅ | `5cf032a`, `125181d` |
| - | UI enhancements | ✅ | `db3ec64`, `a45c2f5`, `0a79d82` |
| - | Final docs + delivery | ✅ | `f023b8e`, `fa7b0b4` |

---

## Phase 1: Project Scaffold & LLM Abstraction

### Task 1.1: Initialize Go module and project structure

**Files:**
- Create: `go.mod`
- Create: `cmd/convallaria/main.go` (stub)

- [ ] **Step 0: Verify environment prerequisites**

```bash
go version  # Expected: go version go1.22.x or later
git --version
```
If Go is not installed, download from https://go.dev/dl/ and install.

- [ ] **Step 1: Initialize Go module**

```bash
cd /path/to/your/project  # Replace with your actual project directory
go mod init github.com/Convallariaxhr/convallaria
```
Expected: `go.mod` created

- [ ] **Step 2: Create directory structure and .gitignore**

```bash
# Bash / Git Bash / WSL:
mkdir -p cmd/convallaria internal/{agent,llm,parser,tools,guardrail,feedback,memory,context,recovery,session,config,server,credential} web test/integration

# PowerShell (Windows):
# New-Item -ItemType Directory -Force -Path cmd/convallaria, internal/agent, internal/llm, internal/parser, internal/tools, internal/guardrail, internal/feedback, internal/memory, internal/context, internal/recovery, internal/session, internal/config, internal/server, internal/credential, web, test/integration
```

Create `.gitignore`:
```
# Binaries
*.exe
convallaria
convallaria.exe

# Environment
.env
.env.local

# IDE
.idea/
.vscode/
*.swp

# Dependencies
vendor/

# Build artifacts
dist/
```

- [ ] **Step 3: Create stub main.go**

```go
// cmd/convallaria/main.go
package main

import "fmt"

func main() {
    fmt.Println("Convallaria Coding Agent Harness")
}
```

- [ ] **Step 4: Verify build**

```bash
go build ./cmd/convallaria/
```
Expected: Build succeeds, no errors

- [x] **Step 5: Commit** ✅

```bash
git add -A && git commit -m "feat: initialize Go module and project structure"
```

---

### Task 1.2: Define LLM Provider interface and Mock implementation

**Files:**
- Create: `internal/llm/provider.go`
- Create: `internal/llm/mock.go`
- Create: `internal/llm/mock_test.go`

- [ ] **Step 1: Write the interface definition**

```go
// internal/llm/provider.go
package llm

import "context"

// Message represents a single chat message.
type Message struct {
    Role    string `json:"role"`    // system, user, assistant, tool
    Content string `json:"content"`
    Name    string `json:"name,omitempty"`
    // ToolCalls is populated for assistant messages that request tool execution.
    ToolCalls []ToolCall `json:"tool_calls,omitempty"`
    // ToolCallID links tool result messages to the originating call.
    ToolCallID string `json:"tool_call_id,omitempty"`
}

// ToolCall represents a function call requested by the LLM.
type ToolCall struct {
    ID       string       `json:"id"`
    Function FunctionCall `json:"function"`
}

// FunctionCall contains the name and arguments of a tool invocation.
type FunctionCall struct {
    Name      string `json:"name"`
    Arguments string `json:"arguments"` // JSON-encoded
}

// StreamEvent is emitted during streaming responses.
type StreamEvent struct {
    Type    string     `json:"type"` // "token", "tool_call", "done", "error"
    Token   string     `json:"token,omitempty"`
    ToolCall *ToolCall `json:"tool_call,omitempty"`
    Error   error      `json:"-"`
}

// Response is the final complete response from the LLM.
type Response struct {
    Text      string     `json:"text"`
    ToolCalls []ToolCall `json:"tool_calls,omitempty"`
    StopReason string    `json:"stop_reason"` // "stop", "tool_calls", "max_tokens"
}

// Provider defines the interface for LLM interactions.
type Provider interface {
    // Chat sends messages and returns a channel of streaming events.
    Chat(ctx context.Context, messages []Message) (<-chan StreamEvent, error)
    // ChatSync sends messages and returns a complete response (no streaming).
    ChatSync(ctx context.Context, messages []Message) (*Response, error)
}
```

- [ ] **Step 2: Write the Mock implementation**

```go
// internal/llm/mock.go
package llm

import (
    "context"
    "encoding/json"
)

// MockProvider returns preset responses for deterministic testing.
type MockProvider struct {
    // Responses is a queue of responses to return.
    Responses []*Response
    // Events is a queue of event sequences for streaming.
    Events [][]StreamEvent
    callCount int
}

func NewMockProvider() *MockProvider {
    return &MockProvider{}
}

// AddResponse appends a sync response to the queue.
func (m *MockProvider) AddResponse(r *Response) {
    m.Responses = append(m.Responses, r)
}

// AddEvents appends a streaming event sequence to the queue.
func (m *MockProvider) AddEvents(events []StreamEvent) {
    m.Events = append(m.Events, events)
}

func (m *MockProvider) Chat(ctx context.Context, messages []Message) (<-chan StreamEvent, error) {
    ch := make(chan StreamEvent, 100)
    go func() {
        defer close(ch)
        if m.callCount < len(m.Events) {
            for _, ev := range m.Events[m.callCount] {
                select {
                case <-ctx.Done():
                    return
                case ch <- ev:
                }
            }
        } else if m.callCount < len(m.Responses) {
            resp := m.Responses[m.callCount]
            for _, r := range resp.Text {
                select {
                case <-ctx.Done():
                    return
                case ch <- StreamEvent{Type: "token", Token: string(r)}:
                }
            }
            for _, tc := range resp.ToolCalls {
                select {
                case <-ctx.Done():
                    return
                case ch <- StreamEvent{Type: "tool_call", ToolCall: &tc}:
                }
            }
            select {
            case <-ctx.Done():
                return
            case ch <- StreamEvent{Type: "done"}:
            }
        } else {
            select {
            case <-ctx.Done():
                return
            case ch <- StreamEvent{Type: "done"}:
            }
        }
        m.callCount++
    }()
    return ch, nil
}

func (m *MockProvider) ChatSync(ctx context.Context, messages []Message) (*Response, error) {
    if m.callCount < len(m.Responses) {
        resp := m.Responses[m.callCount]
        m.callCount++
        return resp, nil
    }
    return &Response{Text: "", StopReason: "stop"}, nil
}

// CallCount returns the number of times Chat/ChatSync has been called.
func (m *MockProvider) CallCount() int {
    return m.callCount
}

// Helper to create a tool call response quickly.
func MockToolCallResponse(id, name, args string) *Response {
    return &Response{
        ToolCalls: []ToolCall{
            {ID: id, Function: FunctionCall{Name: name, Arguments: args}},
        },
        StopReason: "tool_calls",
    }
}

// Helper to create a text response quickly.
func MockTextResponse(text string) *Response {
    return &Response{Text: text, StopReason: "stop"}
}
```

- [ ] **Step 3: Write the Mock tests**

```go
// internal/llm/mock_test.go
package llm

import (
    "context"
    "testing"
)

func TestMockProvider_ChatSync_ReturnsPresetResponse(t *testing.T) {
    mock := NewMockProvider()
    mock.AddResponse(MockTextResponse("Hello, world!"))

    resp, err := mock.ChatSync(context.Background(), nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if resp.Text != "Hello, world!" {
        t.Errorf("expected 'Hello, world!', got %q", resp.Text)
    }
    if resp.StopReason != "stop" {
        t.Errorf("expected stop reason 'stop', got %q", resp.StopReason)
    }
}

func TestMockProvider_ChatSync_ReturnsToolCall(t *testing.T) {
    mock := NewMockProvider()
    mock.AddResponse(MockToolCallResponse("call_1", "file_write", `{"path":"main.go","content":"package main"}`))

    resp, err := mock.ChatSync(context.Background(), nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(resp.ToolCalls) != 1 {
        t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
    }
    if resp.ToolCalls[0].Function.Name != "file_write" {
        t.Errorf("expected tool 'file_write', got %q", resp.ToolCalls[0].Function.Name)
    }
}

func TestMockProvider_Chat_StreamsTokens(t *testing.T) {
    mock := NewMockProvider()
    mock.AddResponse(MockTextResponse("Hi!"))

    ch, err := mock.Chat(context.Background(), nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    var tokens []string
    for ev := range ch {
        if ev.Type == "token" {
            tokens = append(tokens, ev.Token)
        }
    }
    result := ""
    for _, tok := range tokens {
        result += tok
    }
    if result != "Hi!" {
        t.Errorf("expected 'Hi!', got %q", result)
    }
}

func TestMockProvider_CallCount(t *testing.T) {
    mock := NewMockProvider()
    mock.AddResponse(MockTextResponse("a"))
    mock.AddResponse(MockTextResponse("b"))

    mock.ChatSync(context.Background(), nil)
    if mock.CallCount() != 1 {
        t.Errorf("expected call count 1, got %d", mock.CallCount())
    }
    mock.ChatSync(context.Background(), nil)
    if mock.CallCount() != 2 {
        t.Errorf("expected call count 2, got %d", mock.CallCount())
    }
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/llm/ -v
```
Expected: All 4 tests PASS

- [x] **Step 5: Commit** ✅

```bash
git add -A && git commit -m "feat: add LLM Provider interface and Mock implementation"
```

---

## Phase 2: Config & Credential

### Task 2.1: Configuration system

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/config/config_test.go
package config

import (
    "os"
    "path/filepath"
    "testing"
)

func TestLoadConfig_FromYAML(t *testing.T) {
    dir := t.TempDir()
    yamlContent := `
llm:
  provider: deepseek
  model: deepseek-chat
  max_tokens: 4096
agent:
  max_turns: 50
  workspace: /tmp/test
`
    yamlPath := filepath.Join(dir, "convallaria.yaml")
    os.WriteFile(yamlPath, []byte(yamlContent), 0644)

    cfg, err := Load(yamlPath)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if cfg.LLM.Provider != "deepseek" {
        t.Errorf("expected provider deepseek, got %q", cfg.LLM.Provider)
    }
    if cfg.LLM.MaxTokens != 4096 {
        t.Errorf("expected max_tokens 4096, got %d", cfg.LLM.MaxTokens)
    }
    if cfg.Agent.MaxTurns != 50 {
        t.Errorf("expected max_turns 50, got %d", cfg.Agent.MaxTurns)
    }
}

func TestLoadConfig_Defaults(t *testing.T) {
    dir := t.TempDir()
    yamlPath := filepath.Join(dir, "convallaria.yaml")
    os.WriteFile(yamlPath, []byte(""), 0644)

    cfg, err := Load(yamlPath)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if cfg.Agent.MaxTurns != 30 {
        t.Errorf("expected default max_turns 30, got %d", cfg.Agent.MaxTurns)
    }
    if cfg.LLM.Provider != "deepseek" {
        t.Errorf("expected default provider deepseek, got %q", cfg.LLM.Provider)
    }
}

func TestLoadConfig_EnvOverride(t *testing.T) {
    dir := t.TempDir()
    yamlPath := filepath.Join(dir, "convallaria.yaml")
    os.WriteFile(yamlPath, []byte("llm:\n  provider: openai"), 0644)

    os.Setenv("CONVALLARIA_PROVIDER", "deepseek")
    defer os.Unsetenv("CONVALLARIA_PROVIDER")

    cfg, err := Load(yamlPath)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if cfg.LLM.Provider != "deepseek" {
        t.Errorf("expected env override deepseek, got %q", cfg.LLM.Provider)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/config/ -v
```
Expected: FAIL (package not found or compile errors)

- [ ] **Step 3: Write the config implementation**

```go
// internal/config/config.go
package config

import (
    "os"

    "gopkg.in/yaml.v3"
)

// Config holds all configuration for Convallaria.
type Config struct {
    LLM      LLMConfig      `yaml:"llm"`
    Agent    AgentConfig    `yaml:"agent"`
    Context  ContextConfig  `yaml:"context"`
    Tools    ToolsConfig    `yaml:"tools"`
    Guardrails GuardrailsConfig `yaml:"guardrails"`
    Memory   MemoryConfig   `yaml:"memory"`
}

type LLMConfig struct {
    Provider    string  `yaml:"provider"`
    Model       string  `yaml:"model"`
    MaxTokens   int     `yaml:"max_tokens"`
    Temperature float64 `yaml:"temperature"`
    APIKeyEnv   string  `yaml:"api_key_env"`
}

type AgentConfig struct {
    MaxTurns  int    `yaml:"max_turns"`
    Workspace string `yaml:"workspace"`
}

type ContextConfig struct {
    MaxContextTokens     int     `yaml:"max_context_tokens"`
    CompressionThreshold float64 `yaml:"compression_threshold"`
}

type ToolsConfig struct {
    Shell ShellConfig `yaml:"shell"`
    File  FileConfig  `yaml:"file"`
    Git   GitConfig   `yaml:"git"`
}

type ShellConfig struct {
    Enabled bool `yaml:"enabled"`
    Timeout int  `yaml:"timeout"`
}

type FileConfig struct {
    AllowedPaths []string `yaml:"allowed_paths"`
}

type GitConfig struct {
    AutoCommit bool `yaml:"auto_commit"`
}

type GuardrailsConfig struct {
    DangerousCommands bool `yaml:"dangerous_commands"`
    FileScope         bool `yaml:"file_scope"`
    GitDangerousOps   bool `yaml:"git_dangerous_ops"`
}

type MemoryConfig struct {
    VectorStore    string `yaml:"vector_store"`
    TopK           int    `yaml:"top_k"`
    EmbeddingModel string `yaml:"embedding_model"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
    return &Config{
        LLM: LLMConfig{
            Provider:    "deepseek",
            Model:       "deepseek-chat",
            MaxTokens:   4096,
            Temperature: 0.0,
            APIKeyEnv:   "DEEPSEEK_API_KEY",
        },
        Agent: AgentConfig{
            MaxTurns:  30,
            Workspace: ".",
        },
        Context: ContextConfig{
            MaxContextTokens:     64000,
            CompressionThreshold: 0.8,
        },
        Tools: ToolsConfig{
            Shell: ShellConfig{Enabled: true, Timeout: 120},
            File:  FileConfig{AllowedPaths: []string{"."}},
            Git:   GitConfig{AutoCommit: false},
        },
        Guardrails: GuardrailsConfig{
            DangerousCommands: true,
            FileScope:         true,
            GitDangerousOps:   true,
        },
        Memory: MemoryConfig{
            VectorStore:    "sqlite",
            TopK:           5,
            EmbeddingModel: "local",
        },
    }
}

// Load reads a YAML config file and applies defaults + env overrides.
func Load(path string) (*Config, error) {
    cfg := DefaultConfig()

    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            // Return defaults if no config file exists
            // (but still apply env overrides)
            cfg.applyEnvOverrides()
            return cfg, nil
        }
        return nil, err
    }

    if len(data) > 0 {
        if err := yaml.Unmarshal(data, cfg); err != nil {
            return nil, err
        }
    }

    cfg.applyEnvOverrides()
    return cfg, nil
}

func (c *Config) applyEnvOverrides() {
    if v := os.Getenv("CONVALLARIA_PROVIDER"); v != "" {
        c.LLM.Provider = v
    }
    if v := os.Getenv("CONVALLARIA_MODEL"); v != "" {
        c.LLM.Model = v
    }
    if v := os.Getenv("CONVALLARIA_API_KEY"); v != "" {
        // Key itself is handled by credential package, but env var name is stored here
        c.LLM.APIKeyEnv = "CONVALLARIA_API_KEY"
        _ = v
    }
}
```

- [ ] **Step 4: Add yaml dependency**

```bash
# If proxy.golang.org is blocked (common in China):
#   GOPROXY=https://goproxy.cn,direct go get gopkg.in/yaml.v3
# Or use direct:
#   GOPROXY=direct go get gopkg.in/yaml.v3
go get gopkg.in/yaml.v3
```

> **网络备用方案**：如果 `gopkg.in/yaml.v3` 无法下载，可以改用 `encoding/json` + JSON 格式配置文件。此时将 `convallaria.yaml` 改为 `convallaria.json`，`gopkg.in/yaml.v3` 替换为 `encoding/json`，YAML 结构体 tag 改为 JSON tag。

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./internal/config/ -v
```
Expected: All 3 tests PASS

- [ ] **Step 6: Commit**

```bash
git add -A && git commit -m "feat: add configuration system with YAML + env overrides"
```

---

### Task 2.2: Credential management

**Files:**
- Create: `internal/credential/credential.go`
- Create: `internal/credential/credential_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/credential/credential_test.go
package credential

import (
    "testing"
)

func TestMemoryStore_SetAndGet(t *testing.T) {
    store := NewMemoryStore()
    err := store.Set("deepseek", "sk-test-key-12345")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    key, err := store.Get("deepseek")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if key != "sk-test-key-12345" {
        t.Errorf("expected 'sk-test-key-12345', got %q", key)
    }
}

func TestMemoryStore_GetMissing(t *testing.T) {
    store := NewMemoryStore()
    _, err := store.Get("nonexistent")
    if err != ErrNotFound {
        t.Errorf("expected ErrNotFound, got %v", err)
    }
}

func TestMemoryStore_List(t *testing.T) {
    store := NewMemoryStore()
    store.Set("deepseek", "sk-abc")
    store.Set("openai", "sk-xyz")

    providers := store.List()
    if len(providers) != 2 {
        t.Fatalf("expected 2 providers, got %d", len(providers))
    }
}

func TestMemoryStore_Delete(t *testing.T) {
    store := NewMemoryStore()
    store.Set("deepseek", "sk-abc")
    err := store.Delete("deepseek")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    _, err = store.Get("deepseek")
    if err != ErrNotFound {
        t.Errorf("expected ErrNotFound after delete, got %v", err)
    }
}

func TestMaskKey(t *testing.T) {
    masked := MaskKey("sk-abcdefghijklmnop12345")
    if masked != "sk-****2345" {
        t.Errorf("expected 'sk-****2345', got %q", masked)
    }
}

func TestMaskKey_Short(t *testing.T) {
    masked := MaskKey("abc")
    if masked != "***" {
        t.Errorf("expected '***', got %q", masked)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/credential/ -v
```
Expected: FAIL

- [ ] **Step 3: Write the credential implementation**

```go
// internal/credential/credential.go
package credential

import (
    "errors"
    "sync"
)

var ErrNotFound = errors.New("credential not found")

// Store is the interface for secure credential storage.
type Store interface {
    Set(provider string, key string) error
    Get(provider string) (string, error)
    List() []string
    Delete(provider string) error
}

// MemoryStore is an in-memory credential store for testing.
// In production, this is replaced by OS keychain.
type MemoryStore struct {
    mu    sync.RWMutex
    keys  map[string]string
}

func NewMemoryStore() *MemoryStore {
    return &MemoryStore{keys: make(map[string]string)}
}

func (s *MemoryStore) Set(provider string, key string) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.keys[provider] = key
    return nil
}

func (s *MemoryStore) Get(provider string) (string, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    key, ok := s.keys[provider]
    if !ok {
        return "", ErrNotFound
    }
    return key, nil
}

func (s *MemoryStore) List() []string {
    s.mu.RLock()
    defer s.mu.RUnlock()
    providers := make([]string, 0, len(s.keys))
    for p := range s.keys {
        providers = append(providers, p)
    }
    return providers
}

func (s *MemoryStore) Delete(provider string) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    delete(s.keys, provider)
    return nil
}

// MaskKey returns a masked version of an API key showing only the last 4 chars.
func MaskKey(key string) string {
    if len(key) <= 8 {
        return "***"
    }
    // key[:3] includes the dash (e.g. "sk-"), so don't add another dash
    return key[:3] + "****" + key[len(key)-4:]
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/credential/ -v
```
Expected: All 6 tests PASS

- [x] **Step 5: Commit** ✅

```bash
git add -A && git commit -m "feat: add credential management with in-memory store"
```

---

## Phase 3: Parser & Tool System

### Task 3.1: Action parser

**Files:**
- Create: `internal/parser/parser.go`
- Create: `internal/parser/parser_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/parser/parser_test.go
package parser

import (
    "testing"

    "github.com/Convallariaxhr/convallaria/internal/llm"
)

func TestParse_PureText_NoToolCalls(t *testing.T) {
    resp := &llm.Response{
        Text:       "Hello, how can I help you?",
        StopReason: "stop",
    }
    actions := Parse(resp)
    if len(actions) != 0 {
        t.Errorf("expected 0 actions, got %d", len(actions))
    }
    if !actions.IsStop() {
        t.Error("expected IsStop() to be true for pure text")
    }
}

func TestParse_SingleToolCall(t *testing.T) {
    resp := &llm.Response{
        ToolCalls: []llm.ToolCall{
            {
                ID: "call_1",
                Function: llm.FunctionCall{
                    Name:      "file_read",
                    Arguments: `{"path":"main.go"}`,
                },
            },
        },
        StopReason: "tool_calls",
    }
    actions := Parse(resp)
    if len(actions) != 1 {
        t.Fatalf("expected 1 action, got %d", len(actions))
    }
    if actions[0].Tool != "file_read" {
        t.Errorf("expected tool 'file_read', got %q", actions[0].Tool)
    }
    if actions[0].Params["path"] != "main.go" {
        t.Errorf("expected path 'main.go', got %v", actions[0].Params["path"])
    }
}

func TestParse_MultipleToolCalls(t *testing.T) {
    resp := &llm.Response{
        ToolCalls: []llm.ToolCall{
            {ID: "call_1", Function: llm.FunctionCall{Name: "file_read", Arguments: `{"path":"a.go"}`}},
            {ID: "call_2", Function: llm.FunctionCall{Name: "file_write", Arguments: `{"path":"b.go","content":"x"}`}},
        },
        StopReason: "tool_calls",
    }
    actions := Parse(resp)
    if len(actions) != 2 {
        t.Fatalf("expected 2 actions, got %d", len(actions))
    }
}

func TestParse_MalformedJSON(t *testing.T) {
    resp := &llm.Response{
        ToolCalls: []llm.ToolCall{
            {ID: "call_1", Function: llm.FunctionCall{Name: "shell_run", Arguments: `{not json}`}},
        },
        StopReason: "tool_calls",
    }
    actions := Parse(resp)
    if len(actions) != 1 {
        t.Fatalf("expected 1 action even with malformed JSON, got %d", len(actions))
    }
    if actions[0].ParseError == nil {
        t.Error("expected ParseError for malformed JSON")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/parser/ -v
```
Expected: FAIL

- [ ] **Step 3: Write the parser implementation**

```go
// internal/parser/parser.go
package parser

import (
    "encoding/json"

    "github.com/Convallariaxhr/convallaria/internal/llm"
)

// Action represents a parsed tool call that the agent should execute.
type Action struct {
    ToolCallID string
    Tool       string
    Params     map[string]any
    ParseError error // non-nil if arguments JSON was malformed
}

// ActionList is a list of actions with helper methods.
type ActionList []Action

// IsStop returns true if the LLM indicated it's done (no tool calls).
func (al ActionList) IsStop() bool {
    return len(al) == 0
}

// Parse extracts actions from an LLM response.
func Parse(resp *llm.Response) ActionList {
    actions := make([]Action, 0, len(resp.ToolCalls))
    for _, tc := range resp.ToolCalls {
        action := Action{
            ToolCallID: tc.ID,
            Tool:       tc.Function.Name,
        }
        var params map[string]any
        if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
            action.ParseError = err
            action.Params = map[string]any{"_raw": tc.Function.Arguments}
        } else {
            action.Params = params
        }
        actions = append(actions, action)
    }
    return ActionList(actions)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/parser/ -v
```
Expected: All 4 tests PASS

- [x] **Step 5: Commit** ✅

```bash
git add -A && git commit -m "feat: add action parser for LLM response parsing"
```

---

### Task 3.2: Tool registry and individual tools

**Files:**
- Create: `internal/tools/registry.go`
- Create: `internal/tools/registry_test.go`
- Create: `internal/tools/file_reader.go`
- Create: `internal/tools/file_writer.go`
- Create: `internal/tools/shell_runner.go`
- Create: `internal/tools/searcher.go`
- Create: `internal/tools/test_runner.go`
- Create: `internal/tools/git_ops.go`

- [ ] **Step 1: Write the failing test for registry**

```go
// internal/tools/registry_test.go
package tools

import (
    "context"
    "testing"
)

func TestRegistry_RegisterAndExecute(t *testing.T) {
    reg := NewRegistry()
    reg.Register("echo", &EchoTool{})

    result, err := reg.Execute(context.Background(), "echo", map[string]any{"message": "hello"})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result.Output != "hello" {
        t.Errorf("expected 'hello', got %q", result.Output)
    }
}

func TestRegistry_UnknownTool(t *testing.T) {
    reg := NewRegistry()
    _, err := reg.Execute(context.Background(), "nonexistent", nil)
    if err != ErrUnknownTool {
        t.Errorf("expected ErrUnknownTool, got %v", err)
    }
}

func TestRegistry_ListTools(t *testing.T) {
    reg := NewRegistry()
    reg.Register("tool_a", &EchoTool{})
    reg.Register("tool_b", &EchoTool{})

    tools := reg.List()
    if len(tools) != 2 {
        t.Errorf("expected 2 tools, got %d", len(tools))
    }
}

// EchoTool is a simple test tool.
type EchoTool struct{}

func (e *EchoTool) Execute(ctx context.Context, params map[string]any) (*Result, error) {
    msg, _ := params["message"].(string)
    return &Result{Output: msg, Success: true}, nil
}

func (e *EchoTool) Name() string        { return "echo" }
func (e *EchoTool) Description() string { return "Echoes the message parameter" }
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/tools/ -v
```
Expected: FAIL

- [ ] **Step 3: Write the registry + tool interface + all tools**

```go
// internal/tools/registry.go
package tools

import (
    "context"
    "errors"
    "sync"
)

var ErrUnknownTool = errors.New("unknown tool")

// Result is the output of a tool execution.
type Result struct {
    Output  string `json:"output"`
    Success bool   `json:"success"`
    Error   string `json:"error,omitempty"`
}

// Tool defines the interface for all executable tools.
type Tool interface {
    Name() string
    Description() string
    Execute(ctx context.Context, params map[string]any) (*Result, error)
}

// Registry manages tool registration and dispatch.
type Registry struct {
    mu    sync.RWMutex
    tools map[string]Tool
}

func NewRegistry() *Registry {
    return &Registry{tools: make(map[string]Tool)}
}

func (r *Registry) Register(name string, tool Tool) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.tools[name] = tool
}

func (r *Registry) Execute(ctx context.Context, name string, params map[string]any) (*Result, error) {
    r.mu.RLock()
    tool, ok := r.tools[name]
    r.mu.RUnlock()
    if !ok {
        return &Result{Success: false, Error: "unknown tool: " + name}, ErrUnknownTool
    }
    return tool.Execute(ctx, params)
}

func (r *Registry) List() []Tool {
    r.mu.RLock()
    defer r.mu.RUnlock()
    tools := make([]Tool, 0, len(r.tools))
    for _, t := range r.tools {
        tools = append(tools, t)
    }
    return tools
}
```

```go
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
```

```go
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
```

```go
// internal/tools/shell_runner.go
package tools

import (
    "context"
    "os/exec"
    "runtime"
    "time"
)

type ShellRunner struct {
    Timeout time.Duration
}

func (s *ShellRunner) Name() string        { return "shell_run" }
func (s *ShellRunner) Description() string { return "Execute a shell command" }

func (s *ShellRunner) Execute(ctx context.Context, params map[string]any) (*Result, error) {
    command, _ := params["command"].(string)
    timeout := s.Timeout
    if timeout == 0 {
        timeout = 120 * time.Second
    }

    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    var cmd *exec.Cmd
    if runtime.GOOS == "windows" {
        cmd = exec.CommandContext(ctx, "cmd", "/c", command)
    } else {
        cmd = exec.CommandContext(ctx, "sh", "-c", command)
    }

    output, err := cmd.CombinedOutput()
    if err != nil {
        return &Result{
            Output:  string(output),
            Success: false,
            Error:   err.Error(),
        }, nil
    }
    return &Result{Output: string(output), Success: true}, nil
}
```

```go
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

func (s *Searcher) Execute(ctx context.Context, params map[string]any) (*Result, error) {
    pattern, _ := params["pattern"].(string)
    searchPath, _ := params["path"].(string)
    if searchPath == "" {
        searchPath = "."
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
```

```go
// internal/tools/test_runner.go
package tools

import (
    "context"
    "os/exec"
)

type TestRunner struct{}

func (t *TestRunner) Name() string        { return "test_run" }
func (t *TestRunner) Description() string { return "Run tests in the project" }

func (t *TestRunner) Execute(ctx context.Context, params map[string]any) (*Result, error) {
    testPath, _ := params["path"].(string)
    args := []string{"test", "-json"}
    if testPath != "" {
        args = append(args, testPath)
    }
    args = append(args, "./...")

    cmd := exec.CommandContext(ctx, "go", args...)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return &Result{
            Output:  string(output),
            Success: false,
            Error:   "tests failed",
        }, nil
    }
    return &Result{Output: string(output), Success: true}, nil
}
```

```go
// internal/tools/git_ops.go
package tools

import (
    "context"
    "os/exec"
    "strings"
)

type GitOps struct{}

func (g *GitOps) Name() string        { return "git" }
func (g *GitOps) Description() string { return "Execute git operations (status, commit, branch, diff)" }

func (g *GitOps) Execute(ctx context.Context, params map[string]any) (*Result, error) {
    operation, _ := params["operation"].(string)
    args := []string{operation}

    switch operation {
    case "status":
        args = append(args, "--short")
    case "commit":
        message, _ := params["message"].(string)
        args = append(args, "-m", message)
    case "diff":
        // no extra args
    case "branch":
        // list branches
    default:
        return &Result{Success: false, Error: "unsupported git operation: " + operation}, nil
    }

    cmd := exec.CommandContext(ctx, "git", args...)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return &Result{
            Output:  string(output),
            Success: false,
            Error:   err.Error(),
        }, nil
    }
    return &Result{Output: strings.TrimSpace(string(output)), Success: true}, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/tools/ -v
```
Expected: All 3 registry tests PASS

- [x] **Step 5: Commit** ✅

```bash
git add -A && git commit -m "feat: add tool registry and 6 tool implementations"
```

---

## Phase 4: Guardrails

### Task 4.1: Guardrail implementation

**Files:**
- Create: `internal/guardrail/guardrail.go`
- Create: `internal/guardrail/guardrail_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/guardrail/guardrail_test.go
package guardrail

import (
    "testing"
)

func TestGuardrail_BlocksDangerousCommand(t *testing.T) {
    g := New(Config{
        DangerousCommands: true,
        FileScope:         true,
        GitDangerousOps:   true,
        Workspace:         "/tmp/test",
    })

    reason := g.Check("shell_run", map[string]any{"command": "rm -rf /"})
    if reason == nil {
        t.Fatal("expected block for 'rm -rf /'")
    }
    if reason.Level != "dangerous_command" {
        t.Errorf("expected level 'dangerous_command', got %q", reason.Level)
    }
}

func TestGuardrail_BlocksFileOutsideWorkspace(t *testing.T) {
    g := New(Config{
        DangerousCommands: true,
        FileScope:         true,
        GitDangerousOps:   true,
        Workspace:         "/tmp/test",
    })

    reason := g.Check("file_write", map[string]any{"path": "/etc/passwd", "content": "x"})
    if reason == nil {
        t.Fatal("expected block for writing outside workspace")
    }
    if reason.Level != "file_scope" {
        t.Errorf("expected level 'file_scope', got %q", reason.Level)
    }
}

func TestGuardrail_AllowsFileInsideWorkspace(t *testing.T) {
    g := New(Config{
        DangerousCommands: true,
        FileScope:         true,
        GitDangerousOps:   true,
        Workspace:         "/tmp/test",
    })

    reason := g.Check("file_write", map[string]any{"path": "/tmp/test/main.go", "content": "x"})
    if reason != nil {
        t.Errorf("expected no block for workspace file, got %v", reason)
    }
}

func TestGuardrail_BlocksGitForcePush(t *testing.T) {
    g := New(Config{
        DangerousCommands: true,
        FileScope:         true,
        GitDangerousOps:   true,
        Workspace:         "/tmp/test",
    })

    reason := g.Check("git", map[string]any{"operation": "push", "force": true})
    if reason == nil {
        t.Fatal("expected block for git push --force")
    }
    if reason.Level != "git_dangerous" {
        t.Errorf("expected level 'git_dangerous', got %q", reason.Level)
    }
}

func TestGuardrail_AllowsSafeCommand(t *testing.T) {
    g := New(Config{
        DangerousCommands: true,
        FileScope:         true,
        GitDangerousOps:   true,
        Workspace:         "/tmp/test",
    })

    reason := g.Check("shell_run", map[string]any{"command": "go build ./..."})
    if reason != nil {
        t.Errorf("expected no block for 'go build', got %v", reason)
    }
}

func TestGuardrail_DisabledGuardrails(t *testing.T) {
    g := New(Config{
        DangerousCommands: false,
        FileScope:         false,
        GitDangerousOps:   false,
        Workspace:         "/tmp/test",
    })

    reason := g.Check("shell_run", map[string]any{"command": "rm -rf /"})
    if reason != nil {
        t.Errorf("expected no block when guardrails disabled, got %v", reason)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/guardrail/ -v
```
Expected: FAIL

- [ ] **Step 3: Write the guardrail implementation**

```go
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
    config           Config
    dangerousPatterns []*regexp.Regexp
    gitDangerousOps  map[string]bool
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/guardrail/ -v
```
Expected: All 6 tests PASS

- [x] **Step 5: Commit** ✅

```bash
git add -A && git commit -m "feat: add guardrail with three-layer safety checks"
```

---

## Phase 5: Feedback Loop (重点维度)

### Task 5.1: Feedback validators

**Files:**
- Create: `internal/feedback/feedback.go`
- Create: `internal/feedback/build_validator.go`
- Create: `internal/feedback/vet_validator.go`
- Create: `internal/feedback/test_validator.go`
- Create: `internal/feedback/feedback_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/feedback/feedback_test.go
package feedback

import (
    "context"
    "os"
    "path/filepath"
    "testing"
)

func TestBuildValidator_ValidGoFile(t *testing.T) {
    dir := t.TempDir()
    goFile := filepath.Join(dir, "main.go")
    os.WriteFile(goFile, []byte("package main\n\nfunc main() { println(\"hello\") }\n"), 0644)

    v := &BuildValidator{}
    fb := v.Validate(context.Background(), dir)
    if fb.Status != "passed" {
        t.Errorf("expected passed, got %s: %s", fb.Status, fb.Summary)
    }
}

func TestBuildValidator_InvalidGoFile(t *testing.T) {
    dir := t.TempDir()
    goFile := filepath.Join(dir, "main.go")
    os.WriteFile(goFile, []byte("package main\n\nfunc main() {\n\tundefinedVar\n}\n"), 0644)

    v := &BuildValidator{}
    fb := v.Validate(context.Background(), dir)
    if fb.Status == "passed" {
        t.Error("expected build failure for invalid Go file")
    }
    if len(fb.Errors) == 0 {
        t.Error("expected at least one error")
    }
}

func TestFeedbackLoop_AllPass(t *testing.T) {
    dir := t.TempDir()
    os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() { println(\"ok\") }\n"), 0644)

    loop := NewLoop()
    result := loop.Run(context.Background(), dir)
    if result.Status != "passed" {
        t.Errorf("expected all pass, got %s", result.Status)
    }
}

func TestFeedbackLoop_BuildFailure(t *testing.T) {
    dir := t.TempDir()
    os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {\n\tbroken\n}\n"), 0644)

    loop := NewLoop()
    result := loop.Run(context.Background(), dir)
    if result.Status != "failed" {
        t.Errorf("expected failed, got %s", result.Status)
    }
    if result.Stage != "build" {
        t.Errorf("expected failure at build stage, got %s", result.Stage)
    }
}

func TestFeedbackToMessage(t *testing.T) {
    fb := &Feedback{
        Stage:  "build",
        Status: "failed",
        Errors: []FeedbackError{
            {File: "main.go", Line: 3, Column: 2, Message: "undefined: broken"},
        },
        Summary: "Build failed: 1 error",
    }
    msg := fb.ToMessage()
    if msg.Role != "tool" {
        t.Errorf("expected role 'tool', got %q", msg.Role)
    }
    if msg.Name != "feedback" {
        t.Errorf("expected name 'feedback', got %q", msg.Name)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/feedback/ -v
```
Expected: FAIL

- [ ] **Step 3: Write the feedback implementation**

```go
// internal/feedback/feedback.go
package feedback

import (
    "context"
    "encoding/json"

    "github.com/Convallariaxhr/convallaria/internal/llm"
)

// Feedback is the structured result of running validators.
type Feedback struct {
    Stage   string          `json:"stage"`   // build, vet, test
    Status  string          `json:"status"`  // passed, failed
    Errors  []FeedbackError `json:"errors"`
    Summary string          `json:"summary"`
}

// FeedbackError describes a single validation error.
type FeedbackError struct {
    File    string `json:"file"`
    Line    int    `json:"line"`
    Column  int    `json:"column"`
    Message string `json:"message"`
}

// Validator checks code quality and returns structured feedback.
type Validator interface {
    Validate(ctx context.Context, workspace string) *Feedback
}

// LoopResult is the aggregated result of running all validators.
type LoopResult struct {
    Status string
    Stage  string
    Errors []FeedbackError
}

// Loop runs validators in order and returns the first failure.
type Loop struct {
    validators []Validator
}

func NewLoop() *Loop {
    return &Loop{
        validators: []Validator{
            &BuildValidator{},
            &VetValidator{},
            &TestValidator{},
        },
    }
}

// Run executes all validators in sequence. Returns the first failure.
func (l *Loop) Run(ctx context.Context, workspace string) *LoopResult {
    for _, v := range l.validators {
        fb := v.Validate(ctx, workspace)
        if fb.Status == "failed" {
            return &LoopResult{
                Status: "failed",
                Stage:  fb.Stage,
                Errors: fb.Errors,
            }
        }
    }
    return &LoopResult{Status: "passed"}
}

// ToMessage converts feedback into an LLM message for context injection.
func (fb *Feedback) ToMessage() llm.Message {
    data, _ := json.Marshal(fb)
    return llm.Message{
        Role:    "tool",
        Name:    "feedback",
        Content: string(data),
    }
}
```

```go
// internal/feedback/build_validator.go
package feedback

import (
    "context"
    "os/exec"
    "regexp"
    "strconv"
    "strings"
)

type BuildValidator struct{}

func (v *BuildValidator) Validate(ctx context.Context, workspace string) *Feedback {
    fb := &Feedback{Stage: "build", Status: "passed"}

    cmd := exec.CommandContext(ctx, "go", "build", "./...")
    cmd.Dir = workspace
    output, err := cmd.CombinedOutput()

    if err == nil {
        fb.Summary = "Build passed"
        return fb
    }

    fb.Status = "failed"
    fb.Summary = "Build failed"

    // Parse Go build errors: file:line:col: message
    errStr := string(output)
    re := regexp.MustCompile(`(.+?):(\d+):(\d+):\s*(.+)`)
    for _, line := range strings.Split(errStr, "\n") {
        matches := re.FindStringSubmatch(line)
        if len(matches) == 5 {
            lineNo, _ := strconv.Atoi(matches[2])
            colNo, _ := strconv.Atoi(matches[3])
            fb.Errors = append(fb.Errors, FeedbackError{
                File:    matches[1],
                Line:    lineNo,
                Column:  colNo,
                Message: strings.TrimSpace(matches[4]),
            })
        }
    }
    return fb
}
```

```go
// internal/feedback/vet_validator.go
package feedback

import (
    "context"
    "os/exec"
    "strings"
)

type VetValidator struct{}

func (v *VetValidator) Validate(ctx context.Context, workspace string) *Feedback {
    fb := &Feedback{Stage: "vet", Status: "passed"}

    cmd := exec.CommandContext(ctx, "go", "vet", "./...")
    cmd.Dir = workspace
    output, err := cmd.CombinedOutput()

    if err == nil {
        fb.Summary = "Vet passed"
        return fb
    }

    fb.Status = "failed"
    fb.Summary = "Vet found issues"
    for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
        if line != "" {
            fb.Errors = append(fb.Errors, FeedbackError{
                Message: line,
            })
        }
    }
    return fb
}
```

```go
// internal/feedback/test_validator.go
package feedback

import (
    "context"
    "encoding/json"
    "os/exec"
)

type TestValidator struct{}

func (v *TestValidator) Validate(ctx context.Context, workspace string) *Feedback {
    fb := &Feedback{Stage: "test", Status: "passed"}

    cmd := exec.CommandContext(ctx, "go", "test", "-json", "./...")
    cmd.Dir = workspace
    output, err := cmd.CombinedOutput()

    if err == nil {
        fb.Summary = "Tests passed"
        return fb
    }

    fb.Status = "failed"
    fb.Summary = "Tests failed"

    // Parse go test -json output
    type testEvent struct {
        Action  string `json:"Action"`
        Test    string `json:"Test"`
        Package string `json:"Package"`
        Output  string `json:"Output"`
    }

    for _, line := range bytesToLines(output) {
        var ev testEvent
        if json.Unmarshal([]byte(line), &ev) == nil {
            if ev.Action == "fail" && ev.Test != "" {
                fb.Errors = append(fb.Errors, FeedbackError{
                    File:    ev.Package,
                    Message: "Test failed: " + ev.Test,
                })
            }
        }
    }
    return fb
}

func bytesToLines(data []byte) []string {
    var lines []string
    start := 0
    for i, b := range data {
        if b == '\n' {
            lines = append(lines, string(data[start:i]))
            start = i + 1
        }
    }
    if start < len(data) {
        lines = append(lines, string(data[start:]))
    }
    return lines
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/feedback/ -v
```
Expected: All 5 tests PASS

- [x] **Step 5: Commit** ✅

```bash
git add -A && git commit -m "feat: add feedback loop with Build/Vet/Test validators"
```

---

## Phase 6: Agent Main Loop

### Task 6.1: Agent main loop with mock LLM integration

**Files:**
- Create: `internal/agent/loop.go`
- Create: `internal/agent/loop_test.go`

- [ ] **Step 1: Write the failing test (the key mechanism demo)**

```go
// internal/agent/loop_test.go
package agent

import (
    "context"
    "os"
    "path/filepath"
    "testing"

    "github.com/Convallariaxhr/convallaria/internal/feedback"
    "github.com/Convallariaxhr/convallaria/internal/guardrail"
    "github.com/Convallariaxhr/convallaria/internal/llm"
    "github.com/Convallariaxhr/convallaria/internal/parser"
    "github.com/Convallariaxhr/convallaria/internal/tools"
)

func TestAgent_TextResponse_ReturnsFinalReply(t *testing.T) {
    mock := llm.NewMockProvider()
    mock.AddResponse(llm.MockTextResponse("Hello! I can help you write code."))

    agent := New(Config{
        MaxTurns: 5,
        Provider: mock,
    })

    resp, err := agent.Run(context.Background(), "Write a hello world program")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if resp != "Hello! I can help you write code." {
        t.Errorf("expected text response, got %q", resp)
    }
}

func TestAgent_ToolCall_ExecutesAndReturnsResult(t *testing.T) {
    dir := t.TempDir()
    filePath := filepath.Join(dir, "hello.txt")

    mock := llm.NewMockProvider()
    // First response: tool call to write file
    mock.AddResponse(llm.MockToolCallResponse("call_1", "file_write",
        `{"path":"`+filePath+`","content":"hello world"}`))
    // Second response: text completion
    mock.AddResponse(llm.MockTextResponse("Done! I've created hello.txt with 'hello world'."))

    agent := New(Config{
        MaxTurns:  5,
        Provider:  mock,
        Workspace: dir,
    })

    resp, err := agent.Run(context.Background(), "Create a file called hello.txt")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if resp != "Done! I've created hello.txt with 'hello world'." {
        t.Errorf("unexpected response: %q", resp)
    }
    // Verify file was actually created
    data, err := os.ReadFile(filePath)
    if err != nil {
        t.Fatalf("file not created: %v", err)
    }
    if string(data) != "hello world" {
        t.Errorf("file content mismatch: %q", string(data))
    }
}

func TestAgent_GuardrailBlocksDangerousAction(t *testing.T) {
    mock := llm.NewMockProvider()
    // LLM tries to run a dangerous command
    mock.AddResponse(llm.MockToolCallResponse("call_1", "shell_run",
        `{"command":"rm -rf /"}`))
    // After blocked, it should get a text response
    mock.AddResponse(llm.MockTextResponse("Sorry, I cannot execute that command."))

    agent := New(Config{
        MaxTurns:  5,
        Provider:  mock,
        Workspace: "/tmp/test",
    })

    resp, err := agent.Run(context.Background(), "Delete everything")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if resp == "" {
        t.Error("expected a response after guardrail block")
    }
    // The dangerous command should NOT have been executed
    // (the mock would have returned a tool result if it was)
}

func TestAgent_FeedbackLoop_DetectsBuildError(t *testing.T) {
    dir := t.TempDir()
    // Write a broken Go file
    brokenFile := filepath.Join(dir, "broken.go")
    os.WriteFile(brokenFile, []byte("package main\n\nfunc main() {\n\tundefined\n}\n"), 0644)

    mock := llm.NewMockProvider()
    // LLM writes a broken file
    mock.AddResponse(llm.MockToolCallResponse("call_1", "file_write",
        `{"path":"`+brokenFile+`","content":"package main\n\nfunc main() {\n\tundefined\n}\n"}`))
    // Feedback loop should detect build failure and inform LLM
    // LLM then tries to fix
    mock.AddResponse(llm.MockToolCallResponse("call_2", "file_write",
        `{"path":"`+brokenFile+`","content":"package main\n\nfunc main() { println(\"hello\") }\n"}`))
    // After fix, text response
    mock.AddResponse(llm.MockTextResponse("Fixed the build error!"))

    agent := New(Config{
        MaxTurns:  5,
        Provider:  mock,
        Workspace: dir,
    })

    resp, err := agent.Run(context.Background(), "Write a hello world program")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if resp == "" {
        t.Error("expected a response after feedback loop")
    }
    // Verify the feedback loop ran: the second tool call should have been triggered
    // by the build failure feedback from the first file write
    if mock.CallCount() < 2 {
        t.Error("expected at least 2 LLM calls (initial + after feedback)")
    }
}

func TestAgent_MaxTurnsExceeded(t *testing.T) {
    mock := llm.NewMockProvider()
    // Add many tool calls that will never converge
    for i := 0; i < 10; i++ {
        mock.AddResponse(llm.MockToolCallResponse("call_"+string(rune('a'+i)), "file_write",
            `{"path":"/tmp/test/x.go","content":"x"}`))
    }

    agent := New(Config{
        MaxTurns:  3,
        Provider:  mock,
        Workspace: "/tmp/test",
    })

    _, err := agent.Run(context.Background(), "Write code")
    if err != ErrMaxTurnsExceeded {
        t.Errorf("expected ErrMaxTurnsExceeded, got %v", err)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/agent/ -v
```
Expected: FAIL

- [ ] **Step 3: Write the agent main loop implementation**

```go
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
    MaxTurns  int
    Provider  llm.Provider
    Workspace string
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/agent/ -v
```
Expected: All 5 tests PASS

- [x] **Step 5: Commit** ✅

```bash
git add -A && git commit -m "feat: add agent main loop with mock-LLM integration tests"
```

---

## Phase 7: Context Window & Error Recovery

### Task 7.1: Context window manager

**Files:**
- Create: `internal/context/manager.go`
- Create: `internal/context/manager_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/context/manager_test.go
package context

import (
    "testing"

    "github.com/Convallariaxhr/convallaria/internal/llm"
)

func TestManager_EstimateTokens(t *testing.T) {
    mgr := New(Config{MaxTokens: 64000, Threshold: 0.8})

    msgs := []llm.Message{
        {Role: "system", Content: "You are a coding agent."},
        {Role: "user", Content: "Write a hello world program."},
    }
    tokens := mgr.EstimateTokens(msgs)
    if tokens <= 0 {
        t.Errorf("expected positive token count, got %d", tokens)
    }
    // Rough estimate: ~20 tokens for these messages
    if tokens < 10 || tokens > 100 {
        t.Errorf("expected ~10-100 tokens, got %d", tokens)
    }
}

func TestManager_NeedsCompression(t *testing.T) {
    mgr := New(Config{MaxTokens: 64000, Threshold: 0.8})

    // Small messages should not need compression
    msgs := []llm.Message{
        {Role: "user", Content: "Hello"},
    }
    if mgr.NeedsCompression(msgs) {
        t.Error("expected no compression needed for small messages")
    }
}

func TestManager_Compress(t *testing.T) {
    mgr := New(Config{MaxTokens: 64000, Threshold: 0.8})

    // Create many messages to simulate long conversation
    msgs := []llm.Message{
        {Role: "system", Content: "System prompt"},
        {Role: "user", Content: "Message 1"},
        {Role: "assistant", Content: "Reply 1"},
        {Role: "user", Content: "Message 2"},
        {Role: "assistant", Content: "Reply 2"},
        {Role: "user", Content: "Message 3"},
    }

    compressed := mgr.Compress(msgs, 3) // Keep last 3
    // Should keep system prompt + last 3 messages
    if len(compressed) < 4 || len(compressed) > 6 {
        t.Errorf("expected 4-6 messages after compression, got %d", len(compressed))
    }
    // First message should still be system prompt
    if compressed[0].Role != "system" {
        t.Error("expected system prompt to be preserved")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/context/ -v
```
Expected: FAIL

- [ ] **Step 3: Write the implementation**

```go
// internal/context/manager.go
package context

import (
    "fmt"

    "github.com/Convallariaxhr/convallaria/internal/llm"
)

// Config configures context window management.
type Config struct {
    MaxTokens int
    Threshold float64
}

// Manager handles context window token counting and compression.
type Manager struct {
    config Config
}

func New(config Config) *Manager {
    if config.MaxTokens == 0 {
        config.MaxTokens = 64000
    }
    if config.Threshold == 0 {
        config.Threshold = 0.8
    }
    return &Manager{config: config}
}

// EstimateTokens provides a rough estimate of token count for messages.
// Uses a simple heuristic: ~4 characters per token for English text.
func (m *Manager) EstimateTokens(messages []llm.Message) int {
    total := 0
    for _, msg := range messages {
        total += len(msg.Role) / 4
        total += len(msg.Content) / 4
        for _, tc := range msg.ToolCalls {
            total += len(tc.Function.Name) / 4
            total += len(tc.Function.Arguments) / 4
        }
    }
    if total < 1 {
        total = 1
    }
    return total
}

// NeedsCompression returns true if the token count exceeds the threshold.
func (m *Manager) NeedsCompression(messages []llm.Message) bool {
    tokens := m.EstimateTokens(messages)
    limit := int(float64(m.config.MaxTokens) * m.config.Threshold)
    return tokens > limit
}

// Compress keeps the system prompt and the last N messages, replacing older messages
// with a summary placeholder. This is a simple truncation-based approach.
// For production, the summary would be generated by the LLM.
func (m *Manager) Compress(messages []llm.Message, keepLast int) []llm.Message {
    if len(messages) <= keepLast+1 {
        return messages
    }

    var result []llm.Message

    // Always keep the system prompt
    if len(messages) > 0 && messages[0].Role == "system" {
        result = append(result, messages[0])
        messages = messages[1:]
    }

    // Add a summary placeholder for truncated messages
    truncated := len(messages) - keepLast
    if truncated > 0 {
        result = append(result, llm.Message{
            Role:    "system",
            Content: fmt.Sprintf("[Earlier conversation summarized: %d messages omitted]", truncated),
        })
    }

    // Keep the last N messages
    start := len(messages) - keepLast
    if start < 0 {
        start = 0
    }
    result = append(result, messages[start:]...)

    return result
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/context/ -v
```
Expected: All 3 tests PASS

- [x] **Step 5: Commit** ✅

```bash
git add -A && git commit -m "feat: add context window manager with token estimation and compression"
```

---

### Task 7.2: Error recovery

**Files:**
- Create: `internal/recovery/recovery.go`
- Create: `internal/recovery/recovery_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/recovery/recovery_test.go
package recovery

import (
    "context"
    "errors"
    "testing"

    "github.com/Convallariaxhr/convallaria/internal/llm"
)

func TestRecovery_RetryOnParseError(t *testing.T) {
    mock := llm.NewMockProvider()
    // First response: malformed
    mock.AddResponse(&llm.Response{
        ToolCalls: []llm.ToolCall{
            {ID: "call_1", Function: llm.FunctionCall{Name: "shell_run", Arguments: `{bad json}`}},
        },
        StopReason: "tool_calls",
    })
    // Second response: corrected
    mock.AddResponse(llm.MockTextResponse("OK, fixed."))

    cfg := Config{MaxRetries: 3, MaxCorrections: 2}
    result, err := WithRecovery(context.Background(), mock, cfg, func(resp *llm.Response) (string, error) {
        if resp.ToolCalls[0].Function.Arguments == `{bad json}` {
            return "", errors.New("parse error")
        }
        return "success", nil
    })

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result != "success" {
        t.Errorf("expected 'success', got %q", result)
    }
    if mock.CallCount() < 2 {
        t.Error("expected at least 2 LLM calls (retry)")
    }
}

func TestRecovery_DegradeOnMaxRetries(t *testing.T) {
    mock := llm.NewMockProvider()
    for i := 0; i < 5; i++ {
        mock.AddResponse(&llm.Response{
            ToolCalls: []llm.ToolCall{
                {ID: "call_1", Function: llm.FunctionCall{Name: "shell_run", Arguments: `{bad}`}},
            },
            StopReason: "tool_calls",
        })
    }

    cfg := Config{MaxRetries: 2, MaxCorrections: 1}
    _, err := WithRecovery(context.Background(), mock, cfg, func(resp *llm.Response) (string, error) {
        return "", errors.New("parse error")
    })

    if err != ErrMaxRetriesExceeded {
        t.Errorf("expected ErrMaxRetriesExceeded, got %v", err)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/recovery/ -v
```
Expected: FAIL

- [ ] **Step 3: Write the implementation**

```go
// internal/recovery/recovery.go
package recovery

import (
    "context"
    "errors"
    "fmt"

    "github.com/Convallariaxhr/convallaria/internal/llm"
)

var ErrMaxRetriesExceeded = errors.New("max retries exceeded")

// Config configures error recovery behavior.
type Config struct {
    MaxRetries     int
    MaxCorrections int
}

// Handler is a function that processes an LLM response and returns a result.
type Handler func(resp *llm.Response) (string, error)

// WithRecovery wraps a handler with retry and correction logic.
func WithRecovery(ctx context.Context, provider llm.Provider, cfg Config, handler Handler) (string, error) {
    var messages []llm.Message

    for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
        resp, err := provider.ChatSync(ctx, messages)
        if err != nil {
            if attempt >= cfg.MaxRetries {
                return "", fmt.Errorf("llm error after %d retries: %w", attempt, err)
            }
            messages = append(messages, llm.Message{
                Role:    "system",
                Content: fmt.Sprintf("Error: %v. Please try again with a valid response.", err),
            })
            continue
        }

        result, err := handler(resp)
        if err == nil {
            return result, nil
        }

        if attempt >= cfg.MaxRetries {
            return "", ErrMaxRetriesExceeded
        }

        // Inject error feedback and retry
        messages = append(messages,
            llm.Message{Role: "assistant", Content: fmt.Sprintf("%v", resp)},
            llm.Message{Role: "system", Content: fmt.Sprintf("Error processing your response: %v. Please correct and retry.", err)},
        )
    }

    return "", ErrMaxRetriesExceeded
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/recovery/ -v
```
Expected: All 2 tests PASS

- [x] **Step 5: Commit** ✅

```bash
git add -A && git commit -m "feat: add error recovery with retry, correction, and degradation"
```

---

## Phase 8: Memory System

### Task 8.1: Memory store with SQLite

**Files:**
- Create: `internal/memory/rules.go`
- Create: `internal/memory/store.go`
- Create: `internal/memory/embedder.go`
- Create: `internal/memory/memory_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/memory/memory_test.go
package memory

import (
    "os"
    "path/filepath"
    "testing"
)

func TestRulesLoader_LoadsCONVALLARIAMD(t *testing.T) {
    dir := t.TempDir()
    os.WriteFile(filepath.Join(dir, "CONVALLARIA.md"), []byte("# Project Rules\n\n- Use tabs for indentation"), 0644)

    rules, err := LoadRules(dir)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if rules == "" {
        t.Error("expected non-empty rules")
    }
    if !contains(rules, "Use tabs for indentation") {
        t.Errorf("expected rules to contain 'Use tabs for indentation', got %q", rules)
    }
}

func TestRulesLoader_NoFile(t *testing.T) {
    dir := t.TempDir()
    rules, err := LoadRules(dir)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if rules != "" {
        t.Errorf("expected empty rules, got %q", rules)
    }
}

func TestMemoryStore_InsertAndSearch(t *testing.T) {
    store := NewStore()
    err := store.Insert(MemoryEntry{
        Content:  "We decided to use Go modules for dependency management",
        Category: "decision",
    })
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    results, err := store.Search("Go dependency management", 5)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(results) == 0 {
        t.Error("expected at least one search result")
    }
}

func TestMemoryStore_SearchEmpty(t *testing.T) {
    store := NewStore()
    results, err := store.Search("nonexistent query", 5)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(results) != 0 {
        t.Errorf("expected 0 results, got %d", len(results))
    }
}

func contains(s, substr string) bool {
    return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
    for i := 0; i <= len(s)-len(substr); i++ {
        if s[i:i+len(substr)] == substr {
            return true
        }
    }
    return false
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/memory/ -v
```
Expected: FAIL

- [ ] **Step 3: Write the implementation**

```go
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
```

```go
// internal/memory/store.go
package memory

import (
    "fmt"
    "strings"
    "sync"
)

// MemoryEntry represents a stored memory.
type MemoryEntry struct {
    ID        string
    Content   string
    Category  string
    FilePath  string
    CreatedAt int64
}

// Store is an in-memory memory store with keyword-based search.
// In production, this is replaced by SQLite + vector search.
type Store struct {
    mu      sync.RWMutex
    entries []MemoryEntry
    nextID  int
}

func NewStore() *Store {
    return &Store{}
}

// Insert adds a new memory entry.
func (s *Store) Insert(entry MemoryEntry) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.nextID++
    entry.ID = fmt.Sprintf("mem_%d", s.nextID)
    s.entries = append(s.entries, entry)
    return nil
}

// Search performs a simple keyword-based search over stored memories.
// In production, this would use embedding vectors + cosine similarity.
func (s *Store) Search(query string, topK int) ([]MemoryEntry, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    query = strings.ToLower(query)
    queryWords := strings.Fields(query)

    type scored struct {
        entry MemoryEntry
        score int
    }
    var scored []scored

    for _, entry := range s.entries {
        content := strings.ToLower(entry.Content)
        score := 0
        for _, word := range queryWords {
            if strings.Contains(content, word) {
                score++
            }
        }
        if score > 0 {
            scored = append(scored, scored{entry: entry, score: score})
        }
    }

    // Simple sort by score descending
    for i := 0; i < len(scored); i++ {
        for j := i + 1; j < len(scored); j++ {
            if scored[j].score > scored[i].score {
                scored[i], scored[j] = scored[j], scored[i]
            }
        }
    }

    if len(scored) > topK {
        scored = scored[:topK]
    }

    results := make([]MemoryEntry, len(scored))
    for i, s := range scored {
        results[i] = s.entry
    }
    return results, nil
}
```

```go
// internal/memory/embedder.go
package memory

// Embedder is a placeholder for the embedding model integration.
// In production, this would use a local model (e.g., all-MiniLM-L6-v2 via ONNX)
// or a remote embedding API.
type Embedder struct{}

// Embed converts text to a vector. Placeholder: returns nil.
func (e *Embedder) Embed(text string) ([]float32, error) {
    // Placeholder: in production, run inference with a local model
    return nil, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/memory/ -v
```
Expected: All 4 tests PASS

- [x] **Step 5: Commit** ✅

```bash
git add -A && git commit -m "feat: add memory system with rules loader and keyword search store"
```

---

## Phase 9: Session Management

### Task 9.1: Session manager with SQLite

**Files:**
- Create: `internal/session/manager.go`
- Create: `internal/session/manager_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/session/manager_test.go
package session

import (
    "testing"

    "github.com/Convallariaxhr/convallaria/internal/llm"
)

func TestManager_CreateAndGetSession(t *testing.T) {
    mgr := NewManager()
    sess, err := mgr.Create("Test session", "/tmp/test")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if sess.Title != "Test session" {
        t.Errorf("expected title 'Test session', got %q", sess.Title)
    }

    got, err := mgr.Get(sess.ID)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if got.ID != sess.ID {
        t.Errorf("expected session ID %q, got %q", sess.ID, got.ID)
    }
}

func TestManager_ListSessions(t *testing.T) {
    mgr := NewManager()
    mgr.Create("Session A", "/tmp/a")
    mgr.Create("Session B", "/tmp/b")

    sessions := mgr.List()
    if len(sessions) != 2 {
        t.Errorf("expected 2 sessions, got %d", len(sessions))
    }
}

func TestManager_DeleteSession(t *testing.T) {
    mgr := NewManager()
    sess, _ := mgr.Create("To delete", "/tmp/test")

    err := mgr.Delete(sess.ID)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    _, err = mgr.Get(sess.ID)
    if err != ErrSessionNotFound {
        t.Errorf("expected ErrSessionNotFound, got %v", err)
    }
}

func TestManager_AddMessage(t *testing.T) {
    mgr := NewManager()
    sess, _ := mgr.Create("Test", "/tmp/test")

    err := mgr.AddMessage(sess.ID, llm.Message{Role: "user", Content: "Hello"})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    err = mgr.AddMessage(sess.ID, llm.Message{Role: "assistant", Content: "Hi!"})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    msgs, err := mgr.GetMessages(sess.ID)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(msgs) != 2 {
        t.Errorf("expected 2 messages, got %d", len(msgs))
    }
}

func TestManager_ExportSession(t *testing.T) {
    mgr := NewManager()
    sess, _ := mgr.Create("Export test", "/tmp/test")
    mgr.AddMessage(sess.ID, llm.Message{Role: "user", Content: "Hello"})
    mgr.AddMessage(sess.ID, llm.Message{Role: "assistant", Content: "World"})

    exported, err := mgr.Export(sess.ID)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(exported) == 0 {
        t.Error("expected non-empty export")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/session/ -v
```
Expected: FAIL

- [ ] **Step 3: Write the implementation**

```go
// internal/session/manager.go
package session

import (
    "errors"
    "fmt"
    "sync"
    "time"

    "github.com/Convallariaxhr/convallaria/internal/llm"
)

var ErrSessionNotFound = errors.New("session not found")

// Session represents a chat session.
type Session struct {
    ID         string
    Title      string
    ProjectDir string
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

// Manager manages sessions and their messages in memory.
// In production, this is backed by SQLite.
type Manager struct {
    mu       sync.RWMutex
    sessions map[string]*Session
    messages map[string][]llm.Message
    nextID   int
}

func NewManager() *Manager {
    return &Manager{
        sessions: make(map[string]*Session),
        messages: make(map[string][]llm.Message),
    }
}

func (m *Manager) Create(title, projectDir string) (*Session, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.nextID++
    sess := &Session{
        ID:         fmt.Sprintf("sess_%d", m.nextID),
        Title:      title,
        ProjectDir: projectDir,
        CreatedAt:  time.Now(),
        UpdatedAt:  time.Now(),
    }
    m.sessions[sess.ID] = sess
    m.messages[sess.ID] = make([]llm.Message, 0)
    return sess, nil
}

func (m *Manager) Get(id string) (*Session, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    sess, ok := m.sessions[id]
    if !ok {
        return nil, ErrSessionNotFound
    }
    return sess, nil
}

func (m *Manager) List() []*Session {
    m.mu.RLock()
    defer m.mu.RUnlock()
    sessions := make([]*Session, 0, len(m.sessions))
    for _, s := range m.sessions {
        sessions = append(sessions, s)
    }
    return sessions
}

func (m *Manager) Delete(id string) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    if _, ok := m.sessions[id]; !ok {
        return ErrSessionNotFound
    }
    delete(m.sessions, id)
    delete(m.messages, id)
    return nil
}

func (m *Manager) AddMessage(sessionID string, msg llm.Message) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    if _, ok := m.sessions[sessionID]; !ok {
        return ErrSessionNotFound
    }
    m.messages[sessionID] = append(m.messages[sessionID], msg)
    m.sessions[sessionID].UpdatedAt = time.Now()
    return nil
}

func (m *Manager) GetMessages(sessionID string) ([]llm.Message, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    if _, ok := m.sessions[sessionID]; !ok {
        return nil, ErrSessionNotFound
    }
    msgs := make([]llm.Message, len(m.messages[sessionID]))
    copy(msgs, m.messages[sessionID])
    return msgs, nil
}

// Export returns the session's messages as a formatted string.
func (m *Manager) Export(sessionID string) (string, error) {
    msgs, err := m.GetMessages(sessionID)
    if err != nil {
        return "", err
    }
    var result string
    for _, msg := range msgs {
        result += fmt.Sprintf("## %s\n\n%s\n\n", msg.Role, msg.Content)
    }
    return result, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/session/ -v
```
Expected: All 5 tests PASS

- [x] **Step 5: Commit** ✅

```bash
git add -A && git commit -m "feat: add session manager with create, list, delete, and export"
```

---

## Phase 10: HTTP/SSE Server

### Task 10.1: Server with SSE streaming

**Files:**
- Create: `internal/server/handler.go`
- Create: `internal/server/sse.go`
- Create: `internal/server/server_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/server/server_test.go
package server

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
    "time"

    "github.com/Convallariaxhr/convallaria/internal/agent"
    "github.com/Convallariaxhr/convallaria/internal/llm"
    "github.com/Convallariaxhr/convallaria/internal/session"
)

func TestServer_ChatEndpoint_SSE(t *testing.T) {
    mock := llm.NewMockProvider()
    mock.AddResponse(llm.MockTextResponse("Hello! I'm Convallaria."))

    ag := agent.New(agent.Config{
        MaxTurns:     5,
        Provider:     mock,
        Workspace:    "/tmp/test",
        SystemPrompt: "You are a helpful coding agent.",
    })

    sessMgr := session.NewManager()
    srv := New(Config{Port: 0}, ag, sessMgr)

    body := `{"session_id":"","message":"Hello"}`
    req := httptest.NewRequest("POST", "/api/chat", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    srv.mux.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
    }
    // Verify SSE content type
    ct := w.Header().Get("Content-Type")
    if ct != "text/event-stream" {
        t.Errorf("expected text/event-stream, got %q", ct)
    }
}

func TestServer_SessionListEndpoint(t *testing.T) {
    sessMgr := session.NewManager()
    sessMgr.Create("Test session", "/tmp/test")

    srv := New(Config{Port: 0}, nil, sessMgr)

    req := httptest.NewRequest("GET", "/api/sessions", nil)
    w := httptest.NewRecorder()
    srv.mux.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", w.Code)
    }
    var sessions []map[string]any
    json.Unmarshal(w.Body.Bytes(), &sessions)
    if len(sessions) != 1 {
        t.Errorf("expected 1 session, got %d", len(sessions))
    }
}

func TestSSEWriter_WriteEvent(t *testing.T) {
    w := httptest.NewRecorder()
    sse := NewSSEWriter(w)

    sse.WriteEvent("token", `{"token":"H"}`)
    sse.WriteEvent("done", `{}`)

    body := w.Body.String()
    if !strings.Contains(body, "event: token") {
        t.Error("expected 'event: token' in SSE output")
    }
    if !strings.Contains(body, "event: done") {
        t.Error("expected 'event: done' in SSE output")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/server/ -v
```
Expected: FAIL

- [ ] **Step 3: Write the implementation**

```go
// internal/server/sse.go
package server

import (
    "fmt"
    "net/http"
)

// SSEWriter writes Server-Sent Events to an http.ResponseWriter.
type SSEWriter struct {
    w       http.ResponseWriter
    flusher http.Flusher
}

// NewSSEWriter creates a new SSE writer and sends the initial headers.
func NewSSEWriter(w http.ResponseWriter) *SSEWriter {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*")

    flusher, ok := w.(http.Flusher)
    if !ok {
        panic("http.ResponseWriter does not implement http.Flusher")
    }

    return &SSEWriter{w: w, flusher: flusher}
}

// WriteEvent writes an SSE event with the given type and data.
func (s *SSEWriter) WriteEvent(eventType, data string) {
    fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", eventType, data)
    s.flusher.Flush()
}
```

```go
// internal/server/handler.go
package server

import (
    "encoding/json"
    "fmt"
    "net/http"
    "strings"

    "github.com/Convallariaxhr/convallaria/internal/agent"
    "github.com/Convallariaxhr/convallaria/internal/session"
)

// Config configures the HTTP server.
type Config struct {
    Port     int
    StaticDir string
}

// Server is the HTTP/SSE server for Convallaria.
type Server struct {
    config  Config
    agent   *agent.Agent
    sessions *session.Manager
    mux     *http.ServeMux
}

// New creates a new Server.
func New(config Config, ag *agent.Agent, sessMgr *session.Manager) *Server {
    s := &Server{
        config:   config,
        agent:    ag,
        sessions: sessMgr,
        mux:      http.NewServeMux(),
    }
    s.routes()
    return s
}

func (s *Server) routes() {
    s.mux.HandleFunc("/api/chat", s.handleChat)
    s.mux.HandleFunc("/api/sessions", s.handleSessions)
    s.mux.HandleFunc("/api/sessions/", s.handleSessionByID)
    if s.config.StaticDir != "" {
        s.mux.Handle("/", http.FileServer(http.Dir(s.config.StaticDir)))
    }
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    s.mux.ServeHTTP(w, r)
}

// Start begins listening on the configured port.
func (s *Server) Start() error {
    addr := fmt.Sprintf(":%d", s.config.Port)
    if s.config.Port == 0 {
        addr = ":8080"
    }
    return http.ListenAndServe(addr, s.mux)
}

type chatRequest struct {
    SessionID string `json:"session_id"`
    Message   string `json:"message"`
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req chatRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Create or get session
    sessID := req.SessionID
    if sessID == "" {
        sess, err := s.sessions.Create("New Chat", "")
        if err != nil {
            http.Error(w, "Failed to create session", http.StatusInternalServerError)
            return
        }
        sessID = sess.ID
    }

    // Save user message
    s.sessions.AddMessage(sessID, llmMessage("user", req.Message))

    // Set up SSE
    sse := NewSSEWriter(w)

    // Send session ID
    sse.WriteEvent("session", fmt.Sprintf(`{"id":"%s"}`, sessID))

    // Run agent (in production, this would stream via the Provider's Chat method)
    resp, err := s.agent.Run(r.Context(), req.Message)
    if err != nil {
        sse.WriteEvent("error", fmt.Sprintf(`{"message":"%s"}`, err.Error()))
        return
    }

    // Simulate streaming by sending tokens
    for _, r := range resp {
        sse.WriteEvent("token", fmt.Sprintf(`{"token":"%s"}`, string(r)))
    }
    sse.WriteEvent("done", `{}`)

    // Save assistant message
    s.sessions.AddMessage(sessID, llmMessage("assistant", resp))
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    sessions := s.sessions.List()
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(sessions)
}

func (s *Server) handleSessionByID(w http.ResponseWriter, r *http.Request) {
    id := strings.TrimPrefix(r.URL.Path, "/api/sessions/")

    switch r.Method {
    case http.MethodGet:
        msgs, err := s.sessions.GetMessages(id)
        if err != nil {
            http.Error(w, "Session not found", http.StatusNotFound)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(msgs)

    case http.MethodDelete:
        if err := s.sessions.Delete(id); err != nil {
            http.Error(w, "Session not found", http.StatusNotFound)
            return
        }
        w.WriteHeader(http.StatusNoContent)

    default:
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}

func llmMessage(role, content string) llmMessage2 {
    return llmMessage2{Role: role, Content: content}
}

type llmMessage2 struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}
```

Wait, there's a naming conflict. Let me fix the handler to use the correct llm.Message type.

Actually, let me simplify the handler to avoid the import cycle. The session manager stores `llm.Message`, but the server test uses `session.NewManager()`. Let me fix the handler.

```go
// internal/server/handler.go (corrected)
package server

import (
    "encoding/json"
    "fmt"
    "net/http"
    "strings"

    "github.com/Convallariaxhr/convallaria/internal/agent"
    "github.com/Convallariaxhr/convallaria/internal/llm"
    "github.com/Convallariaxhr/convallaria/internal/session"
)

// Config configures the HTTP server.
type Config struct {
    Port      int
    StaticDir string
}

// Server is the HTTP/SSE server for Convallaria.
type Server struct {
    config   Config
    agent    *agent.Agent
    sessions *session.Manager
    mux      *http.ServeMux
}

// New creates a new Server.
func New(config Config, ag *agent.Agent, sessMgr *session.Manager) *Server {
    s := &Server{
        config:   config,
        agent:    ag,
        sessions: sessMgr,
        mux:      http.NewServeMux(),
    }
    s.routes()
    return s
}

func (s *Server) routes() {
    s.mux.HandleFunc("/api/chat", s.handleChat)
    s.mux.HandleFunc("/api/sessions", s.handleSessions)
    s.mux.HandleFunc("/api/sessions/", s.handleSessionByID)
    if s.config.StaticDir != "" {
        s.mux.Handle("/", http.FileServer(http.Dir(s.config.StaticDir)))
    }
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    s.mux.ServeHTTP(w, r)
}

// Start begins listening on the configured port.
func (s *Server) Start() error {
    addr := fmt.Sprintf(":%d", s.config.Port)
    if s.config.Port == 0 {
        addr = ":8080"
    }
    return http.ListenAndServe(addr, s.mux)
}

type chatRequest struct {
    SessionID string `json:"session_id"`
    Message   string `json:"message"`
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req chatRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    sessID := req.SessionID
    if sessID == "" {
        sess, err := s.sessions.Create("New Chat", "")
        if err != nil {
            http.Error(w, "Failed to create session", http.StatusInternalServerError)
            return
        }
        sessID = sess.ID
    }

    s.sessions.AddMessage(sessID, llm.Message{Role: "user", Content: req.Message})

    sse := NewSSEWriter(w)
    sse.WriteEvent("session", fmt.Sprintf(`{"id":"%s"}`, sessID))

    resp, err := s.agent.Run(r.Context(), req.Message)
    if err != nil {
        sse.WriteEvent("error", fmt.Sprintf(`{"message":"%s"}`, err.Error()))
        return
    }

    for _, r := range resp {
        sse.WriteEvent("token", fmt.Sprintf(`{"token":"%s"}`, string(r)))
    }
    sse.WriteEvent("done", `{}`)

    s.sessions.AddMessage(sessID, llm.Message{Role: "assistant", Content: resp})
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    sessions := s.sessions.List()
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(sessions)
}

func (s *Server) handleSessionByID(w http.ResponseWriter, r *http.Request) {
    id := strings.TrimPrefix(r.URL.Path, "/api/sessions/")

    switch r.Method {
    case http.MethodGet:
        msgs, err := s.sessions.GetMessages(id)
        if err != nil {
            http.Error(w, "Session not found", http.StatusNotFound)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(msgs)

    case http.MethodDelete:
        if err := s.sessions.Delete(id); err != nil {
            http.Error(w, "Session not found", http.StatusNotFound)
            return
        }
        w.WriteHeader(http.StatusNoContent)

    default:
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/server/ -v
```
Expected: All 3 tests PASS

- [x] **Step 5: Commit** ✅

```bash
git add -A && git commit -m "feat: add HTTP/SSE server with chat and session endpoints"
```

---

## Phase 11: Web UI (Material Design 3)

### Task 11.1: Web UI shell with Material Design 3

**Files:**
- Create: `web/index.html`
- Create: `web/css/style.css`
- Create: `web/js/app.js`
- Create: `web/js/sse.js`

- [ ] **Step 1: Create the HTML shell**

```html
<!-- web/index.html -->
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Convallaria - Coding Agent</title>
    <link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Roboto:wght@300;400;500;700&display=swap">
    <link rel="stylesheet" href="css/style.css">
</head>
<body>
    <div class="app-container">
        <!-- Sidebar -->
        <aside class="sidebar" id="sidebar">
            <div class="sidebar-header">
                <h1 class="logo">Convallaria</h1>
                <button class="btn-icon" id="btn-new-chat" title="新建会话">+</button>
            </div>
            <div class="session-list" id="session-list">
                <!-- Session items rendered by JS -->
            </div>
        </aside>

        <!-- Main -->
        <main class="main">
            <div class="chat-container" id="chat-container">
                <div class="chat-messages" id="chat-messages">
                    <div class="welcome-message">
                        <h2>Convallaria Coding Agent</h2>
                        <p>输入你的编码任务，我会帮你完成。</p>
                    </div>
                </div>
                <div class="chat-input">
                    <textarea id="chat-input" placeholder="输入消息..." rows="2"></textarea>
                    <button class="btn-send" id="btn-send">发送</button>
                </div>
            </div>
        </main>

        <!-- File Browser Panel (toggle) -->
        <aside class="panel" id="file-panel" style="display:none;">
            <div class="panel-header">
                <h3>文件浏览</h3>
                <button class="btn-icon" id="btn-close-files">×</button>
            </div>
            <div class="file-tree" id="file-tree">
                <!-- File tree rendered by JS -->
            </div>
        </aside>

        <!-- Config Panel (toggle) -->
        <aside class="panel" id="config-panel" style="display:none;">
            <div class="panel-header">
                <h3>配置</h3>
                <button class="btn-icon" id="btn-close-config">×</button>
            </div>
            <div class="config-content">
                <!-- Config form rendered by JS -->
            </div>
        </aside>
    </div>

    <script src="js/sse.js"></script>
    <script src="js/app.js"></script>
</body>
</html>
```

- [ ] **Step 2: Create the CSS**

```css
/* web/css/style.css */
:root {
    --md-sys-color-surface: #1e1e2e;
    --md-sys-color-surface-variant: #2a2a3e;
    --md-sys-color-primary: #6750a4;
    --md-sys-color-on-surface: #e0e0e0;
    --md-sys-color-outline: #444;
    --sidebar-width: 260px;
}

* { margin: 0; padding: 0; box-sizing: border-box; }

body {
    font-family: 'Roboto', sans-serif;
    background: var(--md-sys-color-surface);
    color: var(--md-sys-color-on-surface);
    height: 100vh;
    overflow: hidden;
}

.app-container {
    display: flex;
    height: 100vh;
}

/* Sidebar */
.sidebar {
    width: var(--sidebar-width);
    background: var(--md-sys-color-surface-variant);
    border-right: 1px solid var(--md-sys-color-outline);
    display: flex;
    flex-direction: column;
}

.sidebar-header {
    padding: 16px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px solid var(--md-sys-color-outline);
}

.logo { font-size: 18px; font-weight: 500; }

.session-list {
    flex: 1;
    overflow-y: auto;
    padding: 8px;
}

.session-item {
    padding: 10px 12px;
    border-radius: 8px;
    cursor: pointer;
    margin-bottom: 4px;
    font-size: 14px;
}

.session-item:hover { background: rgba(255,255,255,0.08); }
.session-item.active { background: var(--md-sys-color-primary); }

/* Main */
.main {
    flex: 1;
    display: flex;
    flex-direction: column;
}

.chat-container {
    flex: 1;
    display: flex;
    flex-direction: column;
    max-width: 800px;
    margin: 0 auto;
    width: 100%;
}

.chat-messages {
    flex: 1;
    overflow-y: auto;
    padding: 24px;
}

.welcome-message {
    text-align: center;
    margin-top: 40vh;
    transform: translateY(-50%);
    opacity: 0.6;
}

.welcome-message h2 { font-size: 24px; margin-bottom: 8px; }

.message {
    margin-bottom: 16px;
    padding: 12px 16px;
    border-radius: 12px;
    max-width: 85%;
}

.message.user {
    background: var(--md-sys-color-primary);
    margin-left: auto;
}

.message.assistant {
    background: var(--md-sys-color-surface-variant);
    margin-right: auto;
}

.message.tool {
    background: transparent;
    border: 1px solid var(--md-sys-color-outline);
    font-family: monospace;
    font-size: 13px;
    margin-right: auto;
}

/* Chat Input */
.chat-input {
    padding: 16px;
    display: flex;
    gap: 8px;
    border-top: 1px solid var(--md-sys-color-outline);
}

.chat-input textarea {
    flex: 1;
    background: var(--md-sys-color-surface-variant);
    border: 1px solid var(--md-sys-color-outline);
    border-radius: 8px;
    color: var(--md-sys-color-on-surface);
    padding: 10px;
    font-family: inherit;
    font-size: 14px;
    resize: none;
}

.btn-send {
    background: var(--md-sys-color-primary);
    color: white;
    border: none;
    border-radius: 8px;
    padding: 0 20px;
    cursor: pointer;
    font-weight: 500;
}

.btn-icon {
    background: none;
    border: none;
    color: var(--md-sys-color-on-surface);
    font-size: 20px;
    cursor: pointer;
    padding: 4px 8px;
    border-radius: 4px;
}

.btn-icon:hover { background: rgba(255,255,255,0.1); }

/* Panel */
.panel {
    width: 300px;
    background: var(--md-sys-color-surface-variant);
    border-left: 1px solid var(--md-sys-color-outline);
    display: flex;
    flex-direction: column;
}

.panel-header {
    padding: 16px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px solid var(--md-sys-color-outline);
}
```

- [ ] **Step 3: Create the JavaScript**

```javascript
// web/js/sse.js
class SSEClient {
    constructor(url) {
        this.url = url;
        this.eventSource = null;
        this.listeners = {};
    }

    connect(body) {
        // Use fetch + ReadableStream for POST-based SSE
        return fetch(this.url, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body),
        }).then(response => {
            const reader = response.body.getReader();
            const decoder = new TextDecoder();
            let buffer = '';

            const processChunk = ({ done, value }) => {
                if (done) return;
                buffer += decoder.decode(value, { stream: true });
                const lines = buffer.split('\n');
                buffer = lines.pop() || '';

                let eventType = '';
                for (const line of lines) {
                    if (line.startsWith('event: ')) {
                        eventType = line.slice(7);
                    } else if (line.startsWith('data: ')) {
                        const data = line.slice(6);
                        this.emit(eventType, JSON.parse(data));
                    }
                }
                return reader.read().then(processChunk);
            };
            return reader.read().then(processChunk);
        });
    }

    on(event, callback) {
        if (!this.listeners[event]) this.listeners[event] = [];
        this.listeners[event].push(callback);
    }

    emit(event, data) {
        (this.listeners[event] || []).forEach(cb => cb(data));
    }
}
```

```javascript
// web/js/app.js
class ConvallariaApp {
    constructor() {
        this.currentSessionId = '';
        this.sessions = [];
        this.init();
    }

    init() {
        this.bindElements();
        this.bindEvents();
        this.loadSessions();
    }

    bindElements() {
        this.chatMessages = document.getElementById('chat-messages');
        this.chatInput = document.getElementById('chat-input');
        this.btnSend = document.getElementById('btn-send');
        this.btnNewChat = document.getElementById('btn-new-chat');
        this.sessionList = document.getElementById('session-list');
    }

    bindEvents() {
        this.btnSend.addEventListener('click', () => this.sendMessage());
        this.btnNewChat.addEventListener('click', () => this.newChat());
        this.chatInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                this.sendMessage();
            }
        });
    }

    async loadSessions() {
        try {
            const resp = await fetch('/api/sessions');
            this.sessions = await resp.json();
            this.renderSessions();
        } catch (e) {
            console.error('Failed to load sessions:', e);
        }
    }

    renderSessions() {
        this.sessionList.innerHTML = this.sessions.map(s => `
            <div class="session-item ${s.ID === this.currentSessionId ? 'active' : ''}"
                 onclick="app.switchSession('${s.ID}')">
                ${s.Title}
            </div>
        `).join('');
    }

    newChat() {
        this.currentSessionId = '';
        this.chatMessages.innerHTML = `
            <div class="welcome-message">
                <h2>Convallaria Coding Agent</h2>
                <p>输入你的编码任务，我会帮你完成。</p>
            </div>
        `;
    }

    switchSession(id) {
        this.currentSessionId = id;
        this.loadMessages(id);
        this.renderSessions();
    }

    async loadMessages(sessionId) {
        try {
            const resp = await fetch(`/api/sessions/${sessionId}`);
            const msgs = await resp.json();
            this.chatMessages.innerHTML = '';
            msgs.forEach(msg => this.appendMessage(msg.Role, msg.Content));
        } catch (e) {
            console.error('Failed to load messages:', e);
        }
    }

    async sendMessage() {
        const message = this.chatInput.value.trim();
        if (!message) return;
        this.chatInput.value = '';

        this.appendMessage('user', message);

        const sse = new SSEClient('/api/chat');
        let assistantContent = '';

        sse.on('session', (data) => {
            this.currentSessionId = data.id;
            this.loadSessions();
        });

        sse.on('token', (data) => {
            assistantContent += data.token;
            this.updateAssistantMessage(assistantContent);
        });

        sse.on('error', (data) => {
            this.appendMessage('system', `Error: ${data.message}`);
        });

        sse.on('done', () => {
            this.loadSessions();
        });

        sse.connect({
            session_id: this.currentSessionId,
            message: message,
        });
    }

    appendMessage(role, content) {
        const div = document.createElement('div');
        div.className = `message ${role}`;
        div.textContent = content;
        this.chatMessages.appendChild(div);
        this.chatMessages.scrollTop = this.chatMessages.scrollHeight;
    }

    updateAssistantMessage(content) {
        let lastMsg = this.chatMessages.querySelector('.message.assistant:last-child');
        if (!lastMsg) {
            lastMsg = document.createElement('div');
            lastMsg.className = 'message assistant';
            this.chatMessages.appendChild(lastMsg);
        }
        lastMsg.textContent = content;
        this.chatMessages.scrollTop = this.chatMessages.scrollHeight;
    }
}

const app = new ConvallariaApp();
```

- [ ] **Step 4: Verify the web UI files exist**

```bash
ls -la web/index.html web/css/style.css web/js/app.js web/js/sse.js
```

- [x] **Step 5: Commit** ✅

```bash
git add -A && git commit -m "feat: add Material Design 3 Web UI shell"
```

---

## Phase 12: CLI Entry Point

### Task 12.1: Main entry point and distribution build

**Files:**
- Modify: `cmd/convallaria/main.go`

- [ ] **Step 1: Update main.go**

```go
// cmd/convallaria/main.go
package main

import (
    "flag"
    "fmt"
    "log"
    "os"

    "github.com/Convallariaxhr/convallaria/internal/agent"
    "github.com/Convallariaxhr/convallaria/internal/config"
    "github.com/Convallariaxhr/convallaria/internal/credential"
    "github.com/Convallariaxhr/convallaria/internal/llm"
    "github.com/Convallariaxhr/convallaria/internal/server"
    "github.com/Convallariaxhr/convallaria/internal/session"
)

func main() {
    configPath := flag.String("config", "convallaria.yaml", "Path to config file")
    port := flag.Int("port", 8080, "HTTP server port")
    flag.Parse()

    // Load config
    cfg, err := config.Load(*configPath)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Load credentials
    store := credential.NewMemoryStore()
    apiKey, err := store.Get(cfg.LLM.Provider)
    if err != nil {
        apiKey = os.Getenv(cfg.LLM.APIKeyEnv)
    }
    if apiKey == "" {
        fmt.Println("Warning: No API key found. Run 'convallaria init' to configure.")
    }

    // Create LLM provider (currently only mock is supported without real API)
    // In production: deepseek.New(apiKey), openai.New(apiKey), etc.
    provider := llm.NewMockProvider()

    // Create agent
    ag := agent.New(agent.Config{
        MaxTurns:  cfg.Agent.MaxTurns,
        Provider:  provider,
        Workspace: cfg.Agent.Workspace,
    })

    // Create session manager
    sessMgr := session.NewManager()

    // Start server
    srv := server.New(server.Config{
        Port:      *port,
        StaticDir: "web",
    }, ag, sessMgr)

    fmt.Printf("Convallaria starting on http://localhost:%d\n", *port)
    if err := srv.Start(); err != nil {
        log.Fatalf("Server error: %v", err)
    }
}
```

- [ ] **Step 2: Verify build**

```bash
go build -o convallaria.exe ./cmd/convallaria/
```
Expected: Build succeeds

- [ ] **Step 3: Run all tests**

```bash
go test ./... -v
```
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "feat: add CLI entry point and distribution build"
```

---

## Post-Plan: CI/CD

### Task 13.1: GitLab CI configuration

**Files:**
- Create: `.gitlab-ci.yml`

```yaml
# .gitlab-ci.yml
stages:
  - test
  - build

unit-test:
  stage: test
  image: golang:1.22
  script:
    - go test ./... -v -count=1
  cache:
    paths:
      - /go/pkg/mod

build:
  stage: build
  image: golang:1.22
  script:
    - go build -o convallaria ./cmd/convallaria/
  artifacts:
    paths:
      - convallaria
```

---

## Plan Summary

| Phase | Tasks | Key Deliverables |
|-------|-------|-----------------|
| 1 | 1.1-1.2 | Go module, LLM interface, Mock provider |
| 2 | 2.1-2.2 | Config system, Credential management |
| 3 | 3.1-3.2 | Action parser, Tool registry, 6 tools |
| 4 | 4.1 | Guardrail (3-layer safety checks) |
| 5 | 5.1 | **Feedback loop (Build/Vet/Test validators)** |
| 6 | 6.1 | Agent main loop with mock integration tests |
| 7 | 7.1-7.2 | Context window manager, Error recovery |
| 8 | 8.1 | Memory system (rules + keyword search) |
| 9 | 9.1 | Session management (CRUD + export) |
| 10 | 10.1 | HTTP/SSE server |
| 11 | 11.1 | Material Design 3 Web UI |
| 12 | 12.1 | CLI entry point, build, CI |