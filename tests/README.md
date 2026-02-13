# 测试策略与目录结构

> **更新日期**：2026-02-10
> **存储引擎**：MongoDB（默认）/ PostgreSQL / SQLite，通过 `PersistentStore` 接口统一抽象

## 1. 测试分层

```
                    ╱╲
                   ╱  ╲         tests/e2e/           ← 部署环境运行（发布前验收）
                  ╱────╲        E2E / Acceptance Tests
                 ╱      ╲
                ╱────────╲      tests/integration/    ← 本地运行 + 真实 DB
               ╱          ╲     Integration Tests
              ╱────────────╲
             ╱              ╲   tests/handler/        ← 本地运行，纯内存
            ╱────────────────╲  Handler Unit Tests
           ╱                  ╲
          ╱────────────────────╲  internal/*_test.go  ← 本地运行，无依赖
         ╱                      ╲ Pure Unit Tests
```

> **关于回归测试**：回归测试（Regression Testing）不是一个独立的测试层级，而是一种
> **变更后再验证策略**——任何层级（单元/集成/E2E）的测试都可以承担回归验证的职责。
> 本项目通过 CI 在每次变更后自动运行各层级测试来实现回归保护，不再设置独立的
> `regression/` 目录。参见 [ISTQB 术语表](https://istqb-glossary.page/regression-testing/)。

## 2. 执行环境

| 测试层级 | 执行环境 | 外部依赖 | 触发时机 |
|----------|----------|----------|----------|
| **纯单元测试** | 本地开发环境 | 无（纯内存 / Mock） | 每次提交前 |
| **Handler 单元测试** | 本地开发环境 | 无（httptest + Mock） | 每次提交前 |
| **集成测试** | 本地开发环境 | 真实数据库（MongoDB / PG） | 功能开发完成后 |
| **E2E 验收测试** | **部署环境** | 完整部署的 API Server + DB | 发布前验收 |

> **关键区别**：
> - **单元/Handler 测试**：`go test` 直接运行，零外部依赖
> - **集成测试**：本地运行，连接真实数据库（如 `mongodb://localhost:27017`）
> - **E2E 验收测试**：需要完整部署的测试环境（API Server + MongoDB + 认证）

## 3. 目录结构

```
tests/
├── README.md                    # 本文档
├── testutil/                    # 共享测试基础设施
│   ├── db.go                    # 数据库连接工具（集成测试用）
│   ├── env.go                   # 进程内测试环境 InProcEnv（handler/integration 用）
│   └── e2e.go                   # E2E 客户端 E2EClient（HTTPS + Cookie 认证）
│
├── handler/                     # Handler 单元测试（本地运行）
│   ├── 10-task-management/
│   └── 20-run-management/
│
├── integration/                 # 集成测试（本地运行 + 真实 DB）
│   ├── 10-task-management/
│   ├── 20-run-management/
│   ├── 30-scheduling/
│   ├── 40-node-management/
│   ├── 48-agent/
│   ├── 50-auth/
│   ├── 55-runner/
│   └── 60-proxy/
│
└── e2e/                         # E2E 验收测试（部署环境运行）
    ├── README.md                # 验收测试总览与 API 覆盖率
    ├── health/                  # 健康检查、指标、引导配置
    ├── auth/                    # 认证授权（登录、令牌、权限）
    ├── task/                    # 任务管理（CRUD、分页、子任务、树）
    ├── run/                     # 执行管理（生命周期、事件流、取消）
    ├── node/                    # 节点管理（心跳、环境配置、预配置）
    ├── proxy/                   # 代理管理（CRUD、连通性测试）
    ├── instance/                # 实例管理（CRUD、启停）
    ├── terminal/                # 终端会话（CRUD）
    ├── template/                # 模板资源（任务/Agent/技能/MCP/安全策略）
    ├── hitl/                    # 人机协作（审批、反馈、干预、确认）
    ├── operation/               # 系统操作（Operation、Action、账号、Agent类型）
    ├── monitor/                 # 监控（统计、工作流）
    └── sysconfig/               # 系统配置（读写）
```

## 4. 测试层级详解

### 4.1 纯单元测试

**位置**: `internal/**/*_test.go`（与源码同目录）

- 测试单个函数或方法，完全隔离
- 使用 Mock 替代外部依赖
- 速度最快（微秒级）

```bash
go test ./internal/...
```

### 4.2 Handler 单元测试

**位置**: `tests/handler/`

- 使用 `httptest.NewRecorder` + `router.ServeHTTP()` 在内存中调用
- 可用 Mock Database 隔离，速度极快（毫秒级）

```bash
go test ./tests/handler/...
```

### 4.3 集成测试

**位置**: `tests/integration/`

- 使用 `httptest.NewServer` 启动真实 HTTP 服务器
- 连接真实数据库（MongoDB 默认，可选 PostgreSQL）
- 测试组件间的真实交互

```bash
# 默认使用 MongoDB
go test ./tests/integration/...

# 使用 PostgreSQL
TEST_DB_DRIVER=postgres TEST_DATABASE_URL="postgres://..." go test ./tests/integration/...
```

### 4.4 E2E 验收测试

**位置**: `tests/e2e/`

**定位**：发布前验收测试，按功能维度全面覆盖项目所有 API 端点。

- 13 个功能子目录，覆盖 **65+ 个 API 端点**
- 共享 `testutil.E2EClient`：HTTPS（TLS 跳过验证）+ Cookie 认证 + 自动登录
- 每个子目录包含 `README.md` 详细描述覆盖的功能
- **需要完整部署的测试环境**

```bash
# 全量验收
API_BASE_URL=https://test-server:8080 go test -v ./tests/e2e/...

# 按维度运行
API_BASE_URL=https://test-server:8080 go test -v ./tests/e2e/health/...
API_BASE_URL=https://test-server:8080 go test -v ./tests/e2e/task/...
API_BASE_URL=https://test-server:8080 go test -v ./tests/e2e/run/...

# 快速冒烟
API_BASE_URL=https://test-server:8080 go test -v ./tests/e2e/health/... ./tests/e2e/auth/...
```

**E2E 环境变量**：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `API_BASE_URL` | `https://localhost:8080` | API Server 地址 |
| `ADMIN_EMAIL` | `admin@agents-admin.local` | 管理员邮箱 |
| `ADMIN_PASSWORD` | `admin123456` | 管理员密码 |

## 5. 回归测试策略

回归测试是一种**变更后再验证的测试活动**（ISTQB），不是固定的测试形态。
本项目通过以下方式实现回归保护：

| 变更阶段 | 回归手段 | 覆盖范围 |
|----------|----------|----------|
| **每次提交** | 单元测试 + Handler 测试 | 函数/接口逻辑 |
| **功能完成** | 集成测试 | 组件间交互、数据库读写 |
| **发布前** | E2E 验收测试（全量） | 所有功能维度端到端 |
| **热修复后** | 按影响范围选择性运行 | 受影响的测试层级 |

> 不再维护独立的 `tests/regression/` 目录。
> 任何层级的测试在变更后重新运行，都在承担回归验证的职责。

## 6. 存储驱动环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `TEST_DB_DRIVER` | `mongodb` | 存储驱动：`mongodb` / `postgres` |
| `TEST_MONGO_URI` | `mongodb://localhost:27017` | MongoDB 连接 URI |
| `TEST_MONGO_DB` | `agents_admin_test` | MongoDB 测试数据库名 |
| `TEST_DATABASE_URL` | `postgres://agents:...@localhost:5432/agents_admin` | PostgreSQL 连接 URL |
| `TEST_REDIS_URL` | `redis://localhost:6380/0` | Redis 连接 URL（可选） |

## 7. 快速参考

```bash
# ---- 本地开发（每次提交前） ----
go test ./internal/...                # 纯单元测试
go test ./tests/handler/...           # Handler 测试

# ---- 本地开发（功能完成后） ----
go test ./tests/integration/...       # 集成测试（需本地 MongoDB）
go test -v ./internal/shared/storage/mongostore/...  # mongostore 测试

# ---- 部署环境（发布前验收） ----
API_BASE_URL=https://test-server:8080 go test -v ./tests/e2e/...

# ---- 全量验证 ----
go test ./...
```
