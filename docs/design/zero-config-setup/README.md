# Zero-Config Setup 架构设计

## 核心思想

**API Server 是配置的唯一真相源（Single Source of Truth）**：
- API Server 首次运行 → Web 向导配置自身（DB、Redis、TLS、Auth）
- Node Manager 首次运行 → 只需 API Server URL → 自动获取所有配置

## 当前 vs 目标

```
当前 Node Manager 安装流程（4 步）:
  Step 1: Node ID + API Server URL + Workspace
  Step 2: Redis URL
  Step 3: TLS/CA 证书
  Step 4: 确认

目标流程（1 步）:
  Step 1: API Server URL → 自动连接 → 下载 CA → 获取 Redis URL → 完成
```

## Phase 1: Node Bootstrap（本次实施）

### API Server 新端点

```
GET /api/v1/node-bootstrap  （免认证）

Response:
{
  "redis_url": "redis://host:port/db",
  "tls": {
    "enabled": true,
    "ca_url": "/ca.pem"
  }
}
```

### Node Manager 向导简化

```
旧流程: 4 步分步表单
新流程:
  1. 用户输入 API Server URL
  2. 后端自动执行:
     a. InsecureSkipVerify 连接 API Server
     b. 如果 HTTPS → 下载 /ca.pem → 用 CA 重新验证
     c. 调用 /api/v1/node-bootstrap 获取 Redis URL
     d. 自动生成 UUID Node ID
     e. 自动检测 Workspace 目录
  3. 显示确认摘要 → 保存 → 启动
```

### 修改文件

- `internal/apiserver/server/handler.go` — 注册 bootstrap 路由
- `internal/apiserver/server/common.go` — BootstrapConfig + handler
- `internal/apiserver/auth/middleware.go` — 添加 bootstrap 到公开路由
- `internal/nodemanager/setup/handlers.go` — handleBootstrap 合并连接+下载+配置
- `internal/nodemanager/setup/static/index.html` — 前端简化为 1 步
- `cmd/api-server/main.go` — 传递 bootstrap 配置

## Phase 2: API Server 首次运行向导（后续迭代）

### 需要配置的项目

| 配置项 | 必需 | 说明 |
|--------|------|------|
| PostgreSQL | 是 | host, port, user, password, dbname |
| Redis | 是 | host, port |
| TLS 模式 | 是 | auto_generate / ACME / manual |
| Admin 账户 | 是 | email, password |
| JWT Secret | 自动生成 | 32+ 字符随机串 |
| Server Port | 默认 8080 | 可选修改 |

### 架构复用

复用 Node Manager 向导模式：
- 嵌入 Web UI + Token 认证
- 首次运行检测（无配置文件 → 启动向导）
- 分步表单 + 连接测试
- 配置文件生成 + 自动重启

## TODO

- [x] Phase 1: API Server 添加 /api/v1/node-bootstrap 端点 ✅
- [x] Phase 1: Node Manager 向导简化为 2 步（输入 URL → 确认保存）✅
- [x] Phase 1: 单元测试（17 pass）+ 远程 192.168.213.24 验证 ✅
- [x] Phase 2: API Server 首次运行向导后端 + 前端 ✅
- [x] Phase 2: 单元测试（12 pass）+ 远程 192.168.213.24 集成验证 ✅
