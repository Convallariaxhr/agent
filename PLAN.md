# Convallaria Coding Agent Harness · Implementation Plan

> Superpowers: brainstorming → writing-plans → using-git-worktrees → subagent-driven-development → test-driven-development → requesting-code-review → finishing-a-development-branch

**Goal:** Build a complete coding agent harness (Go backend + Material Design 3 Web UI) with mock-LLM-driven deterministic unit tests.

**Architecture:** Go backend + HTTP/SSE + agent main loop + 6 tools + guardrails + feedback loop + memory + session management. Web UI with SSE streaming. Multi-provider LLM abstraction (DeepSeek/OpenAI/Anthropic/Mock).

**Tech Stack:** Go 1.22+, SQLite, Material Design 3 + Open Design, SSE, Docker.

---

## Implementation Status (All Complete)

| Phase | Task | Status | Key Commit |
|-------|------|--------|------------|
| 1 | 1.1 Go module init | ✅ | `6bc06d6` |
| 1 | 1.2 LLM Provider interface + Mock | ✅ | `437c9f2` |
| 2 | 2.1 Config system (YAML + env) | ✅ | `7889f4e` |
| 2 | 2.2 Credential management | ✅ | `e864579` |
| 3 | 3.1 Action Parser | ✅ | `c99d266` |
| 3 | 3.2 Tool registry + 6 tool implementations | ✅ | `1c0957d` |
| 4 | 4.1 Guardrail (dangerous cmd + file scope + git) | ✅ | `1a2e207` |
| 5 | 5.1 Feedback loop (Build/Vet/Test validators) | ✅ | `3e1dcac` |
| 6 | 6.1 Agent main loop + Mock integration tests | ✅ | `d3af108` |
| 7 | 7.1 Context window manager | ✅ | `8b89014` |
| 7 | 7.2 Error recovery (retry/degrade) | ✅ | `8b89014` |
| 8 | 8.1 Memory system (rules + keyword search) | ✅ | `46a4e44` |
| 9 | 9.1 Session management (CRUD + export) | ✅ | `51d1ffe` |
| 10 | 10.1 HTTP/SSE server | ✅ | `f42cfe9` |
| 11 | 11.1 Material Design 3 Web UI | ✅ | `213b6e0` |
| 12 | 12.1 CLI entry point | ✅ | `213b6e0` |
| 13 | Docker + CI/CD (GitHub Actions + GitLab CI) | ✅ | `f52e06a`, `8017960` |
| — | Code review: 16 findings, 9 fixed | ✅ | `9e33029` |
| — | DeepSeek provider | ✅ | `3cdac3f` |
| — | SQLite persistence (sessions + memory) | ✅ | `da04eef` |
| — | HITL approval dialog | ✅ | `ea88223` |
| — | File browser + config panels | ✅ | `d21d3f7` |
| — | OpenAI + Anthropic providers | ✅ | `8017960` |
| — | Anti-hallucination detection | ✅ | `200b269`, `3472191` |
| — | OpenCode code review fixes (14 issues) | ✅ | `bb029de`, `3344796` |
| — | Function calling format compliance | ✅ | `5cf032a`, `125181d` |
| — | Session rename + delete, file browser nav | ✅ | `db3ec64`, `a45c2f5` |
| — | Lily-of-the-Valley logo, thinking spinner | ✅ | `0a79d82`, `6fa125d` |
| — | Conversation history + workspace-aware tools | ✅ | `b684f67`, `8a889b9` |
| — | SSE keepalive, system prompt tuning | ✅ | `6a751dc`, `a423318` |
| — | Documentation + delivery | ✅ | `f023b8e`, `fa7b0b4` |

---

## Phase 1: Project Scaffold & LLM Abstraction

### Task 1.1: Initialize Go module and project structure
- **Files:** `go.mod`, `cmd/convallaria/main.go`, directory tree with 14 packages
- **Verify:** `go build ./cmd/convallaria/` succeeds

### Task 1.2: LLM Provider interface and Mock implementation
- **Files:** `internal/llm/provider.go`, `mock.go`, `mock_test.go`
- **Key types:** `Provider` interface (Chat/ChatSync), `Message`, `ToolCall`, `Response`, `StreamEvent`
- **Mock:** Preset response queue for deterministic testing
- **Verify:** 4 mock tests pass — sync text, sync tool call, streaming, call count

---

## Phase 2: Config & Credential

### Task 2.1: Configuration system
- **Files:** `internal/config/config.go`, `config_test.go`
- **Features:** YAML config + env var overrides + sensible defaults
- **Env vars:** `CONVALLARIA_PROVIDER`, `CONVALLARIA_MODEL`, `CONVALLARIA_API_KEY`, `CONVALLARIA_BASE_URL`
- **Verify:** 3 tests — YAML load, defaults, env override

### Task 2.2: Credential management
- **Files:** `internal/credential/credential.go`, `credential_test.go`
- **Features:** MemoryStore with Set/Get/List/Delete, MaskKey for log safety
- **Verify:** 6 tests including mask edge cases

---

## Phase 3: Parser & Tool System

### Task 3.1: Action parser
- **Files:** `internal/parser/parser.go`, `parser_test.go`
- **Features:** Parse LLM response → ActionList, handle malformed JSON gracefully
- **Verify:** 4 tests — text, single tool call, multiple, malformed JSON

### Task 3.2: Tool registry + 6 tool implementations
- **Files:** `internal/tools/registry.go`, `file_reader.go`, `file_writer.go`, `shell_runner.go`, `searcher.go`, `test_runner.go`, `git_ops.go`
- **Each tool:** Name(), Description(), Schema() (JSON Schema for function calling), Execute()
- **Verify:** Registry register/execute/unknown tool tests

---

## Phase 4: Guardrails

### Task 4.1: Guardrail implementation
- **Files:** `internal/guardrail/guardrail.go`, `guardrail_test.go`
- **Three layers:** Dangerous command regex → File scope (workspace boundary) → Git dangerous ops
- **Verify:** 6 tests — block rm -rf, block outside workspace, allow inside, block force push, allow safe, disabled

---

## Phase 5: Feedback Loop (Focus Dimension)

### Task 5.1: Feedback validators
- **Files:** `internal/feedback/feedback.go`, `build_validator.go`, `vet_validator.go`, `test_validator.go`, `feedback_test.go`
- **Pipeline:** Build → Vet → Test, first failure stops, structured errors injected back to LLM
- **Verify:** 5 tests — valid build, invalid build, all pass, build failure, feedback→message conversion

---

## Phase 6: Agent Main Loop

### Task 6.1: Agent main loop with Mock LLM integration
- **Files:** `internal/agent/loop.go`, `loop_test.go`
- **Flow:** Build context → Call LLM → Parse actions → Guardrail check → Execute tools → Feedback loop → Repeat
- **Stop condition:** Pure text response (no tool calls)
- **Verify:** 5 tests — text response, tool call + file creation, guardrail block, feedback loop detection, max turns exceeded

---

## Phase 7: Context Window & Error Recovery

### Task 7.1: Context window manager
- **Files:** `internal/context/manager.go`, `manager_test.go`
- **Features:** Token estimation (~4 chars/token), compression threshold, truncation with summary placeholder
- **Verify:** 3 tests — estimate, needs compression, compress

### Task 7.2: Error recovery
- **Files:** `internal/recovery/recovery.go`, `recovery_test.go`
- **Features:** Retry with error feedback, max retries limit, graceful degradation
- **Verify:** 2 tests — retry on parse error, degrade on max retries

---

## Phase 8: Memory System

### Task 8.1: Memory store
- **Files:** `internal/memory/rules.go`, `store.go`, `embedder.go`, `memory_test.go`
- **Features:** CONVALLARIA.md / CLAUDE.md rule loading, keyword-based search store, embedder placeholder
- **Verify:** 4 tests — rules load, no file, insert+search, empty search

---

## Phase 9: Session Management

### Task 9.1: Session manager
- **Files:** `internal/session/manager.go`, `manager_test.go`
- **Features:** Session CRUD, message history, export, SQLiteStore for persistence
- **Verify:** 5 tests — create/get, list, delete, add messages, export

---

## Phase 10: HTTP/SSE Server

### Task 10.1: Server with SSE streaming
- **Files:** `internal/server/handler.go`, `sse.go`, `server_test.go`
- **Endpoints:** POST `/api/chat` (SSE), GET `/api/sessions`, GET/DELETE/PUT `/api/sessions/:id`, GET `/api/files`, POST `/api/approve`
- **SSE:** CORS headers, keepalive heartbeat, streaming token events
- **Verify:** Server tests — chat SSE, session list, SSE write

---

## Phase 11: Web UI

### Task 11.1: Material Design 3 Web UI
- **Files:** `web/index.html`, `web/css/style.css`, `web/js/app.js`, `web/js/sse.js`
- **Features:** Chat panel with SSE streaming, session sidebar, file browser panel (navigate + preview), config panel, HITL approval dialog, session rename/delete, thinking spinner animation
- **Design:** Dark theme, Lily-of-the-Valley logo, Material Design 3 tokens, Inter + JetBrains Mono fonts

---

## Phase 12: CLI Entry Point

### Task 12.1: Main entry point
- **Files:** `cmd/convallaria/main.go`
- **Flags:** `-config` (yaml path), `-port` (server port), `-workspace` (override)
- **Features:** Config loading, provider selection (DeepSeek/OpenAI/Anthropic/Mock), graceful shutdown
- **Verify:** `go build`, `go run`, browser access at localhost:8080

## Plan Summary

| Phase | Tasks | Deliverables |
|-------|-------|-------------|
| 1 | 1.1–1.2 | Go module, LLM interface, Mock provider |
| 2 | 2.1–2.2 | Config system, Credential management |
| 3 | 3.1–3.2 | Action parser, Tool registry + 6 tools |
| 4 | 4.1 | Guardrail (3-layer safety) |
| 5 | 5.1 | Feedback loop (Build/Vet/Test) |
| 6 | 6.1 | Agent main loop + Mock integration |
| 7 | 7.1–7.2 | Context window, Error recovery |
| 8 | 8.1 | Memory system (rules + keyword search) |
| 9 | 9.1 | Session management (CRUD + export) |
| 10 | 10.1 | HTTP/SSE server |
| 11 | 11.1 | Material Design 3 Web UI |
| 12 | 12.1 | CLI entry, build, CI |
| 13 | CI/CD | Docker + GitHub Actions + GitLab CI |
