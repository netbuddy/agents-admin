# API 规范与文档

本目录包含 OpenAPI 3.0 规范、JSON Schema 定义和代码生成配置。

## 在线文档

API Server 启动后，访问 `/docs/` 可查看交互式 Swagger UI 文档。

```
https://localhost:8080/docs/
```

## 目录结构

```
api/
├── openapi/                   # OpenAPI 3.0 规范（按领域拆分）
│   ├── openapi.yaml           # 主入口文件（引用所有领域）
│   ├── common.yaml            # 公共定义（参数、响应、基础模型）
│   ├── health.yaml            # 健康检查
│   ├── auth.yaml              # 用户认证（注册、登录、令牌刷新）
│   ├── tasks.yaml             # 任务管理
│   ├── runs.yaml              # 执行管理
│   ├── events.yaml            # 事件管理
│   ├── nodes.yaml             # 节点管理（心跳、部署、环境配置）
│   ├── accounts.yaml          # 账号管理（Agent 类型、Volume 归档）
│   ├── operations.yaml        # 系统操作（Operation/Action 统一模型）
│   ├── proxies.yaml           # 代理管理
│   ├── instances.yaml         # 容器实例管理
│   ├── terminals.yaml         # 终端会话管理
│   ├── templates.yaml         # 模板管理（任务/Agent 模板、Skills、MCP、安全策略）
│   ├── hitl.yaml              # 人机协作（审批、反馈、干预、确认）
│   ├── monitor.yaml           # 监控（工作流、统计）
│   ├── sysconfig.yaml         # 系统配置
│   └── bundled.yaml           # 合并后的完整规范（自动生成，勿手动编辑）
├── codegen/                   # oapi-codegen 代码生成配置
│   ├── models.yaml            # 模型生成配置
│   ├── server.yaml            # 服务接口生成配置
│   └── spec.yaml              # 嵌入规范生成配置
├── generated/                 # 生成的 Go 代码（部分 handler 仍在使用）
│   └── go/
│       ├── models.gen.go      # 模型结构体
│       ├── server.gen.go      # 服务接口
│       └── spec.gen.go        # 嵌入规范
├── schemas/                   # JSON Schema（独立于 OpenAPI，用于验证）
│   ├── task_spec.v0.json      # 任务规格定义
│   └── canonical_event.v0.json # 统一事件格式
└── README.md
```

## API 领域概览

| 领域 | 文件 | 路由数 | 说明 |
|------|------|--------|------|
| Health | health.yaml | 1 | 服务健康检查 |
| Auth | auth.yaml | 5 | JWT 认证、Cookie 自动设置 |
| Tasks | tasks.yaml | 5 | 任务 CRUD、子任务、任务树、上下文 |
| Runs | runs.yaml | 5 | 执行 CRUD、取消 |
| Events | events.yaml | 2 | 事件查询、批量上报 |
| Nodes | nodes.yaml | 10 | 心跳、CRUD、环境配置、远程部署、引导 |
| Accounts | accounts.yaml | 7 | Agent 类型、账号 CRUD、Volume 归档 |
| Operations | operations.yaml | 4 | 统一 Operation/Action 模型 |
| Proxies | proxies.yaml | 6 | 代理 CRUD、连接测试 |
| Instances | instances.yaml | 6 | 容器实例 CRUD、启动/停止 |
| Terminals | terminals.yaml | 5 | 终端会话管理 |
| Templates | templates.yaml | 18 | 任务模板、Agent 模板、Skills、MCP、安全策略 |
| HITL | hitl.yaml | 9 | 审批、反馈、干预、确认 |
| Monitor | monitor.yaml | 4 | 工作流监控、统计 |
| SysConfig | sysconfig.yaml | 2 | 系统配置读写 |

## 代码生成

```bash
# 合并拆分的 OpenAPI 文件为 bundled.yaml
make bundle-openapi

# 生成所有代码（增量，仅规范变化时触发）
make generate-api

# 强制重新生成
make generate-api-force
```

生成流程：
```
openapi/*.yaml → [redocly bundle] → bundled.yaml → [oapi-codegen] → *.gen.go
```

> **注意**：生成的代码仅被部分 handler 使用（task、run、node、events），
> 其他领域的 handler 直接使用 `internal/shared/model` 中的类型。

## 规范修改流程

1. 修改 `api/openapi/` 下对应的 YAML 文件
2. 运行 `make generate-api` 重新生成代码（如需要）
3. 运行 `make test` 确保测试通过
4. 提交所有变更（包括 `bundled.yaml` 和生成的代码）

## 多语言客户端生成

```bash
# TypeScript
docker run --rm -v "${PWD}:/local" openapitools/openapi-generator-cli generate \
    -i /local/api/openapi/bundled.yaml \
    -g typescript-fetch \
    -o /local/web/src/api

# Python
docker run --rm -v "${PWD}:/local" openapitools/openapi-generator-cli generate \
    -i /local/api/openapi/bundled.yaml \
    -g python \
    -o /local/sdk/python
```
