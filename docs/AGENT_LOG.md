# AGENT_LOG.md — Convallaria Coding Agent Harness

按时间顺序记录实现过程中的关键节点。

## 2026-07-08

### Phase 0: 规约与计划（上午）

| 时间 | 事件 | 技能/Agent | 产出 |
|------|------|-----------|------|
| 11:14 | 初始化仓库，配置 CLAUDE.md | - | `777a0e2` |
| 11:49 | 完成 SPEC.md 设计文档 | brainstorming | `0a545e5` |
| 12:12 | 完成 PLAN.md 13 个 Phase 实现计划 | writing-plans | `51159e7` |
| 14:39 | 冷启动验证：另一个 agent 发现 10 个问题 | 独立 agent | `4ef415b` |
| 14:51 | 冷启动第二轮：PowerShell 兼容 + GOPROXY | 独立 agent | `6c71639` |
| 14:57 | 更新 CLAUDE.md 进度 | - | `e9ec3a2` |

### Phase 1: 项目脚手架（下午）

| 时间 | Task | 实现方式 | Commit |
|------|------|---------|--------|
| 15:11 | 1.1 Go module 初始化 | TDD 手动 | `6bc06d6` |
| 15:23 | 1.2 LLM Provider 接口 + Mock | TDD 手动 | `437c9f2` |
| 15:27 | MockProvider ChatSync 修复 | review 修复 | `5728684` |
| 15:33 | MockProvider code review 修复 | review 修复 | `001ac7d` |

### Phase 2: 配置与凭据（下午）

| 时间 | Task | 实现方式 | Commit |
|------|------|---------|--------|
| 15:43 | 2.1 配置系统 (YAML + env) | TDD 手动 | `7889f4e` |
| 16:17 | 2.2 Credential 凭据管理 | TDD 手动 | `e864579` |

### Phase 3-4: 解析器 + 工具 + 护栏（下午，批量推进）

| 时间 | Task | 实现方式 | Commit |
|------|------|---------|--------|
| 16:18 | 3.1 Action Parser | TDD 手动 | `c99d266` |
| 16:19 | 3.2 工具注册表 + 6 个工具 | TDD 手动 | `1c0957d` |
| 16:21 | 4.1 三层护栏 | TDD 手动 | `1a2e207` |

### Phase 5-6: 反馈闭环 + Agent 主循环（下午）

| 时间 | Task | 实现方式 | Commit |
|------|------|---------|--------|
| 16:23 | 5.1 反馈闭环 Build/Vet/Test | TDD 手动 | `3e1dcac` |
| 16:25 | 6.1 Agent 主循环 + Mock 集成 | TDD 手动 | `d3af108` |

**人工干预**：`TestAgent_ToolCall_ExecutesAndReturnsResult` 在 Windows 上因路径反斜杠 JSON 转义失败，用 `filepath.ToSlash()` 修复。`TestBuildValidator_ValidGoFile` 因临时目录缺少 `go.mod` 失败，添加 `initGoModule` 辅助函数。

### Phase 7-8: 上下文 + 恢复 + 记忆（下午）

| 时间 | Task | 实现方式 | Commit |
|------|------|---------|--------|
| 16:28 | 7.1 上下文窗口管理 + 7.2 错误恢复 | TDD 手动 | `8b89014` |
| 16:30 | 8.1 记忆系统（规则 + 关键词搜索） | TDD 手动 | `46a4e44` |

**人工干预**：`memory/store.go` 中 `scored` 类型名与变量名冲突，重命名为 `scoredEntry`。

### Phase 9-10: 会话 + 服务器（下午）

| 时间 | Task | 实现方式 | Commit |
|------|------|---------|--------|
| 16:31 | 9.1 会话管理 (CRUD + 导出) | TDD 手动 | `51d1ffe` |
| 16:34 | 10.1 HTTP/SSE 服务器 | TDD 手动 | `f42cfe9` |

### Phase 11-12: Web UI + CLI（下午）

| 时间 | Task | 实现方式 | Commit |
|------|------|---------|--------|
| 16:46 | 11.1 Web UI + 12.1 CLI 入口 | frontend-design skill | `213b6e0` |
| 17:01 | Web UI 初始化修复 + mock 响应 | 调试修复 | `b0a55fd` |
| 17:01 | 更新 CLAUDE.md 进度 | - | `a7a3015` |

**人工干预**：`DOMContentLoaded` 事件可能已触发导致 app 未初始化，改为检查 `document.readyState`。Mock 无响应导致 UI 无可见反馈，添加 demo 响应。

### Code Review 阶段（下午）

| 时间 | 事件 | 技能 | Commit |
|------|------|------|--------|
| 17:15 | Code review 发现 16 个问题（4 Critical + 5 Important + 7 Minor） | requesting-code-review (general-purpose agent) | - |
| 17:15 | 修复全部 9 个 Critical/Important 问题 | 手动 | `9e33029` |

**关键修复**：
- Agent 竞态条件：`messages` 从 struct 字段改为局部变量
- SSE JSON 注入：用 `json.Marshal` 替代 `fmt.Sprintf`
- Shell/Git 缺工作目录：Agent 自动注入 `workspace` 参数
- CORS preflight：添加 OPTIONS handler
- 优雅关闭：`signal.Notify` + `http.Server.Shutdown`

### 增强阶段（下午）

| 时间 | Task | 实现方式 | Commit |
|------|------|---------|--------|
| 17:37 | DeepSeek LLM Provider | TDD 手动 | `3cdac3f` |
| 17:46 | SQLite 持久化（会话 + 记忆） | 手动重构 | `da04eef` |
| 17:55 | HITL 审批弹窗（前后端） | 手动 | `ea88223` |
| 18:01 | 文件浏览 + 配置面板 | 手动 | `d21d3f7` |

**人工干预**：SQLite 集成时重构 `session.Manager` 为 `Store` 接口驱动，`memoryStore` 作为默认实现，`SQLiteStore` 可选替换。HITL 审批流程使用 channel 实现异步等待。

## 统计

- **总 commits**: 24
- **Superpowers 技能使用**: brainstorming, writing-plans, using-git-worktrees, subagent-driven-development, test-driven-development, requesting-code-review, frontend-design, finishing-a-development-branch
- **测试数量**: 52 个（全部通过）
- **Go 包数量**: 14 个
- **人工干预次数**: 6 次（路径兼容、变量冲突、初始化时机、SQLite 重构、HITL 架构、mock 响应）

## 2026-07-09

### 合并与部署（下午）

| 时间 | 事件 | 产出 |
|------|------|------|
| 19:17 | 合并 worktree-phase-1-scaffold 到 master | `ccaebe2` |
| 19:25 | Docker 构建推送加入 CI/CD | `b837122` |
| 19:28 | 推送 master 到 GitHub | `8d29914` |

### Bug 修复（下午）

| 时间 | 问题 | 修复 |
|------|------|------|
| 19:35 | 会话 ID 重启后碰撞，导致消息丢失 | `nextID` 初始化从数据库扫描最大 ID，冲突时自动递增重试 |
| 19:38 | `loadMessages` JSON 字段大小写不匹配 (`m.Role` vs `m.role`) | 改为小写 `m.role` / `m.content` |
| 19:40 | 聊天区域无法滚动 | Flex 父容器添加 `min-height: 0` |
| 19:41 | 浏览器缓存旧 JS/CSS | 添加版本号 `?v=N` 缓存破坏 |

### 功能增强（下午）

| 时间 | 功能 | 说明 |
|------|------|------|
| 19:44 | 铃兰 Logo | 替换侧边栏和欢迎页图标，Material Design 3 风格 |
| 19:50 | 交互式文件浏览器 | 目录导航、文件内容预览、面包屑路径 |
| 19:55 | 会话重命名 | 右键菜单 → 内联编辑，SQLite 持久化 |

## 最终统计

- **总 commits**: 30+
- **测试数量**: 52 个（全部通过）
- **Go 包数量**: 14 个
- **多 Provider**: DeepSeek / OpenAI / Anthropic / Mock
- **部署**: Docker + GitHub Actions + GitLab CI