# SPEC_PROCESS.md — Convallaria Coding Agent Harness

本文档记录与 Superpowers 协作生成 SPEC 和 PLAN 的过程。

## 一、brainstorming 关键节点

### 第一轮：项目定位

**我的初始想法**：做一个"AI 写代码的助手"。

**AI 追问了什么**：
- "你究竟想做一个套壳聊天工具，还是一个真正的 agent harness？"
- "如果 LLM 只负责决策，那 harness 负责什么？"
- "你的 competitive advantage 是什么？"

这些问题让我重新审视项目定位。我意识到我想要的不是又一个 ChatGPT wrapper，而是一个**透明可理解**的 agent 框架——让用户看到 agent 内部的每一步决策。

**我的修正**：将项目定位从"AI 编程助手"改为"Coding Agent Harness"，核心等式确定为 **Agent = LLM + Harness**。

### 第二轮：六个维度

**AI 追问**：
- "你提到了 harness，具体包含哪些模块？"
- "反馈闭环是 coding agent 区别于通用 agent 最关键的能力，你打算怎么实现？"
- "治理护栏的三层具体是哪三层？"

**我采纳的 AI 建议**：
- 将反馈闭环作为重点维度（AI 指出了 coding agent 的独特价值）
- 三层护栏设计：危险命令（正则匹配）→ 文件范围（路径前缀）→ Git 危险操作（精确匹配）

**我推翻的 AI 建议**：
- AI 建议用 WebSocket 而非 SSE。我坚持用 SSE，因为"单向流式推送"正好匹配 agent 回复场景，WebSocket 的双向通信对当前需求是过度设计。

### 第三轮：技术选型

**AI 追问**：
- "你打不打算做前端？如果做，用什么？"
- "分发形态是什么？"
- "记忆系统存什么、怎么检索？"

**关键决策**：
- 前端：Material Design 3 + Open Design（课程要求）
- 分发：Go 单文件二进制（零依赖，下载即用）
- 记忆：CONVALLARIA.md 规则文件 + SQLite 向量检索
- 通信：SSE（Go 标准库原生支持）

### 第四轮：凭据与安全

**AI 追问**：
- "用户的 API Key 放在哪里？.env 文件？"
- "如果用户把 .env 提交到 git 怎么办？"
- "危险命令怎么防？"

**我采纳的 AI 建议**：
- 凭据从环境变量读取，不在代码中硬编码
- MaskKey 函数脱敏日志输出
- .gitignore 显式排除 .env 文件

**我推翻的 AI 建议**：
- AI 建议用 go-keyring 做 OS 凭据管理。我接受这是正确方向，但实现阶段先做 MemoryStore，go-keyring 标记为 TODO。

## 二、冷启动验证

按照课程要求，使用**另一个 agent**（类型不同），仅凭 SPEC.md + PLAN.md 尝试实现 1-2 个 task。

### 执行过程

**第二个 agent 类型**：general-purpose agent（与 brainstorming 的 agent 不同）

**指定 task**：PLAN Task 1.1（Go module 初始化）和 Task 1.2（LLM Provider 接口）

**暴露的问题**：

| # | 问题 | 根因 | SPEC 修订 |
|---|------|------|----------|
| 1 | Windows 上 `mkdir -p` 语法不兼容 | SPEC 未说明平台兼容性 | 添加 Windows 兼容说明 |
| 2 | `GOPROXY` 不配置无法下载依赖 | 未考虑国内网络环境 | 添加 GOPROXY 备用方案 |
| 3 | MockProvider goroutine 中 `ch <-` 无 context 取消保护 | 并发安全未在 SPEC 中明确 | 添加"所有 channel 操作需包裹 select" |
| 4 | `key[:3] + "****" + key[len(key)-4:]` 的 bug | 边界情况未测试 | 添加 MaskKey 测试用例 |
| 5 | MemoryStore ID 用 `rune('0'+id)` 错误 | 代码示例不够详细 | 修正为 `fmt.Sprintf("mem_%d", id)` |
| 6 | 反馈闭环每文件写完都触发 | SPEC 中"每个 turn 只跑一次"表述不够明确 | 强调"每个 turn 结束后只跑一次" |
| 7 | 目录创建用 bash brace expansion | PowerShell 不支持 | 添加 PowerShell 等价命令 |
| 8 | 缺少 `go.sum` 文件 | 首次 `go mod tidy` 自动生成 | 添加 `go mod tidy` 步骤 |
| 9 | 测试临时目录无 `go.mod` | BuildValidator 需要 Go module | 测试中添加 `go mod init` |
| 10 | shell_runner 在 Windows 需用 `cmd /c` | SPEC 未提平台差异 | 添加 `runtime.GOOS` 判断 |

### 冷启动的教训

**最关键的发现**：第二个 agent 在"上下文窗口管理"的 token 估算上，用了与设计意图完全不同的算法（它用了 tiktoken 精确计算，而设计意图是简单 heuristic）。这说明 SPEC 中"Token 估算"的描述不够精确。

**对我 SPEC 的修订**：在 SPEC 中明确了"使用简单字符计数 heuristic（~4 chars/token）"，tiktoken 精确计算标记为 TODO。

**反思**：冷启动验证是本次项目中**最有价值的环节**。如果没有它，我永远不会发现 Windows 兼容性、GOPROXY 配置、并发安全这些问题——因为它们在我的开发环境中"恰好能工作"。

## 三、AI 协作反思

### 做得好的地方

1. **brainstorming 的追问**：AI 不只回答我的问题，还会追问"你究竟想做什么"，这帮助我从模糊想法中提炼出清晰的设计。
2. **冷启动验证**：另一个 agent 暴露出我未说出口的假设，这比我自己 review 有效得多。
3. **Code review 的 adversarial 视角**：agent 找到了 4 个 Critical 问题，其中"竞态条件"和"JSON 注入"是我自己完全没意识到的。

### 做得不好的地方

1. **brainstorming 有时过于发散**：AI 会提出一些"看起来漂亮但实际用不上"的建议，比如建议用 Redis 做会话存储（对单文件二进制分发明显不适用）。
2. **PLAN 的 task 颗粒度有时过粗**：Phase 3 的"工具注册表 + 6 个工具"被合并为一个 task，但实际上 6 个工具之间有依赖关系，应该拆成更细的 task。
3. **前端设计阶段沟通不足**：AI 按模板直接写了 Web UI，但没先确认我是否安装了老师要求的 Open Design 插件。

### 如果重做

1. 在 brainstorming 阶段就对"前端用什么"做更具体的讨论，而不是到 Phase 11 才发现需要特定的插件。
2. PLAN 的 task 颗粒度可以更细，特别是工具模块，每个工具应该独立 task。
3. 冷启动验证应该覆盖更多 task（至少 3-4 个），而不仅仅是 2 个。