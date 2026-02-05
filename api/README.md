# API 规范与代码生成

本目录包含 OpenAPI 规范和自动生成的代码。

## 目录结构

```
api/
├── openapi/                   # OpenAPI 3.0 规范（按领域拆分）
│   ├── openapi.yaml           # 主入口文件
│   ├── common.yaml            # 公共定义（参数、响应、基础模型）
│   ├── health.yaml            # 健康检查 API
│   ├── tasks.yaml             # Task 任务管理 API
│   ├── runs.yaml              # Run 执行管理 API
│   ├── events.yaml            # Event 事件管理 API
│   ├── nodes.yaml             # Node 节点管理 API
│   └── bundled.yaml           # 合并后的完整规范（自动生成）
├── codegen/                   # 代码生成配置
│   ├── models.yaml            # 模型生成配置
│   ├── server.yaml            # 服务接口生成配置
│   └── spec.yaml              # 嵌入规范生成配置
├── generated/
│   └── go/                    # 生成的 Go 代码（按类型拆分）
│       ├── models.gen.go      # 模型定义（~250 行）
│       ├── server.gen.go      # 服务接口（~680 行）
│       └── spec.gen.go        # 嵌入规范（~130 行）
└── README.md                  # 本文件
```

## 使用方式

### 增量代码生成

```bash
# 只在 OpenAPI 规范变化时重新生成（推荐）
make generate-api

# 强制重新生成
make generate-api-force

# 单独生成某一类型
make generate-api-models   # 只生成模型
make generate-api-server   # 只生成服务接口
make generate-api-spec     # 只生成嵌入规范

# 合并拆分的 OpenAPI 文件
make bundle-openapi

# 构建时自动检查是否需要重新生成
make build
```

### 生成流程

```
openapi/*.yaml  --[redocly bundle]-->  bundled.yaml  --[oapi-codegen]-->  *.gen.go
    (拆分的规范)                         (合并的规范)                       (Go 代码)
```

### 生成的代码内容

| 文件 | 内容 |
|------|------|
| `models.gen.go` | 模型结构体：`Task`, `Run`, `Event`, `Node`, `CreateTaskRequest` 等 |
| `server.gen.go` | 服务接口：`ServerInterface`（定义所有 API 方法签名）|
| `spec.gen.go` | 嵌入的 OpenAPI 规范（用于 Swagger UI） |

### 架构说明

由于历史原因，当前采用**双轨制**：

| 用途 | 使用的结构体 | 位置 |
|------|-------------|------|
| API 文档 | 生成的模型 | `api/generated/go/` |
| 客户端 SDK | 生成的模型 | `api/generated/go/` |
| Handler 内部 | 本地定义的模型 | `internal/apiserver/handler/*.go` |

**原因**：
- 本地模型使用 `internal/shared/model` 中的复杂类型（如 `model.WorkspaceConfig`）
- 生成的模型与 OpenAPI 规范严格对应，类型更简单

**后续计划**：
- 逐步将 `internal/shared/model` 中的类型迁移到 OpenAPI 规范
- 统一使用生成的模型

## 多语言客户端生成

使用 openapi-generator 生成其他语言的客户端：

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

## 规范修改流程

1. 修改 `api/openapi/` 下对应的 YAML 文件（按领域拆分）
2. 运行 `make generate-api` 重新生成代码
3. 运行 `make test` 确保测试通过
4. 提交所有变更（包括 `bundled.yaml` 和生成的代码）

## 注意事项

- 生成的代码 **已纳入 Git 管理**（确保 CI/CD 环境一致）
- 修改 OpenAPI 规范后必须重新生成代码
- `make build` 会自动检查规范是否变化
- `bundled.yaml` 是自动生成的，不要手动编辑
