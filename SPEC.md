# SPEC.md — Convallaria Coding Agent Harness

> AI4SE 期末项目 · A · Coding Agent Harness
>
> 核心等式：**Agent = LLM + Harness**。LLM（DeepSeek 等）负责"决定下一步做什么"，harness 负责主循环、工具执行、治理护栏、反馈闭环、记忆管理、上下文窗口、错误恢复、会话管理。

---

## 1. 问题陈述

### 1.1 要解决的问题

当前市面上的编码智能体（Claude Code、Cursor、GitHub Copilot 等）均为闭源商业产品，用户无法理解其内部机制，也无法自定义其行为。对于想要学习"AI 如何驱动编码"的开发者，缺乏一个**透明、可理解、可扩展**的开源 harness 实现。

### 1.2 目标用户

- 学习 AI4SE 的学生，想理解"Agent = LLM + Harness"的工程实现
- 个人开发者，想要一个能用自己 API Key 的、透明可控的编码助手
- 对 agent 机制感兴趣的研究者，需要一个可注入 mock 进行实验的框架

### 1.3 为什么值得做

当前 AI 编码工具的 harness 层（治理、反馈、记忆、安全）是决定 agent 可靠性最关键的部分，但商业产品将它们封闭在黑盒里。本项目将其拆解为可独立测试、可替换的模块，降低理解和定制的门槛。

---

## 2. 用户故事

| # | 用户故事 | 验收标准 |
|---|---------|---------|
| US1 | 作为一个开发者，我可以在浏览器中打开 Web UI，输入编码任务，看到 agent 一步步执行并流式展示结果 | Web UI 可访问，输入任务后 SSE 实时推送 token 和工具执行状态 |
| US2 | 作为一个开发者，我可以配置自己的 API Key（DeepSeek/OpenAI 等），Key 安全存储在系统凭据管理中，不暴露在代码或配置文件中 | 首次运行引导录入 Key，后续可从凭据管理读取，查看状态时不回显明文 |
| US3 | 作为一个开发者，我可以让 agent 读写文件、执行 shell 命令、运行测试，agent 会根据测试失败结果自动修正代码 | Agent 写代码 → 跑测试 → 失败 → 自动修正 → 重跑 → 直到通过 |
| US4 | 作为一个项目维护者，我可以在项目根目录放置 CONVALLARIA.md 规则文件，agent 每次启动时自动加载并遵守其中的约定 | 在项目目录下启动 agent，系统提示中包含 CONVALLARIA.md 内容 |
| US5 | 作为一个开发者，当 agent 尝试执行危险命令（如 rm -rf /）时，系统会拦截并弹出确认框，我可以选择允许或拒绝 | 危险命令被护栏拦截，前端弹出审批界面，选择后 agent 继续或跳过 |
| US6 | 作为一个开发者，我可以在多个会话之间切换，关闭浏览器后重新打开，历史会话仍然保留 | 会话列表显示所有历史会话，点击可恢复，消息历史完整 |
| US7 | 作为一个开发者，我可以在配置中切换不同的 LLM 供应商（DeepSeek/OpenAI），无需修改代码 | 修改配置文件中的 provider 字段，重启后使用新供应商 |

---

## 3. 功能规约

### 3.1 Agent 主循环

**输入**：用户自然语言任务描述
**行为**：
1. 构建上下文（系统提示 + 规则文件 + 记忆检索 + 对话历史）
2. 调用 LLM（通过 Provider 接口）
3. 解析 LLM 响应（纯文本 = 停机；tool call = 提取动作列表）
4. 对每个动作：护栏检查 → 执行 → 反馈闭环 → 回灌结果
5. 循环直到停机条件满足
**输出**：最终回复文本
**边界条件**：最大轮次（默认 50）、上下文窗口满时触发自动摘要压缩
**错误处理**：LLM 返回格式错误 → 重试（最多 3 次）→ 降级报告

### 3.2 工具执行器

**统一接口**：`Tool.Execute(ctx context.Context, params map[string]any) (Result, error)`

| 工具 | 功能 | 权限要求 |
|------|------|---------|
| FileReader | 读取文件内容，受文件范围限制 | 无 |
| FileWriter | 写入/创建文件，受文件范围限制 | 无 |
| ShellRunner | 执行 shell 命令，受护栏检查 | 需通过护栏 |
| Searcher | Grep 搜索文件内容 / Glob 匹配文件名 | 无 |
| TestRunner | 运行 `go test`（或对应语言测试框架） | 无 |
| GitOps | git status/commit/branch/diff 等 | push --force 等需审批 |

**工具注册表**：通过 `ToolRegistry` 注册，LLM 调用时按名称分发。

### 3.3 反馈闭环（重点维度）

**三层校验器**：

| 校验器 | 触发条件 | 检查内容 | 失败时操作 |
|--------|---------|---------|-----------|
| BuildValidator | 写入 .go 文件后 | `go build` 编译检查 | 结构化错误回灌 LLM |
| VetValidator | Build 通过后 | `go vet` 静态分析 | 警告回灌 LLM |
| TestValidator | 写入测试或源码后 | `go test -json` | 失败用例详情回灌 LLM |

**反馈消息格式**（见 §3.3.1 架构部分的数据流），每个错误包含：文件名、行号、列号、错误信息、修复建议。

**停机条件**：三层校验全部通过，或 LLM 声明完成。

### 3.4 治理护栏

**三层拦截**：

| 层级 | 检查内容 | 匹配方式 | 动作 |
|------|---------|---------|------|
| L1: 命令 | `rm -rf /`、`mkfs`、`dd`、fork bomb、`chmod 777 /`、`> /dev/sda`、`shutdown`、`reboot`、`curl ... \| sh` 等 | 正则 + 危险模式库 | 拦截 → HITL |
| L2: 文件 | 路径是否在工作区（项目目录）内 | 路径前缀匹配 | 超出范围 → 拦截 → HITL |
| L3: Git | `push --force`、`reset --hard`、`clean -fdx`、`branch -D` | 精确匹配 | 拦截 → HITL |

**HITL 审批流程**：拦截 → SSE 推送 `approval_required` → 前端弹窗 → 用户选择「允许一次」「永久允许」「拒绝」→ 结果回灌

### 3.5 记忆系统

**双层架构**：

| 层 | 存储 | 加载方式 | 内容 |
|----|------|---------|------|
| 规则层 | 文件系统（`CONVALLARIA.md`、`convallaria.yaml`） | 全量加载到系统提示 | 编码规范、项目约定、工具权限 |
| 语义层 | SQLite + 嵌入向量 | 按语义相似度检索 Top-K，按需注入 | 历史决策、代码模式、踩坑记录 |

**记忆生命周期**：写入（决策后自动提取摘要）→ 嵌入（文本 → 向量）→ 检索（当前任务 → 余弦相似度 Top-K）→ 注入（加入 LLM 上下文）

### 3.6 上下文窗口管理

- **Token 估算**：使用 tiktoken 兼容算法估算消息 token 数
- **阈值**：达到窗口 80% 时触发压缩
- **压缩策略**：保留系统提示 + 最近 N 轮完整对话，早期消息 → 调用 LLM 生成摘要替换
- **配置项**：`max_context_tokens`（默认 64000）、`compression_threshold`（默认 0.8）

### 3.7 错误恢复

| 层级 | 触发条件 | 操作 | 限制 |
|------|---------|------|------|
| 重试 | LLM 返回格式错误的 JSON | 错误信息回灌 → 重新请求 | 最多 3 次 |
| 纠错 | 工具执行失败（文件不存在等） | 错误信息回灌 → LLM 修正参数 | 最多 2 次 |
| 降级 | 重试/纠错耗尽 | 跳过当前动作，报告失败原因 | — |

### 3.8 会话管理

- **多会话**：每个项目/任务独立会话，独立上下文和记忆
- **持久化**：SQLite 存储全部消息历史，关闭浏览器后完全恢复
- **会话列表**：前端侧边栏展示，支持切换、重命名、删除
- **导出**：导出为 Markdown 或 JSON 文件

### 3.9 多模型支持

**Provider 接口**：
```go
type LLMProvider interface {
    Chat(ctx context.Context, messages []Message) (<-chan StreamEvent, error)
    Models() []ModelInfo
}
```

**实现**：
| Provider | 端点 | 协议 |
|----------|------|------|
| DeepSeek | `api.deepseek.com` | OpenAI 兼容 |
| OpenAI | `api.openai.com` | 原生 |
| Anthropic | `api.anthropic.com` | Anthropic Messages |
| MockProvider | — | 预设响应（测试用） |

**配置示例**：
```yaml
llm:
  provider: deepseek
  model: deepseek-chat
  api_key_env: DEEPSEEK_API_KEY
```

### 3.10 配置系统

**三层配置，优先级从低到高**：

1. **全局配置**：`~/.convallaria.yaml` — LLM endpoint、默认模型
2. **项目配置**：`./convallaria.yaml` — 工具权限、护栏规则、工作目录
3. **环境变量**：`CONVALLARIA_API_KEY`、`CONVALLARIA_PROVIDER` 等覆盖

### 3.11 Web UI

- **技术栈**：Material Design 3 + Open Design
- **通信**：SSE（流式推送）+ HTTP（请求）
- **页面**：对话面板、文件浏览面板、配置面板、会话列表侧边栏
- **SSE 事件类型**：`token`、`tool_start`、`tool_output`、`tool_end`、`feedback`、`approval`

---

## 4. 非功能性需求

### 4.1 性能

- SSE 首次 token 延迟 < 3 秒（排除 LLM API 延迟）
- 前端渲染不阻塞 SSE 流
- SQLite 记忆检索 < 100ms

### 4.2 安全（凭据威胁模型）

**威胁模型**：
- T1：攻击者读取源码 → 获取 API Key → 未授权使用 LLM
- T2：攻击者读取日志/终端 history → 获取 API Key
- T3：攻击者读取 `.env` 文件 → 获取 API Key
- T4：恶意 prompt 注入 → 绕过护栏 → 执行危险命令

**对策**：
| 威胁 | 对策 |
|------|------|
| T1 | API Key 存储于操作系统凭据管理（Windows Credential Manager / macOS Keychain / Linux Secret Service），源码中无硬编码 |
| T2 | 日志中不记录 API Key，命令行参数不接受 Key（只通过环境变量或凭据管理） |
| T3 | `.env` 已加入 `.gitignore`，首次运行引导用户通过隐藏输入录入 |
| T4 | 护栏是代码实现的确定性检查，不依赖 LLM 遵从提示词 |

**凭据生命周期**：
- **录入**：首次运行 `convallaria init`，引导用户选择 provider → 隐藏输入 Key → 存入 OS 凭据管理
- **查看**：`convallaria credential status` 显示 provider 名称和 Key 的掩码（如 `sk-****xxxx`）
- **更新**：`convallaria credential set` 覆盖旧 Key
- **清除**：`convallaria credential clear` 从凭据管理中删除

### 4.3 可用性

- 首次运行引导：初始化凭据 → 创建默认配置 → 打开 Web UI
- 错误信息用英文描述（技术工具标准），包含建议的修复操作
- 所有操作提供明确的进度反馈

### 4.4 可观测性

- 日志级别：debug/info/warn/error
- 记录每个 turn 的：LLM 调用耗时、token 消耗、工具执行耗时、护栏触发次数
- 日志输出到文件 + 终端

---

## 5. 系统架构

### 5.1 组件图

```
浏览器 (Material Design 3 Web UI)
  ├── 💬 对话面板
  ├── 📁 文件浏览
  ├── ⚙️ 配置面板
  └── 📋 会话列表
        ↓ SSE (流式推送)   ↑ HTTP (用户请求)
┌──────────────────────────────────────────────────┐
│                  Go 后端 :8080                     │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐  │
│  │HTTP/SSE  │ │Agent     │ │治理护栏 (Guardrail)│  │
│  │Handler   │ │主循环     │ │L1命令 L2文件 L3Git│  │
│  └──────────┘ └──────────┘ └──────────────────┘  │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐  │
│  │工具执行器  │ │反馈闭环   │ │记忆系统           │  │
│  │6个工具    │ │Build+Vet │ │规则文件+向量检索  │  │
│  │统一注册表  │ │+Test     │ │SQLite            │  │
│  └──────────┘ └──────────┘ └──────────────────┘  │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐  │
│  │上下文管理  │ │错误恢复   │ │会话管理           │  │
│  │Token计数  │ │重试→纠错  │ │多会话+持久化     │  │
│  │摘要压缩   │ │→降级     │ │+导出             │  │
│  └──────────┘ └──────────┘ └──────────────────┘  │
│  ┌──────────┐ ┌──────────┐                       │
│  │LLM Provider│ │配置系统   │                      │
│  │DeepSeek   │ │全局+项目  │                      │
│  │OpenAI     │ │+环境变量  │                      │
│  │Anthropic  │ │+CLI      │                      │
│  │Mock       │ │          │                      │
│  └──────────┘ └──────────┘                       │
└──────────────────────────────────────────────────┘
        ↓ HTTP (各供应商 API)   ↑
🤖 DeepSeek / OpenAI / Anthropic
```

### 5.2 数据流

1. 用户通过 Web UI 输入指令 → HTTP POST 到后端
2. Server Handler 创建/恢复会话 → 交给 Agent 主循环
3. Agent 构建上下文（系统提示 + 规则文件 + 记忆检索 + 对话历史）
4. 通过 Provider 接口调用 LLM
5. LLM 返回流式响应（token 通过 SSE 实时推送前端）
6. 解析器提取 tool call → 护栏检查
7. 工具执行器执行 → 反馈闭环检查（编译/测试）
8. 结果回灌给 LLM → 循环直到停机
9. 会话自动持久化到 SQLite

### 5.3 外部依赖

- DeepSeek / OpenAI / Anthropic API（LLM 推理）
- 本地嵌入模型（记忆向量化，可选：onnxruntime 或调用外部嵌入 API）
- SQLite（会话 + 记忆存储）
- 操作系统凭据管理 API（安全存储 Key）

---

## 6. 数据模型

### 6.1 会话 (Session)

```sql
CREATE TABLE sessions (
    id          TEXT PRIMARY KEY,    -- UUID
    title       TEXT NOT NULL,       -- 会话标题
    project_dir TEXT,                -- 关联项目目录
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);
```

### 6.2 消息 (Message)

```sql
CREATE TABLE messages (
    id          TEXT PRIMARY KEY,    -- UUID
    session_id  TEXT NOT NULL REFERENCES sessions(id),
    role        TEXT NOT NULL,       -- user | assistant | system | tool
    content     TEXT NOT NULL,
    tool_calls  TEXT,                -- JSON, 仅 assistant 消息
    tool_result TEXT,                -- JSON, 仅 tool 消息
    created_at  INTEGER NOT NULL
);
```

### 6.3 记忆 (Memory)

```sql
CREATE TABLE memories (
    id          TEXT PRIMARY KEY,
    content     TEXT NOT NULL,
    embedding   BLOB NOT NULL,       -- float32 数组
    category    TEXT,                -- decision | pattern | convention | lesson
    file_path   TEXT,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);
```

### 6.4 配置 (Config)

```yaml
# convallaria.yaml
llm:
  provider: deepseek           # deepseek | openai | anthropic
  model: deepseek-chat
  max_tokens: 4096
  temperature: 0.0

agent:
  max_turns: 50
  workspace: "."               # 工作目录，文件操作范围

context:
  max_context_tokens: 64000
  compression_threshold: 0.8

tools:
  shell:
    enabled: true
    timeout: 120               # 秒
  file:
    allowed_paths: ["."]       # 允许读写的路径
  git:
    auto_commit: false

guardrails:
  dangerous_commands: true
  file_scope: true
  git_dangerous_ops: true

memory:
  vector_store: sqlite
  top_k: 5
  embedding_model: local       # local | remote
```

---

## 7. 凭据与分发设计

### 7.1 凭据存储

- **方案**：操作系统凭据管理
  - Windows：Windows Credential Manager（通过 `wincred` 或 `github.com/danieljoos/wincred`）
  - macOS：Keychain（通过 `security` 命令或 `github.com/keybase/go-keychain`）
  - Linux：Secret Service / D-Bus（通过 `github.com/zalando/go-keyring`）
- **跨平台库**：`github.com/zalando/go-keyring` 自动适配各平台
- **录入流程**：`convallaria init` → 引导选择 provider → 隐藏输入 Key → 写入凭据管理
- **查看**：`convallaria credential status` → 显示 `Provider: deepseek | Key: sk-****abcd`
- **更新**：`convallaria credential set --provider deepseek`
- **清除**：`convallaria credential clear --provider deepseek`

### 7.2 分发形态

- **形态**：Go 单文件二进制
- **构建**：`go build -o convallaria ./cmd/convallaria`
- **目标平台**：Windows (amd64)、macOS (amd64 + arm64)、Linux (amd64)
- **CI 构建**：GitHub Actions 自动构建 + 上传到 Release
- **安装**：下载二进制 → 放到 PATH → 运行 `convallaria init`
- **Key 配置**：首次运行 `convallaria init` 引导录入，或手动设置环境变量 `CONVALLARIA_API_KEY`

### 7.3 环境前提

- **Go 1.22+**：构建和运行必需，`go version` 确认
- **Git**：`git` 命令可用（Windows 下 Git Bash 或 WSL）
- **操作系统**：Windows (cmd/powershell)、macOS (sh)、Linux (sh) 均支持
- 工具实现需要做 OS 判断：shell 命令 Windows 用 `cmd /c`，Unix 用 `sh -c`；搜索用 Go 原生实现而非系统 grep

### 7.4 已知限制

- 文件操作沙箱基于路径匹配，非真正的 OS 级沙箱
- 嵌入模型如选本地，需额外下载模型文件
- 仅支持通过 OpenAI 兼容 API 的供应商（Anthropic 需适配层）
- Windows 下 shell 使用 `cmd /c`，与 Unix `sh -c` 行为有差异；搜索使用 Go 原生实现而非系统 grep

---

## 8. 技术选型与理由

| 维度 | 选择 | 理由 |
|------|------|------|
| 语言 | Go | 单文件编译分发、静态类型 + interface 天然支持 mock 注入、goroutine 适配 SSE 并发 |
| LLM 供应商 | DeepSeek（默认）+ OpenAI + Anthropic | OpenAI 兼容协议最通用，Provider 接口可切换 |
| 前端 | Material Design 3 + Open Design | 课程要求，Material Design 3 组件最全、适合工具型应用 |
| 通信 | SSE | 单向流式推送，Go 标准库原生支持，比 WebSocket 简单 |
| 存储 | SQLite | 零配置、嵌入式、单文件，无需额外数据库服务 |
| 嵌入模型 | 本地小模型（如 all-MiniLM-L6-v2） | 不依赖外部 API，离线可用 |
| 凭据存储 | OS 凭据管理（go-keyring） | 跨平台，安全级别高于 .env 明文 |
| 分发 | Go 单文件二进制 | 用户无需安装运行时，下载即用 |
| Open Design | Material Design 3 skill | 课程要求，组件丰富，适合 Web UI 型工具 |

---

## 9. 领域与机制设计

### 9.1 领域分析：Coding 场景的四类机制

| 机制 | Coding 场景特点 | 设计决策 |
|------|---------------|---------|
| **动作/工具** | 读写代码、执行 shell、运行测试、git 操作 | 6 个工具，统一 `Tool` 接口，注册表分发 |
| **客观反馈信号** | 编译错误、测试失败、lint 警告——都是确定性的、可解析的 | Build + Vet + Test 三层校验器，结构化回灌 |
| **危险动作** | 删除文件、危险 shell、git 破坏性操作 | 三层正则+路径+精确匹配，HITL 审批 |
| **记忆** | 项目约定、编码规范、历史决策、代码模式 | 规则文件全量加载 + 向量语义检索 |

### 9.2 重点维度：反馈闭环

**为什么选反馈闭环**：它是 coding agent 区别于通用 agent 的核心能力。一个 coding agent 之所以能写代码，不是因为它聪明，而是因为它能跑测试 → 看到失败 → 修正代码 → 再跑测试。这个"感知失败 → 自我修正"的闭环，就是 coding agent 的发动机。

**如何编码实现**：
- `BuildValidator`：执行 `go build`，解析编译错误输出，提取文件名、行号、列号、错误信息
- `VetValidator`：执行 `go vet`，解析静态分析警告
- `TestValidator`：执行 `go test -json`，解析 JSON 输出，提取失败用例和堆栈
- `FeedbackLoop`：聚合三个校验器，生成结构化反馈消息，回灌到 LLM 的 messages 中

**Mock 可测试性**：注入预设的编译错误/测试失败 → 断言 agent 下一步动作是修复代码而非继续生成新功能

### 9.3 机制必须是代码，不是提示词

| 机制 | 提示词版（不计入） | 代码版（计入） |
|------|------------------|--------------|
| 护栏 | "请勿执行 rm -rf" | `guardrail.Check(action)` 函数，正则匹配 → 拦截 |
| 反馈 | "请自行检查代码是否正确" | `BuildValidator.Validate()` 函数，执行编译 → 解析结果 → 回灌 |
| 记忆 | "请记住之前的约定" | `MemoryStore.Search()` 函数，向量检索 → 注入上下文 |

**判据**：移除真实 LLM、注入 mock LLM 后，每个机制仍能用确定性单元测试验证。

---

## 10. 验收标准

| 功能 | 验收标准 |
|------|---------|
| 主循环 | Mock LLM 返回 tool call → agent 执行工具 → 结果回灌 → 循环直到停机 |
| 反馈闭环 | 注入编译错误 → agent 尝试修复 → 再次编译通过 |
| 护栏 | 传入 `rm -rf /` → 护栏拦截 → 前端收到审批请求 |
| 记忆 | 写入记忆 → 重启 → 语义检索返回相关记忆 |
| 配置 | 修改 `convallaria.yaml` → 重启 → agent 行为按新配置 |
| 凭据 | `convallaria init` → 录入 Key → 查看状态不回显明文 → 清除后无法调用 LLM |
| 分发 | `go build` → 单文件二进制 → 拷贝到另一台机器 → 运行 `convallaria init` → 正常使用 |
| Web UI | 浏览器打开 → 输入任务 → 看到 SSE 流式输出 → 工具执行进度 → 最终结果 |
| 会话 | 创建会话 → 关闭浏览器 → 重新打开 → 会话列表显示历史 → 点击恢复 |
| 多模型 | 配置切换 provider → agent 调用不同 LLM API |
| 测试 | `go test ./...` 一键运行，包含 mock LLM 的确定性单测 |

---

## 11. 风险与未决问题

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| LLM 返回格式不稳定 | 解析失败，agent 卡住 | 错误恢复机制：重试 3 次 → 降级 |
| 上下文窗口溢出 | 丢失关键信息，决策质量下降 | Token 计数 + 自动摘要压缩 |
| 护栏误拦截 | 正常命令无法执行 | 用户可一键允许，支持永久允许规则 |
| 嵌入模型下载大 | 首次启动慢 | 可选禁用语义记忆，仅用规则文件 |
| Anthropic API 协议不兼容 | 多模型支持不完整 | 抽象层做协议适配，优先保证 OpenAI 兼容 |
| 前端开发工作量大 | 时间不够 | 使用 Open Design + Material Design 3 组件库加速 |

---

## 附录 A：目录结构

```
convallaria/
├── cmd/convallaria/main.go       # 入口
├── internal/
│   ├── agent/loop.go             # 主循环
│   ├── agent/loop_test.go        # Mock LLM 单测
│   ├── llm/provider.go           # Provider 接口
│   ├── llm/deepseek.go           # DeepSeek 实现
│   ├── llm/openai.go             # OpenAI 实现
│   ├── llm/anthropic.go          # Anthropic 实现
│   ├── llm/mock.go               # Mock 实现
│   ├── parser/parser.go          # 动作解析器
│   ├── tools/registry.go         # 工具注册表
│   ├── tools/file_reader.go
│   ├── tools/file_writer.go
│   ├── tools/shell_runner.go
│   ├── tools/searcher.go
│   ├── tools/test_runner.go
│   ├── tools/git_ops.go
│   ├── guardrail/guardrail.go    # 护栏
│   ├── guardrail/guardrail_test.go
│   ├── feedback/feedback.go      # 反馈闭环
│   ├── feedback/build_validator.go
│   ├── feedback/vet_validator.go
│   ├── feedback/test_validator.go
│   ├── feedback/feedback_test.go
│   ├── memory/store.go           # 记忆存储
│   ├── memory/embedder.go        # 嵌入向量
│   ├── memory/rules.go           # 规则文件加载
│   ├── context/manager.go        # 上下文窗口管理
│   ├── recovery/recovery.go      # 错误恢复
│   ├── session/manager.go        # 会话管理
│   ├── config/config.go          # 配置管理
│   ├── server/handler.go         # HTTP/SSE Handler
│   ├── server/sse.go             # SSE 事件推送
│   └── credential/credential.go  # 凭据管理
├── web/                          # 前端 (Material Design 3)
│   ├── index.html
│   ├── css/
│   ├── js/
│   └── components/
├── test/integration/             # 集成测试
├── docs/                         # 文档
├── .gitignore
├── .gitlab-ci.yml
├── go.mod
├── go.sum
├── README.md
├── SPEC.md                       # 本文件
├── PLAN.md
├── SPEC_PROCESS.md
├── AGENT_LOG.md
└── REFLECTION.md
```