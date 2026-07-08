# Convallaria — Coding Agent Harness

> AI4SE 期末项目（A 类）· Agent = LLM + Harness

一个不套壳的、透明可扩展的 Coding Agent 框架。LLM 只负责决策，harness 负责主循环、工具执行、治理护栏、反馈闭环、记忆管理。

## 快速开始

### 环境要求

- Go 1.22+
- (可选) DeepSeek API Key

### 安装运行

```bash
# 克隆仓库
git clone https://github.com/Convallariaxhr/agent.git
cd agent

# 运行（Mock 模式，无需 API Key）
go run ./cmd/convallaria/

# 使用 DeepSeek
set DEEPSEEK_API_KEY=sk-your-key   # Windows
export DEEPSEEK_API_KEY=sk-your-key  # macOS/Linux
go run ./cmd/convallaria/
```

打开浏览器访问 `http://localhost:8080`

### 构建二进制

```bash
go build -o convallaria.exe ./cmd/convallaria/
./convallaria.exe -port 8080
```

## 系统架构

```
浏览器 (Material Design 3 Web UI)
  ├── 对话面板（SSE 流式推送）
  ├── 文件浏览
  ├── 会话列表
  └── 配置面板
        ↓ SSE         ↑ HTTP
┌──────────────────────────────────────┐
│              Go 后端 :8080            │
│  ┌────────┐ ┌────────┐ ┌──────────┐ │
│  │HTTP/SSE│ │Agent   │ │治理护栏   │ │
│  │Handler │ │主循环   │ │HITL 审批  │ │
│  └────────┘ └────────┘ └──────────┘ │
│  ┌────────┐ ┌────────┐ ┌──────────┐ │
│  │工具执行│ │反馈闭环│ │记忆系统   │ │
│  │6 tools │ │Build   │ │规则+SQLite│ │
│  └────────┘ │Vet+Test│ └──────────┘ │
│             └────────┘               │
└──────────────────────────────────────┘
        ↓ HTTP
   🤖 DeepSeek API
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

## 配置

```yaml
# convallaria.yaml
llm:
  provider: deepseek
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
├── cmd/convallaria/main.go      # CLI 入口
├── internal/
│   ├── agent/                   # Agent 主循环 + 审批
│   ├── config/                  # 配置系统
│   ├── context/                 # 上下文窗口管理
│   ├── credential/              # 凭据管理
│   ├── feedback/                # 反馈闭环 (Build/Vet/Test)
│   ├── guardrail/               # 三层护栏
│   ├── llm/                     # LLM Provider 接口 + DeepSeek + Mock
│   ├── memory/                  # 记忆系统 (规则 + SQLite)
│   ├── parser/                  # LLM 响应解析
│   ├── recovery/                # 错误恢复
│   ├── server/                  # HTTP/SSE 服务器
│   ├── session/                 # 会话管理 (SQLite)
│   └── tools/                   # 6 个工具实现
├── web/                         # Material Design 3 Web UI
│   ├── index.html
│   ├── css/style.css
│   └── js/
│       ├── app.js
│       └── sse.js
├── SPEC.md                      # 设计文档
├── PLAN.md                      # 实现计划
└── convallaria.db               # SQLite 数据库（运行后自动生成）
```

## 安全边界

- **凭据**：API Key 通过环境变量 `DEEPSEEK_API_KEY` 传入，不存储在代码或配置文件中
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
# 运行所有测试
go test ./...

# 带详细输出
go test ./... -v
```

## License

MIT