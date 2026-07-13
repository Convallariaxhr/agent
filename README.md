# Convallaria — Coding Agent Harness

> AI4SE 期末项目（A 类）· Agent = LLM + Harness

一个不套壳的、透明可扩展的 Coding Agent 框架。LLM 只负责决策，harness 负责主循环、工具执行、治理护栏、反馈闭环、记忆管理。

## 快速开始

### 环境要求

- Go 1.22+
- (可选) DeepSeek / OpenAI / Anthropic API Key

### 安装运行

```bash
# 克隆仓库
git clone https://github.com/Convallariaxhr/agent.git
cd agent

# 运行（Mock 模式，无需 API Key）
go run ./cmd/convallaria/

# 使用 DeepSeek
# Windows
set DEEPSEEK_API_KEY=sk-your-key && go run ./cmd/convallaria/
# macOS / Linux
export DEEPSEEK_API_KEY=sk-your-key && go run ./cmd/convallaria/
```

打开浏览器访问 `http://localhost:8080`

## 切换 LLM 供应商

支持三种 API，**无需修改代码**，通过环境变量或配置文件切换：

| 供应商 | 环境变量 | 示例 |
|--------|---------|------|
| DeepSeek | `DEEPSEEK_API_KEY` | `$env:DEEPSEEK_API_KEY="sk-xxx"; go run ./cmd/convallaria/` |
| OpenAI | `CONVALLARIA_API_KEY` + `CONVALLARIA_PROVIDER` | `$env:CONVALLARIA_PROVIDER="openai"; $env:CONVALLARIA_API_KEY="sk-xxx"; go run ./cmd/convallaria/` |
| Anthropic (Claude) | `CONVALLARIA_API_KEY` + `CONVALLARIA_PROVIDER` | `$env:CONVALLARIA_PROVIDER="anthropic"; $env:CONVALLARIA_MODEL="claude-sonnet-4-20250514"; $env:CONVALLARIA_API_KEY="sk-ant-xxx"; go run ./cmd/convallaria/` |

或创建 `convallaria.yaml` 配置文件：

```yaml
llm:
  provider: anthropic           # deepseek / openai / anthropic
  model: claude-sonnet-4-20250514
```

然后只需设 Key 环境变量即可。详细配置见下方[配置](#配置)章节。

### Docker 部署

```bash
docker build -t convallaria .
docker run -p 8080:8080 -e DEEPSEEK_API_KEY=sk-your-key convallaria
```

### 构建二进制

```bash
# 当前平台
go build -o convallaria.exe ./cmd/convallaria/

# 全平台
make build-all
```

## 系统架构

```
浏览器 (Material Design 3 Web UI)
  ├── 对话面板（SSE 流式推送）
  ├── 文件浏览（交互式，支持导航和预览）
  ├── 会话列表（右键重命名）
  └── 配置面板
        ↓ SSE         ↑ HTTP
┌──────────────────────────────────────────┐
│              Go 后端 :8080                │
│  ┌────────┐ ┌────────┐ ┌──────────┐     │
│  │HTTP/SSE│ │Agent   │ │治理护栏   │     │
│  │Handler │ │主循环   │ │HITL 审批  │     │
│  └────────┘ └────────┘ └──────────┘     │
│  ┌────────┐ ┌────────┐ ┌──────────┐     │
│  │工具执行│ │反馈闭环│ │记忆系统   │     │
│  │6 tools │ │Build   │ │规则+SQLite│     │
│  └────────┘ │Vet+Test│ └──────────┘     │
│             └────────┘                   │
└──────────────────────────────────────────┘
        ↓ HTTP (OpenAI 兼容 API)
   🤖 DeepSeek / OpenAI / Anthropic
```

## 六个维度

| 维度 | 说明 |
|------|------|
| 决策封装 | 主循环：组织上下文→调用 LLM→解析动作→分发执行→回灌结果→停机判断 |
| 工具 | 6 个工具：file_read/write、shell_run、search、test_run、git |
| 记忆 | 规则文件 (CONVALLARIA.md) + SQLite 持久化 |
| 治理 | 三层护栏（危险命令/文件范围/Git 危险操作）+ HITL 人工审批 |
| 反馈闭环 | Build + Vet + Test 三层校验，结果回灌 LLM 驱动自我修正 |
| 配置 | convallaria.yaml + 环境变量 + CLI 参数 |

## 功能特性

### Web UI
- 铃兰主题 Material Design 3 界面
- SSE 流式对话，实时显示 LLM 输出
- 多会话管理，右键重命名
- 交互式文件浏览器：目录导航、文件内容预览
- 配置面板查看当前设置
- 危险操作 HITL 审批弹窗

### 后端
- 多 LLM Provider 支持：DeepSeek / OpenAI / Anthropic / Mock
- 会话 SQLite 持久化，重启不丢失
- 反馈闭环：Build → Vet → Test 三层自动校验
- 上下文窗口管理 + 错误恢复
- 凭据脱敏，环境变量注入

## 配置

```yaml
# convallaria.yaml
llm:
  provider: deepseek          # deepseek / openai / anthropic
  model: deepseek-chat
  max_tokens: 4096
  api_key_env: DEEPSEEK_API_KEY
agent:
  max_turns: 50
  workspace: .
guardrails:
  dangerous_commands: true
  file_scope: true
  git_dangerous_ops: true
```

## 项目规则

在项目根目录放置 `CONVALLARIA.md` 或 `CLAUDE.md`，agent 启动时自动加载并遵守其中的约定。

## 目录结构

```
.
├── cmd/convallaria/main.go          # CLI 入口
├── internal/
│   ├── agent/                       # Agent 主循环 + HITL 审批
│   ├── config/                      # 配置系统 (YAML + env)
│   ├── context/                     # 上下文窗口管理
│   ├── credential/                  # 凭据管理 (脱敏)
│   ├── feedback/                    # 反馈闭环 (Build/Vet/Test)
│   ├── guardrail/                   # 三层安全护栏
│   ├── llm/                         # LLM Provider (DeepSeek/OpenAI/Anthropic/Mock)
│   ├── memory/                      # 记忆系统 (规则 + SQLite)
│   ├── parser/                      # LLM 响应解析
│   ├── recovery/                    # 错误恢复
│   ├── server/                      # HTTP/SSE 服务器
│   ├── session/                     # 会话管理 (SQLite 持久化)
│   └── tools/                       # 6 个工具实现
├── web/                             # Material Design 3 Web UI
│   ├── index.html
│   ├── css/style.css
│   └── js/
│       ├── app.js
│       └── sse.js
├── .github/workflows/ci.yml         # GitHub Actions CI/CD
├── .gitlab-ci.yml                   # GitLab CI/CD
├── Dockerfile                       # 多阶段 Docker 构建
├── Makefile                         # 构建/测试/跨平台编译
├── SPEC.md                          # 完整设计文档（11 章）
├── docs/
│   ├── superpowers/plans/           # 13 Phase 实现计划
│   ├── AGENT_LOG.md                 # 实现过程日志
│   ├── DEMO.md                      # 核心机制演示
│   ├── REFLECTION.md                # Superpowers 方法论反思
│   └── SPEC_PROCESS.md              # 规约与设计过程
└── convallaria.db                   # SQLite 数据库（运行后自动生成）
```

## 安全边界

- **凭据**：API Key 通过环境变量传入，不存储在代码或配置文件中；日志输出自动脱敏
- **护栏**：危险命令正则拦截、文件操作限制在工作区范围内、Git 危险操作拦截
- **HITL**：拦截到危险操作时前端弹窗请求人工审批，支持"允许一次"和"拒绝"
- **编码**：SSE 数据使用 `json.Marshal` 安全编码，防止 JSON 注入
- **仓库**：提交前自查，确保 `.env`、真实凭据不被提交

## 已知限制

- 向量嵌入记忆检索尚未实现（当前使用关键词搜索）
- 凭据存储在内存中（非 OS keychain）
- 上下文压缩使用简单截断（非 LLM 摘要）
- 仅支持 Go 项目（Build/Vet/Test 校验器使用 go 工具链）

## 测试

```bash
# 运行所有测试（52 个，全部通过）
go test ./...

# 带详细输出
go test ./... -v
```

## CI/CD

Push 到 master 分支自动触发：
- GitHub Actions：单元测试 → 二进制构建 → Docker 镜像构建推送
- GitLab CI：单元测试 → 二进制构建 → Docker 镜像构建推送

## License

MIT