# 开发环境搭建指南

本文档面向新加入的开发者，介绍如何从零搭建 Agents Admin 的本地开发环境。

## 前置条件

| 工具 | 最低版本 | 用途 |
|------|----------|------|
| **Go** | 1.26+ | 后端编译 |
| **Node.js** | 22+ | 前端构建 |
| **npm** | 10+ | 前端包管理 |
| **Docker** & **Docker Compose** | Docker 24+, Compose v2+ | 基础设施（MongoDB、Redis、MinIO） |
| **Git** | 2.30+ | 版本控制 |
| **oapi-codegen** | 2.4+ | OpenAPI 代码生成（可选，仅修改 API 定义时需要） |

### 安装 Go

```bash
# Linux (amd64)
wget https://go.dev/dl/go1.26.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.26.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc
source ~/.bashrc
go version
```

> macOS 推荐 `brew install go`；Windows 使用官方 MSI 安装包。

### 安装 Node.js

```bash
# 推荐使用 nvm
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.4/install.sh | bash
source ~/.bashrc
nvm install 22
node -v && npm -v
```

### 安装 Docker

参考 [Docker 官方文档](https://docs.docker.com/engine/install/) 安装。确保当前用户在 `docker` 组中：

```bash
sudo usermod -aG docker $USER
# 重新登录后生效
docker compose version
```

## 步骤 1：克隆仓库

```bash
git clone https://github.com/netbuddy/agents-admin.git
cd agents-admin
```

## 步骤 2：配置环境变量

项目提供了三个环境变量文件：

| 文件 | 用途 |
|------|------|
| `.env.template` | 模板，列出所有配置项（提交到 Git） |
| `.env.dev` | 开发环境预设值（提交到 Git） |
| `.env.test` | 测试环境预设值，端口偏移避免冲突（提交到 Git） |

开发环境 **无需** 创建额外的 `.env` 文件，`.env.dev` 已包含全部开发凭据。

## 步骤 3：启动基础设施

```bash
# 启动 MongoDB、Redis、MinIO
make dev-up

# 验证服务状态
docker compose -f deployments/docker-compose.infra.yml --env-file .env.dev ps
```

服务端口（开发环境默认值）：

| 服务 | 端口 |
|------|------|
| MongoDB | 27017 |
| Redis | 6380 |
| MinIO API | 9000 |
| MinIO Console | 9001 |

### 可选：启动可视化管理工具

开发环境提供了 Redis 和 MongoDB 的可视化管理工具，通过 Docker Compose `tools` profile 启动：

```bash
make dev-tools-up
```

| 工具 | 端口 | 用途 |
|------|------|------|
| [RedisInsight](https://hub.docker.com/r/redis/redisinsight) | [http://localhost:5540](http://localhost:5540) | Redis 可视化管理 |
| [dbgate](https://hub.docker.com/r/dbgate/dbgate) | [http://localhost:3100](http://localhost:3100) | MongoDB 可视化管理（已预配连接） |

> **提示**：dbgate 启动后已自动配置好 MongoDB 连接（使用 `.env.dev` 中的凭据），无需手动添加。
> RedisInsight 首次使用需手动添加连接：Host=`localhost`，Port=`6380`，Password=`.env.dev` 中的 `REDIS_PASSWORD`。

停止可视化工具：

```bash
make dev-tools-down
```

### 可选：构建 ttyd 工具镜像

NodeManager 的 Web 终端功能需要 `tools/ttyd:latest` 镜像（用于通过浏览器远程访问 Agent 容器终端）：

```bash
docker build -f deployments/Dockerfile.ttyd -t tools/ttyd:latest .
```

> 该镜像包含 `ttyd` + `docker-cli` + `bash` + `curl`，NodeManager 会按需启动该容器，通过 `docker exec` 连接到目标 Agent 容器。

## 步骤 4：安装后端依赖

```bash
# Go modules 自动下载依赖
go mod download

# 验证编译（开发模式需要 -tags dev 跳过前端静态文件嵌入）
go build -tags dev ./...
```

Go 依赖由 `go.mod` / `go.sum` 管理，`go mod download` 会下载所有依赖到 `$GOPATH/pkg/mod`。

### 安装 oapi-codegen

当修改了 `api/openapi/*.yaml` 文件时需要重新生成代码：

```bash
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
# 验证
oapi-codegen -version
```

## 步骤 5：安装前端依赖

```bash
cd web
npm install
cd ..
```

前端依赖由 `web/package.json` / `web/package-lock.json` 管理，`npm install` 会下载到 `web/node_modules/`。

## 步骤 6：运行开发服务

开发模式需要同时启动 **三个进程**（建议分别在不同终端中运行）：

### 终端 1：前端 Next.js Dev Server

```bash
make run-web
# → http://127.0.0.1:3002
```

### 终端 2：API Server（开发模式，HTTPS + 反向代理前端）

```bash
make run-api-dev
# → https://localhost:8080
# 自动反向代理前端请求到 Next.js :3002
```

### 终端 3：NodeManager

```bash
make run-nodemanager
```

> **提示**：`run-api-dev` 模式下 API Server 会自动生成自签名 TLS 证书到 `./certs/`，浏览器首次访问需接受证书警告。

## 步骤 7：验证

```bash
# 后端单元测试
make test

# 全部测试（含集成测试、E2E 测试）
make test-all

# 代码检查（需安装 golangci-lint）
make lint
```

## 项目结构概览

```
agents-admin/
├── api/                  # OpenAPI 定义 & 生成的 Go 代码
│   ├── openapi/          # 按域拆分的 OpenAPI YAML 文件
│   ├── codegen/          # oapi-codegen 配置
│   └── generated/go/     # 生成的 Go 类型代码（勿手动编辑）
├── cmd/                  # 可执行入口
│   ├── api-server/       # API Server 主程序
│   └── nodemanager/      # NodeManager 主程序
├── configs/              # 环境配置文件（dev.yaml 等）
├── deployments/          # Docker、Compose、Deb 打包、监控配置
├── docs/                 # 用户文档 & 设计文档
├── internal/             # 内部包（不对外导出）
│   ├── apiserver/        # API Server 业务逻辑
│   ├── config/           # 配置加载
│   ├── nodemanager/      # NodeManager 业务逻辑
│   ├── shared/           # 共享库（存储、认证、事件等）
│   └── tlsutil/          # TLS 工具
├── internal-docs/        # 内部技术文档（架构、设计、分析）
├── tests/                # 集成测试 & E2E 测试
└── web/                  # Next.js 前端
```

## 常用 Make 命令

| 命令 | 说明 |
|------|------|
| `make dev-up` | 启动开发基础设施 |
| `make dev-down` | 停止开发基础设施 |
| `make run-api` | 运行 API Server（生产模式，嵌入前端） |
| `make run-api-dev` | 运行 API Server（开发模式，代理前端） |
| `make run-nodemanager` | 运行 NodeManager |
| `make run-web` | 运行前端开发服务器 |
| `make build` | 构建后端二进制 |
| `make test` | 运行单元测试 |
| `make generate-api` | 重新生成 OpenAPI 代码 |
| `make generate-api-force` | 强制重新生成全部代码 |
| `make dev-tools-up` | 启动可视化工具（RedisInsight、dbgate） |
| `make dev-tools-down` | 停止可视化工具 |

## 常见问题

### Q: `go build` 报错 `pattern all:out: no matching files found`？

开发模式下需要使用 `-tags dev` 跳过前端静态文件嵌入：

```bash
go build -tags dev ./...
```

### Q: `go build` 报错找不到依赖？

```bash
go mod tidy
go mod download
```

### Q: 前端 `npm install` 失败？

确保 Node.js 版本 ≥ 22，清除缓存重试：

```bash
cd web && rm -rf node_modules package-lock.json && npm install
```

### Q: Docker Compose 启动失败？

检查端口是否被占用：

```bash
lsof -i :27017  # MongoDB
lsof -i :6380   # Redis
lsof -i :9000   # MinIO
```

### Q: API Server 启动时 TLS 证书错误？

删除旧证书重新生成：

```bash
rm -rf certs/
make run-api-dev
```
