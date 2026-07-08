# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Convallaria** — 一个不套壳的 Coding Agent Harness，AI4SE 期末项目（A 类）。

核心等式：Agent = LLM + Harness。LLM（DeepSeek）只负责决策下一步动作，harness 负责主循环、工具执行、治理护栏、反馈闭环、记忆管理。

## 技术栈

| 维度 | 选择 |
|---|---|
| 语言 | Go |
| LLM | DeepSeek（OpenAI 兼容 API） |
| 前端 | Material Design 3 + Open Design |
| 通信 | SSE（Server-Sent Events）流式推送 |
| 分发 | Go 单文件二进制 |
| 记忆 | 规则文件 + 向量检索（SQLite） |

## 系统架构

```
浏览器 (Material Design 3 Web UI)
  ├── 💬 对话面板
  ├── 📁 文件浏览
  └── ⚙️ 配置面板
        ↓ SSE (流式推送)   ↑ HTTP (用户请求)
┌────────────────────────────────────────────┐
│            Go 后端 :8080                    │
│  ┌──────────┐ ┌──────────┐ ┌───────────┐  │
│  │HTTP/SSE  │ │Agent     │ │治理护栏    │  │
│  │Handler   │ │主循环     │ │危险命令拦截│  │
│  └──────────┘ └──────────┘ │HITL 审批  │  │
│  ┌──────────┐ ┌──────────┐ └───────────┘  │
│  │工具执行器 │ │反馈闭环   │ ┌───────────┐ │
│  │文件/shell │ │编译+测试  │ │记忆系统    │ │
│  │git/搜索   │ │结果回灌   │ │规则+向量   │ │
│  └──────────┘ └──────────┘ └───────────┘  │
└────────────────────────────────────────────┘
        ↓ HTTP (OpenAI 兼容 API)   ↑
        🤖 DeepSeek API
```

## 六个维度（反馈闭环为重点）

- **决策封装**：主循环组织上下文→调用 LLM→解析动作→分发执行→回灌结果→停机判断
- **工具**：读写文件、shell 执行、搜索（grep/glob）、测试运行、git 操作
- **记忆**：规则文件 + 向量检索（SQLite），跨会话持久化
- **治理**：危险命令拦截、文件范围限制、git 危险操作拦截、HITL 人工审批
- **反馈闭环**（重点）：编译检查 + 测试执行，确定性结果回灌给 LLM 驱动自我修正
- **配置**：convallaria.yaml + 环境变量 + CLI 参数 + 项目规则文件

## 当前进度

**Superpowers 七步工作流进度：**

| 步骤 | 状态 | 产出 |
|------|------|------|
| 1. brainstorming | ✅ 完成 | 完整设计决策 |
| 2. writing-plans | ✅ 完成 | `docs/superpowers/plans/2026-07-08-convallaria-implementation.md` |
| 冷启动验证 | ✅ 完成 | 两轮验证，修复 13 个问题 |
| 3. using-git-worktrees | ✅ 完成 | worktree `phase-1-scaffold` 已创建 |
| 4. subagent-driven-development | 🔜 下一步 | 按 PLAN Phase 1 开始实现 |
| 5. test-driven-development | 🔜 | |
| 6. requesting-code-review | 🔜 | |
| 7. finishing-a-development-branch | 🔜 | |

**下一步**：使用 `subagent-driven-development` 按 PLAN.md 的 Phase 1 开始实现，从 Task 1.1（Go module 初始化）开始。

## 关键文件

- `SPEC.md` — 完整设计文档（11 章 + 附录）
- `docs/superpowers/plans/2026-07-08-convallaria-implementation.md` — 13 个 Phase、30+ 个 Task 的实现计划
- `.gitignore` — 已配置，忽略 `.superpowers/`、`.env`、二进制文件

## 仓库信息

- GitHub：`https://github.com/Convallariaxhr/agent.git`（公开仓库）
- 本地路径：`D:\agent`
- 当前 worktree：`D:\agent\.claude\worktrees\phase-1-scaffold`（分支 `worktree-phase-1-scaffold`）
- 推送需要代理（`127.0.0.1:7890`）

## 冷启动验证发现的关键修复

以下问题已在 SPEC 和 PLAN 中修复，实现时注意：
1. **Windows 兼容**：shell_runner 用 `cmd /c`（Windows）或 `sh -c`（Unix）；searcher 用 Go 原生实现而非系统 grep
2. **MaskKey bug**：`key[:3] + "****" + key[len(key)-4:]`，不要多加 `-`
3. **MemoryStore ID**：用 `fmt.Sprintf("mem_%d", id)` 而非 `rune('0'+id)`
4. **反馈闭环**：每个 turn 结束后只跑一次，不是每个文件写完都跑
5. **MockProvider goroutine**：所有 `ch <-` 都要包裹 `select { case <-ctx.Done(): }`
6. **目录创建**：PowerShell 不支持 bash brace expansion，需用 `New-Item` 逐条创建
7. **GOPROXY**：国内可能需 `GOPROXY=https://goproxy.cn,direct`

## 关键设计决策

- 语言：Go 1.22+
- 重点维度：反馈闭环（Build + Vet + Test 三层校验器）
- 前端：Material Design 3 + Open Design
- 通信：SSE（Server-Sent Events）
- 分发：Go 单文件二进制
- 记忆：规则文件 + 向量检索（SQLite）
- 护栏：三层（危险命令 + 文件范围 + Git 危险操作）+ HITL 审批
- 多模型：Provider 接口支持 DeepSeek/OpenAI/Anthropic/Mock
- 新增模块：上下文窗口管理、错误恢复、流式工具输出、会话管理