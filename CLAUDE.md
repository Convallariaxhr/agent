# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a new/empty project workspace. No code, build system, or configuration has been established yet.

## Getting Started

- No build tools, package managers, or frameworks have been configured.
- No git repository has been initialized.
- When starting development, initialize the project with the appropriate tooling for the chosen tech stack.

## 开发纪律

- 本项目是 AI4SE 期末项目（A · Coding Agent Harness），必须严格遵循课程要求
- 强制使用 Superpowers 七步工作流：brainstorming → writing-plans → using-git-worktrees → subagent-driven-development → test-driven-development → requesting-code-review → finishing-a-development-branch
- TDD 是硬性要求：先红、再绿、再重构，不可先写实现再补测试
- 凭据绝不硬编码、不提交 Git
- 核心机制必须是代码而非提示词，移除真实 LLM 后仍可用 mock 做确定性单测