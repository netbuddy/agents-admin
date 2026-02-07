# Agent Admin 用户使用手册

> **版本**：v1.0 (MVP)
> **更新日期**：2026-02-06

## 简介

Agent Admin 是一个 **AI Agent 任务编排与可观测平台**，让你可以像运行 CI Job 一样运行和管理 AI Agent。

平台支持多种 Agent 类型（Qwen-Code、Gemini CLI、Claude Code、Codex CLI），提供统一的任务创建、执行监控、账号管理能力。

## 核心概念

| 概念 | 说明 |
|------|------|
| **Agent 类型** | 支持的 AI Agent 产品（如 Qwen-Code、Gemini CLI） |
| **账号** | Agent 的认证凭据，通过 OAuth / API Key 创建 |
| **实例** | 基于账号创建的运行环境（Docker 容器） |
| **任务** | 一次 Agent 执行的定义（Prompt + 配置） |
| **Run** | 任务的一次具体执行，包含事件流和结果 |
| **节点** | 运行 Agent 的物理/虚拟机器，由 NodeManager 管理 |
| **代理** | 网络代理配置（HTTP/SOCKS5），用于 Agent 访问外网 |

## 文档目录

1. **[快速入门](./01-quick-start.md)** — 部署系统并完成第一次任务执行
2. **[任务管理](./02-task-management.md)** — 创建、执行、监控任务
3. **[账号与实例管理](./03-account-instance.md)** — 配置 Agent 账号和运行实例
4. **[节点管理](./04-node-management.md)** — 管理执行节点
5. **[代理管理](./05-proxy-management.md)** — 配置网络代理
6. **[监控与运维](./06-monitoring.md)** — 系统监控和健康检查
7. **[TLS/HTTPS 安全通信](./07-tls-https.md)** — 配置 HTTPS 加密通信

## 系统架构概览

```
┌─────────────────────────────────────────────────────┐
│                    Web UI (Next.js)                  │
│  看板 │ 任务详情 │ 账号 │ 实例 │ 节点 │ 代理 │ 监控  │
└────────────────────┬────────────────────────────────┘
                     │ HTTP / WebSocket
┌────────────────────▼────────────────────────────────┐
│                  API Server (Go)                     │
│  Task │ Run │ Event │ Node │ Account │ Instance │ …  │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐          │
│  │ Scheduler│  │ EventGW  │  │ Metrics  │          │
│  └──────────┘  └──────────┘  └──────────┘          │
└────────┬──────────────┬──────────────┬──────────────┘
         │              │              │
    PostgreSQL        Redis        NodeManager
    (持久化)       (缓存/队列)    (Agent 执行)
```

## 技术栈

- **后端**：Go 1.22+、PostgreSQL、Redis
- **前端**：Next.js 14、React、TailwindCSS
- **部署**：Docker Compose / 单二进制（Go embed）
- **监控**：Prometheus + Grafana（可选）
