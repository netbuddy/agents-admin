# Agent Kanban

> AI Agent 任务编排与可观测平台

[![CI](https://github.com/your-org/agents-admin/actions/workflows/ci.yml/badge.svg)](https://github.com/your-org/agents-admin/actions/workflows/ci.yml)

## 概述

Agent Kanban 是一个分布式容器化 AI Agent 编排与看板监控系统，支持多种 AI Agent CLI（Claude Code、Gemini CLI、Codex）的统一管理和实时监控。

## 核心能力

- **统一编排**：通过标准化 TaskSpec 管理不同 Agent 的任务
- **实时监控**：Event-First 架构，WebSocket 实时推送事件流
- **安全隔离**：三层防御（容器隔离、CLI 权限、平台熔断）
- **可扩展**：Driver 抽象层，轻松接入新的 Agent CLI

## 快速开始

### 前置要求

- Docker & Docker Compose
- Go 1.22+（开发）

### 启动开发环境

```bash
# 克隆仓库
git clone <repo-url>
cd agents-admin

# 启动基础设施（PostgreSQL、Redis、MinIO）
make dev-up

# 查看服务状态
docker compose -f deployments/docker-compose.yml ps
```

### 访问服务

| 服务 | 地址 | 说明 |
|------|------|------|
| API Server | http://localhost:8080 | REST API |
| MinIO Console | http://localhost:9001 | 对象存储管理 |
| PostgreSQL | localhost:5432 | 数据库 |
| Redis | localhost:6379 | 事件流 |

## 架构

```
┌─────────────────────────────────────────────────────────────┐
│                      Control Plane                          │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────────────┐│
│  │ Web UI  │  │   API   │  │Scheduler│  │  Event Gateway  ││
│  └────┬────┘  └────┬────┘  └────┬────┘  └────────┬────────┘│
│       │            │            │                 │         │
│       └────────────┴────────────┴─────────────────┘         │
│                           │                                  │
│              ┌────────────┴────────────┐                    │
│              │     PostgreSQL + Redis   │                    │
│              └──────────────────────────┘                    │
└─────────────────────────────────────────────────────────────┘
                            │
            ┌───────────────┼───────────────┐
            │               │               │
     ┌──────▼──────┐ ┌──────▼──────┐ ┌──────▼──────┐
     │ Node Agent  │ │ Node Agent  │ │ Node Agent  │
     │  (Worker)   │ │  (Worker)   │ │  (Worker)   │
     └──────┬──────┘ └──────┬──────┘ └──────┬──────┘
            │               │               │
     ┌──────▼──────┐ ┌──────▼──────┐ ┌──────▼──────┐
     │   Runner    │ │   Runner    │ │   Runner    │
     │ (Container) │ │ (Container) │ │ (Container) │
     └─────────────┘ └─────────────┘ └─────────────┘
```

## 文档

- [设计文档](./docs/design/README.md)
- [API 参考](./docs/api.md)（TODO）
- [部署指南](./docs/deployment.md)（TODO）

## 开发

```bash
# 运行测试
make test

# 代码检查
make lint

# 构建
make build
```

详见 [CONTRIBUTING.md](./CONTRIBUTING.md)

## 许可证

MIT
