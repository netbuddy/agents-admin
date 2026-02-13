# E2E 验收测试套件

> **定位**：发布前验收测试（Acceptance Testing），验证所有功能维度在完整部署环境下端到端可用。

## 设计原则

1. **按功能维度划分子目录**，每个子目录独立运行、独立报告
2. **共享客户端基础设施**（`testutil.E2EClient`）：HTTPS + Cookie 认证 + 自动登录
3. **测试自治**：每个测试自行创建前置数据，结束后自行清理
4. **优雅降级**：当前置条件不满足时 `t.Skip()` 而非 `t.Fatal()`

## 目录结构

```
tests/e2e/
├── README.md              ← 本文档
├── health/                ← 健康检查、指标、引导配置
├── auth/                  ← 认证授权（登录、令牌刷新、权限）
├── task/                  ← 任务管理（CRUD、分页、子任务、树）
├── run/                   ← 执行管理（生命周期、事件流、取消）
├── node/                  ← 节点管理（心跳、环境配置、预配置）
├── proxy/                 ← 代理管理（CRUD、连通性测试）
├── instance/              ← 实例管理（CRUD、启停）
├── terminal/              ← 终端会话（CRUD）
├── template/              ← 模板资源（任务/Agent模板、技能、MCP、安全策略）
├── hitl/                  ← 人机协作（审批、反馈、干预、确认）
├── operation/             ← 系统操作（Operation、Action、账号、Agent类型）
├── monitor/               ← 监控（统计、工作流）
└── sysconfig/             ← 系统配置（读写）
```

## API 覆盖率

| 功能维度 | 端点数 | 覆盖端点 |
|----------|--------|----------|
| Health | 3 | `/health`, `/metrics`, `/api/v1/node-bootstrap` |
| Auth | 4 | `login`, `me`, `refresh`, `password` |
| Task | 5 | CRUD + `subtasks` + `tree` |
| Run | 5 | CRUD + `cancel` + `events` (GET/POST) |
| Node | 6 | `heartbeat`, CRUD, `env-config`, `provisions` |
| Proxy | 5 | CRUD + `test` |
| Instance | 5 | CRUD + `start` + `stop` |
| Terminal | 3 | CRUD |
| Template | 15 | Task/Agent 模板, Skills, MCP Servers, Security Policies |
| HITL | 6 | approvals, feedbacks, interventions, confirmations, pending |
| Operation | 4 | operations, actions, accounts, agent-types |
| Monitor | 2 | stats, workflows |
| SysConfig | 2 | GET/PUT config |

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `API_BASE_URL` | `https://localhost:8080` | API Server 地址 |
| `ADMIN_EMAIL` | `admin@agents-admin.local` | 管理员邮箱 |
| `ADMIN_PASSWORD` | `admin123456` | 管理员密码 |

## 运行

```bash
# 全量验收
API_BASE_URL=https://test-server:8080 go test -v ./tests/e2e/...

# 按维度运行
API_BASE_URL=https://test-server:8080 go test -v ./tests/e2e/health/...
API_BASE_URL=https://test-server:8080 go test -v ./tests/e2e/task/...
API_BASE_URL=https://test-server:8080 go test -v ./tests/e2e/run/...

# 快速冒烟（仅健康检查 + 认证）
API_BASE_URL=https://test-server:8080 go test -v ./tests/e2e/health/... ./tests/e2e/auth/...
```

## 前置条件

1. API Server 已部署并可达（HTTPS）
2. MongoDB（或 PostgreSQL）已就绪
3. 管理员账号已创建（首次部署时自动 seed）
4. 自签名证书环境下客户端自动跳过 TLS 验证
