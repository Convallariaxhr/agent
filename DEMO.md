# DEMO.md — 核心机制演示

> 所有演示使用 Mock LLM，无需 API Key，一键运行。

## 运行演示

```bash
make test
# 或
go test ./... -v
```

## 演示内容

### 1. Agent 主循环：文本响应

**测试**：`TestAgent_TextResponse_ReturnsFinalReply`

演示 Agent 接收用户输入 → 调用 LLM → 返回文本回复的完整流程。

```
用户: "Write a hello world program"
Agent 主循环:
  1. 构建上下文 [system prompt + user message]
  2. 调用 LLM (Mock)
  3. LLM 返回纯文本 → 停机
  4. 输出: "Hello! I can help you write code."
```

### 2. Agent 主循环：工具调用

**测试**：`TestAgent_ToolCall_ExecutesAndReturnsResult`

演示 Agent 接收 LLM 的 tool call → 执行工具 → 结果回灌 → 再次调用 LLM。

```
用户: "Create a file called hello.txt"
Agent 主循环 Turn 1:
  1. LLM 返回 tool_call: file_write("hello.txt", "hello world")
  2. 执行 FileWriter → 创建文件
  3. 结果回灌: "File written: hello.txt"
Agent 主循环 Turn 2:
  4. LLM 返回文本: "Done! I've created hello.txt."
  5. 停机
```

### 3. 治理护栏：危险命令拦截

**测试**：`TestAgent_GuardrailBlocksDangerousAction`

演示危险命令被三层护栏拦截 → 注入 BLOCKED 消息 → LLM 响应。

```
用户: "Delete everything"
LLM 返回 tool_call: shell_run("rm -rf /")
护栏检查:
  L1 命令检查: 正则匹配 "rm -rf /" → 拦截
  注入: "BLOCKED: dangerous_command - Dangerous command blocked: rm -rf /"
LLM 回复: "Sorry, I cannot execute that command."
```

### 4. 反馈闭环：Build 错误检测

**测试**：`TestAgent_FeedbackLoop_DetectsBuildError`

演示 Agent 写代码 → 反馈闭环检测编译错误 → 错误回灌 → LLM 修正。

```
Agent 写入 broken.go (含 undefined 变量)
反馈闭环触发:
  BuildValidator: go build → 失败
  错误: "broken.go:3:2: undefined: undefined"
  错误回灌 LLM (结构化 JSON)
LLM 收到反馈 → 修正代码 → 写入修正后的文件
```

### 5. 最大轮次限制

**测试**：`TestAgent_MaxTurnsExceeded`

演示 Agent 在达到最大轮次后停止，防止无限循环。

```
Agent 连续 3 轮 tool call
第 4 轮: 超过 MaxTurns=3 → 返回 ErrMaxTurnsExceeded
```

## 核心机制验证清单

| 机制 | 测试 | 验证内容 |
|------|------|---------|
| 主循环 | TextResponse | 上下文构建 → LLM 调用 → 停机 |
| 工具执行 | ToolCall | 解析 tool call → 执行 → 回灌 |
| 护栏 | GuardrailBlocks | 危险命令拦截 → HITL 审批 |
| 反馈闭环 | FeedbackLoop | 编译错误 → 回灌 → 修正 |
| 边界条件 | MaxTurns | 超限停止 |
| 上下文管理 | EstimateTokens | Token 估算 + 压缩 |
| 错误恢复 | RetryOnParse | 重试 → 纠错 → 降级 |
| 记忆系统 | InsertAndSearch | 规则加载 + 关键词检索 |
| 会话管理 | CreateAndGet | CRUD + 导出 |
| SSE 服务器 | ChatEndpoint | HTTP/SSE 流式推送 |

## 运行结果

```bash
$ make test
=== RUN   TestAgent_TextResponse_ReturnsFinalReply
--- PASS: TestAgent_TextResponse_ReturnsFinalReply (0.00s)
=== RUN   TestAgent_ToolCall_ExecutesAndReturnsResult
--- PASS: TestAgent_ToolCall_ExecutesAndReturnsResult (0.02s)
=== RUN   TestAgent_GuardrailBlocksDangerousAction
--- PASS: TestAgent_GuardrailBlocksDangerousAction (0.00s)
=== RUN   TestAgent_FeedbackLoop_DetectsBuildError
--- PASS: TestAgent_FeedbackLoop_DetectsBuildError (0.00s)
=== RUN   TestAgent_MaxTurnsExceeded
--- PASS: TestAgent_MaxTurnsExceeded (0.04s)
...
PASS — 52 tests, 14 packages
```