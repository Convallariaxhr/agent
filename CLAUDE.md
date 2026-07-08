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

## 开发纪律

- 本项目是 AI4SE 期末项目（A · Coding Agent Harness），必须严格遵循课程要求
- 强制使用 Superpowers 七步工作流：brainstorming → writing-plans → using-git-worktrees → subagent-driven-development → test-driven-development → requesting-code-review → finishing-a-development-branch
- TDD 是硬性要求：先红、再绿、再重构，不可先写实现再补测试
- 凭据绝不硬编码、不提交 Git
- 核心机制必须是代码而非提示词，移除真实 LLM 后仍可用 mock 做确定性单测