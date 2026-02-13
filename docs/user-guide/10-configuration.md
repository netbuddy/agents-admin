# 配置系统指南

本文档详细说明 Agents Admin 的配置系统架构和使用方法。

## 1. 核心设计

### 1.1 每个环境一个配置文件

每个运行环境只有**一个配置文件**，包含所有组件的配置章节。不同组件（API Server / Node Manager）读取同一文件中各自需要的章节，忽略其余章节。

| 环境 | 配置文件 | 生成方式 |
|------|----------|----------|
| 开发 | `configs/dev.yaml` | 手动维护 |
| 测试 | `configs/test.yaml` | 手动维护 |
| 生产 | `/etc/agents-admin/agents-admin.yaml` | Setup 向导自动生成（API Server 和 Node Manager 共用） |

### 1.2 统一 YAML 配置格式

```yaml
# ---- API Server 专用 ----
api_server:    # 监听端口 + URL（Node Manager 连接用）
database:      # 数据库连接
scheduler:     # 任务调度器

# ---- Node Manager 专用 ----
node:          # 节点共性属性（ID、工作空间、标签）

# ---- 共享 ----
redis:         # Redis 连接
tls:           # TLS/HTTPS 证书

# ---- 敏感信息（仅开发环境写入 YAML，生产环境用 .env）----
auth:          # 认证配置（Token TTL）
```

### 1.3 配置加载优先级

从高到低：

| 优先级 | 来源 | 说明 |
|--------|------|------|
| 1（最高） | 环境变量 | `DB_PASSWORD`、`JWT_SECRET`、`ADMIN_PASSWORD` 等 |
| 2 | `.env` 文件 | 仅生产环境，由 systemd EnvironmentFile 加载 |
| 3 | YAML 配置文件 | 结构化配置（每个环境一个文件） |
| 4（最低） | 代码硬编码默认值 | `internal/config/config.go` 中定义 |

## 2. YAML 与 .env 的职责分离

**核心原则：YAML 存结构化配置，.env 仅存敏感信息。**

### 2.1 属于 YAML 的内容（非敏感）

- 数据库连接信息（host / port / name / password）
- Redis 连接信息
- API Server 端口和 URL
- TLS / 调度器 / 节点配置

### 2.2 属于 .env 的内容（敏感）

| 环境变量 | 说明 |
|----------|------|
| `APP_ENV` | 运行环境标识（`prod`） |
| `JWT_SECRET` | JWT 签名密钥（≥32 字符） |
| `DB_PASSWORD` | 数据库密码（覆盖 YAML 中的 `database.password`） |
| `ADMIN_EMAIL` | 默认管理员邮箱 |
| `ADMIN_PASSWORD` | 默认管理员初始密码 |

> **注意**：`DATABASE_URL` 和 `REDIS_URL` 是结构性配置，不是敏感信息，已从 `.env` 移除。数据库 URL 由代码根据 YAML 配置 + `DB_PASSWORD` 自动构建。

### 2.3 为什么用 .env 而不是其他方案

| 方案 | 适用场景 | 复杂度 | 本项目适用性 |
|------|----------|--------|--------------|
| **.env + systemd EnvironmentFile** | 自部署 Linux 服务 | 低 | ✅ 最佳匹配 |
| Docker/K8s Secrets | 容器编排环境 | 中 | 未来可扩展 |
| HashiCorp Vault | 企业合规需求 | 高 | 过度设计 |

本项目的目标用户是**中小型团队的运维人员**，.env 文件配合 systemd EnvironmentFile 是最简单有效的方案：
- Setup 向导自动生成 `.env` 文件，权限设为 `0600`（仅 owner 可读写）
- systemd 通过 `EnvironmentFile=` 在进程启动前注入环境变量
- 配置管理页面只编辑 YAML 文件，**不暴露 .env 中的敏感信息**

### 2.4 不同环境的敏感信息策略

| 环境 | 敏感信息存放位置 | .env 文件 |
|------|------------------|-----------|
| 开发/测试 | 直接写入 YAML（`database.password`、`auth.*`） | **不需要** |
| 生产 | `/etc/agents-admin/agents-admin.env` | Setup 向导自动生成 |

**开发/测试环境**不需要 `.env` 文件——密码等测试值直接写在 `dev.yaml` / `test.yaml` 中。这是行业通用做法（Django、Rails、Laravel 均如此），clone 后零配置即可运行。

**生产环境**使用 `agents-admin.env`，通过 systemd `EnvironmentFile=` 在进程启动前注入：
- 文件权限 `0600`，属主为 `agents-admin` 用户
- godotenv 不会覆盖 systemd 已注入的变量
- 配置管理页面**只操作 YAML**，不暴露 `.env` 中的敏感信息

## 3. 不同环境的配置方式

### 3.1 开发环境

`configs/dev.yaml` 包含所有组件的完整配置，**无需 `.env` 文件**。密码和密钥直接写入 YAML：

```yaml
api_server:
  port: 8080
  url: https://localhost:8080

database:
  driver: mongodb
  host: localhost
  port: 27017
  name: agents_admin

redis:
  host: localhost
  port: 6380
  db: 0

minio:
  endpoint: localhost:9000
  use_ssl: false

node:
  id: dev-node-01
  workspace_dir: /tmp/agents-workspaces
  labels:
    os: linux

tls:
  enabled: true
  auto_generate: true
  cert_dir: "./certs"

auth:
  jwt_secret: "dev-secret-key-at-least-32-characters-long"
  admin_email: "admin@agents-admin.local"
  admin_password: "admin123456"
```

启动方式：

```bash
make run-web          # 终端 1: 前端
make run-api-dev      # 终端 2: API Server
go run -tags dev ./cmd/nodemanager  # 终端 3: Node Manager
```

### 3.2 生产环境

生产环境使用统一文件名 `/etc/agents-admin/agents-admin.yaml`，由 **Setup 向导** 自动生成：

- **API Server 先安装** → 创建 `agents-admin.yaml`（含 api_server/database/redis/scheduler/tls/auth）+ `agents-admin.env`（敏感信息）
- **Node Manager 后安装** → 读取已有 `agents-admin.yaml`，**合并** node/api_server.url 章节（不覆盖已有章节）
- 反过来安装也一样——后安装的组件通过一级深度合并写入已有文件

安装后可通过前端 **系统设置 → 配置管理** 页面在线查看和修改 YAML 文件。

> 修改配置后需重启服务才能生效。

## 4. 配置章节详解

### 4.1 api_server

```yaml
api_server:
  port: 8080                       # API Server 监听端口
  url: https://192.168.1.100:8080  # API Server 完整 URL（Node Manager 连接用）
```

- `port`：API Server 自身使用
- `url`：Node Manager 读取，用于向 API Server 注册心跳和执行任务回调

### 4.2 database

```yaml
# MongoDB（默认推荐）
database:
  driver: mongodb
  host: localhost
  port: 27017
  name: agents_admin    # 数据库名

# PostgreSQL（可选）
# database:
#   driver: postgres
#   host: localhost
#   port: 5432
#   user: agents
#   password: ""        # dev/test 直接写入；生产环境用 DB_PASSWORD 环境变量覆盖
#   name: agents_admin
#   sslmode: disable

# SQLite（可选）
# database:
#   driver: sqlite
#   path: /var/lib/agents-admin/agents-admin.db
```

支持三种存储引擎，通过 `driver` 字段切换：
- **`mongodb`**（默认）：文档模型，天然适配 JSON-heavy 数据，推荐用于大多数场景
- **`postgres`**：需要事务支持和复杂 SQL 查询的场景
- **`sqlite`**：零运维、单文件部署的轻量场景

PostgreSQL 密码优先级：`DB_PASSWORD` 环境变量 → YAML `database.password` → 硬编码默认值 `agents_dev_password`

### 4.3 redis

```yaml
redis:
  host: localhost
  port: 6380
  db: 0
  url: redis://localhost:6380/0  # 直接指定 URL（优先于 host/port/db）
```

### 4.4 minio

```yaml
minio:
  endpoint: localhost:9000     # MinIO 服务地址
  access_key: minioadmin       # 访问密钥（开发环境可直接写入，生产环境用环境变量 MINIO_ROOT_USER）
  secret_key: ""              # 秘密密钥（生产环境用 MINIO_ROOT_PASSWORD）
  bucket: agents-admin         # 存储桶名
  use_ssl: false               # 是否使用 HTTPS 连接 MinIO
```

### 4.5 node（节点共性配置）

```yaml
node:
  id: ""              # 节点唯一标识（UUID，Setup 向导自动生成）
  workspace_dir: ""   # 工作空间目录（自动检测可写路径）
  labels:             # 节点标签（用于任务调度匹配）
    os: linux
```

### 4.6 tls

```yaml
tls:
  enabled: true
  # 模式一：自签名证书（内网 IP）
  auto_generate: true
  cert_dir: "./certs"
  hosts: "host.docker.internal"

  # 模式二：ACME/Let's Encrypt（互联网域名）
  # acme:
  #   enabled: true
  #   domains: ["admin.example.com"]
  #   email: "admin@example.com"

  # 模式三：手动指定证书
  # cert_file: /path/to/server.pem
  # key_file: /path/to/server-key.pem

  ca_file: ""  # Node Manager 使用的 CA 证书路径
```

### 4.7 scheduler

```yaml
scheduler:
  node_id: api-server
  strategy:
    default: label_match
    chain: [direct, affinity, label_match]
    label_match:
      load_balance: true
  redis:
    read_timeout: 5s
    read_count: 10
  fallback:
    interval: 5m
    stale_threshold: 5m
  requeue:
    offline_threshold: 30s
```

### 4.8 auth

```yaml
auth:
  access_token_ttl: "15m"     # 访问令牌有效期
  refresh_token_ttl: "168h"   # 刷新令牌有效期（7天）
```

> `jwt_secret`、`admin_email`、`admin_password` 仅在开发环境的 YAML 中设置。生产环境通过 `.env` 的环境变量提供。

## 5. 配置管理页面

登录前端后，导航到 **系统设置** 即可查看和编辑当前配置文件：

1. 页面显示配置文件路径和带行号的 YAML 内容
2. 点击 **编辑** 进入编辑模式
3. 修改后点击 **保存配置**（后端验证 YAML 语法）
4. 保存成功后提示重启服务

> 配置管理页面**只操作 YAML 文件**，不会读取或展示 `.env` 中的敏感信息。

## 6. 故障排查

**配置不生效**：

```bash
journalctl -u agents-admin-api-server | grep "Config:"
# 输出：Config{Env: prod, Driver: sqlite, DB: file:/var/lib/..., Redis: redis://...}
```

**找不到配置文件**：通过 `--config /path/to/dir` 显式指定目录。

**数据库驱动错误**：确认 `database.driver` 字段为 `mongodb`、`postgres` 或 `sqlite`。
