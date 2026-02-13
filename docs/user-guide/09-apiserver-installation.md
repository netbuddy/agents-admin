# API Server 安装指南

> 本文介绍如何在生产环境安装和配置 API Server，包括 deb 包安装和二进制安装两种方式。

## 概述

API Server 是一个**自包含的单二进制程序**，内嵌了 Web 前端和配置向导。首次运行时会启动配置向导（端口 15800），引导完成数据库、Redis、TLS 和管理员账户的配置。

**两种部署方式：**

| 方式 | 适用场景 | 二进制路径 | 配置路径 | 配置后重启 |
|------|---------|-----------|---------|----------|
| **deb 包安装** | Ubuntu/Debian | `/usr/bin/` | `/etc/agents-admin/` | systemd 自动重启 |
| **二进制安装** | 所有 Linux | `/usr/local/bin/` | `/etc/agents-admin/` | systemd 自动重启 |

> **说明**：两种安装方式使用相同的配置路径和 systemd 服务管理。

**数据库支持：**

| 数据库 | 适用场景 | 外部依赖 | 初始化 |
|--------|---------|---------|--------|
| **MongoDB**（推荐） | 生产环境、多用户 | 需要 MongoDB 服务 | **无需手动初始化**，自动创建集合和索引 |
| **PostgreSQL** | 需要事务/复杂 SQL | 需要 PostgreSQL 服务 | 需执行 `init-db.sql` |
| **SQLite** | 小规模部署、评估测试 | 无 | 自动创建 |

## 前置要求

- Linux 系统（amd64 或 arm64）
- Docker 和 Docker Compose（如使用一键部署基础设施）
- 或已有的 MongoDB + Redis 服务（如使用已有基础设施）

## 方式一：deb 包安装（推荐）

### 1. 构建 deb 包

在开发机上：

```bash
make deb
```

### 2. 传输到目标服务器

```bash
scp deployments/deb/agents-admin-api-server_*.deb user@<目标IP>:/tmp/
```

### 3. 安装

```bash
sudo dpkg -i /tmp/agents-admin-api-server_*.deb
```

安装后 systemd 自动启动服务，首次运行进入 Setup 向导。

### 4. 访问 Setup 向导

```bash
# 查看向导 URL（包含 Token）
sudo journalctl -u agents-admin-api-server -f
```

在浏览器中打开终端输出的 URL（如 `http://<IP>:15800/setup?token=...`），按步骤完成配置。

## 方式二：二进制安装

### 1. 构建二进制

```bash
# 完整生产构建（含前端嵌入）
make release-linux

# 或仅构建后端（开发用）
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/api-server ./cmd/api-server
```

### 2. 传输到目标服务器

```bash
scp bin/api-server-linux-amd64 user@<目标IP>:/tmp/agents-admin-api-server
```

### 3. 安装到系统路径

```bash
# 添加执行权限
sudo chmod +x /tmp/agents-admin-api-server

# 移动到系统 PATH
sudo mv /tmp/agents-admin-api-server /usr/local/bin/

# 验证安装
agents-admin-api-server --help
```

### 4. 首次运行（以 root 权限）

```bash
sudo agents-admin-api-server
```

程序自动检测到无配置文件，启动 Setup 向导（端口 15800）。按照终端输出的 URL 在浏览器中完成配置。

> **为什么需要 sudo？**
> 以 root 运行时，Setup 向导会自动完成：
> - 创建 `agents-admin` 系统用户
> - 创建 `/etc/agents-admin/`、`/var/lib/agents-admin/`、`/var/log/agents-admin/` 目录
> - 将配置写入 `/etc/agents-admin/agents-admin.yaml` 和 `agents-admin.env`
> - 自动安装并启用 systemd 服务

### 5. 配置完成后

配置保存后，systemd 服务已自动安装，直接启动：

```bash
sudo systemctl start agents-admin-api-server
```

## Setup 向导说明

### 第一步：基础设施（Infrastructure）

两种方式：

- **快速部署**（推荐）：一键通过 Docker Compose 部署 MongoDB + Redis + MinIO，自动生成随机密码
  1. 配置端口（默认：MongoDB 27017、Redis 6379、MinIO 9000/9001）
  2. 点击 **「生成并部署」**，系统自动生成 docker-compose.yml 和 .env 文件
  3. 等待所有服务健康（约 30 秒），自动填充后续步骤的连接信息
- **使用已有服务**：跳过此步，在后续步骤中手动填写已有的 MongoDB/Redis 连接信息

### 第二步：数据库

选择数据库类型：

- **MongoDB**（推荐）：填写连接信息（主机、端口、用户名、密码、数据库名）。**无需手动初始化**，程序启动时自动创建集合和索引
- **PostgreSQL**：填写连接信息。确认步骤中可选择**初始化数据库**（执行建表脚本）
- **SQLite**：无需外部数据库，数据保存在 `/var/lib/agents-admin/agents-admin.db`

### 第三步：Redis

填写 Redis 连接信息（主机、端口、密码）。

> 如使用快速部署，连接信息已自动填充。

### 第四步：HTTPS / TLS

三种模式可选：
- **自签名证书**（推荐局域网）：自动生成 CA + 服务端证书
- **Let's Encrypt**：公网域名自动获取证书
- **禁用 HTTPS**：不推荐

### 第五步：管理员账户

设置管理员邮箱和密码（至少 8 位）。JWT 密钥自动生成。

### 第六步：确认并应用

确认所有配置后保存。

> ⚠ **PostgreSQL 数据库初始化**：仅当选择 PostgreSQL 时显示。初始化会创建全部表结构，使用 `IF NOT EXISTS`，可安全重复执行。MongoDB 和 SQLite 无需此步骤。

## 管理操作

### 查看配置文件

```bash
sudo cat /etc/agents-admin/agents-admin.yaml
```

### 查看日志

```bash
sudo journalctl -u agents-admin-api-server -f
```

### 重新配置

```bash
# 方式 A：手动编辑
sudo vim /etc/agents-admin/agents-admin.yaml
sudo systemctl restart agents-admin-api-server

# 方式 B：重新运行向导（删除配置文件）
sudo rm /etc/agents-admin/agents-admin.yaml
sudo systemctl restart agents-admin-api-server
```

### 服务管理

```bash
sudo systemctl start agents-admin-api-server    # 启动
sudo systemctl stop agents-admin-api-server     # 停止
sudo systemctl restart agents-admin-api-server  # 重启
sudo systemctl status agents-admin-api-server   # 状态
sudo systemctl enable agents-admin-api-server   # 开机自启
```

## 生成的配置文件

### agents-admin.yaml

```yaml
server:
  port: "8080"

database:
  driver: mongodb         # 或 postgres / sqlite
  host: localhost
  port: 27017
  name: agents_admin
  # PostgreSQL 专用字段：
  # user: agents
  # sslmode: disable
  # SQLite 专用字段：
  # path: /var/lib/agents-admin/agents-admin.db

redis:
  host: localhost
  port: 6379
  db: 0

tls:
  enabled: true
  auto_generate: true
  cert_dir: /etc/agents-admin/certs

auth:
  access_token_ttl: "15m"
  refresh_token_ttl: "168h"
```

### agents-admin.env

敏感信息（密码、密钥）保存在 `.env` 文件中，权限 `0600`：

```bash
APP_ENV=prod
JWT_SECRET=<自动生成>
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=<你设置的密码>
# MongoDB 凭据
MONGO_ROOT_USERNAME=agents_admin
MONGO_ROOT_PASSWORD=<自动生成>
# Redis 密码
REDIS_PASSWORD=<自动生成>
# PostgreSQL（仅当 driver=postgres 时）
# DB_PASSWORD=<密码>
```

> **说明**：`DATABASE_URL` 和 `REDIS_URL` 是结构性配置，由代码根据 YAML + 环境变量自动构建，不再写入 `.env`。

## 目录结构

| 路径 | 用途 |
|------|------|
| `/etc/agents-admin/` | 配置文件 |
| `/etc/agents-admin/certs/` | TLS 证书 |
| `/var/lib/agents-admin/` | 数据文件（SQLite 数据库等） |
| `/var/log/agents-admin/` | 日志 |

## 下一步

- [Node Manager 安装](./08-nodemanager-installation.md) — 安装执行节点
- [快速入门](./01-quick-start.md) — 开发环境搭建
- [TLS/HTTPS 配置](./07-tls-https.md) — 详细 TLS 说明
