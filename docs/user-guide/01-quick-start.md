# 快速入门

> 本指南将带你完成从部署到执行第一个任务的完整流程。

## 前置要求

- Docker 和 Docker Compose（基础设施）
- Go 1.22+（后端）
- Node.js 18+（前端，仅开发模式需要）

## 启动方式说明

系统提供两种运行模式，请根据使用场景选择：

| 模式 | 命令 | 前端访问地址 | 适用场景 |
|------|------|-------------|---------|
| **开发模式** | `make run-api-dev` + `make run-web` | `http://localhost:3002` | 日常开发，前端支持热更新 |
| **生产模式** | `make release-linux` 后运行二进制 | `http://localhost:8080` | 部署、演示，前端嵌入后端 |

> **⚠️ 注意区别**：
> - `make run-api-dev` 使用 `-tags dev` 编译，**不嵌入前端**，必须配合 `make run-web` 使用
> - `make run-api` 不带 dev tag，会尝试从 `web/out/` 加载嵌入的前端（需先执行 `make web-build`）
> - **不要同时使用** `make run-api` + `make run-web`，两个地址都能访问前端会造成混淆

## 步骤 1：启动基础设施

```bash
# 启动 PostgreSQL + Redis + MinIO
make dev-up
```

等待服务就绪后，终端会显示服务状态。

## 步骤 2：初始化数据库

首次部署需要导入数据库 schema：

```bash
psql "postgres://agents:agents_dev_password@localhost:5432/agents_admin" \
  -f deployments/init-db.sql
```

> 如果数据库已初始化过，可跳过此步。重复执行不会报错（使用 `IF NOT EXISTS`）。

## 步骤 3：启动服务

### 开发模式（推荐日常使用）

```bash
# 终端 1：启动 API Server（开发模式，不嵌入前端）
make run-api-dev

# 终端 2：启动前端 Dev Server（支持热更新）
make run-web

# 终端 3：启动 NodeManager
make run-nodemanager
```

- API Server：`http://localhost:8080`（仅 API）
- 前端页面：`http://localhost:3002`（Next.js Dev Server，API 请求自动代理到 8080）

### 生产模式

```bash
# 1. 构建生产版本（前端静态导出 + 嵌入 Go 二进制）
make release-linux

# 2. 运行（需要 PostgreSQL 和 Redis）
DATABASE_URL="postgres://agents:agents_dev_password@localhost:5432/agents_admin?sslmode=disable" \
REDIS_URL="redis://localhost:6380/0" \
./bin/api-server-linux-amd64
```

前端和 API 通过同一地址访问：`http://localhost:8080`。

## 第一次使用

完成部署后，按以下步骤体验完整流程：

### 步骤 1：确认节点在线

1. 打开浏览器访问 `http://localhost:8080`（生产模式）或 `http://localhost:3002`（开发模式）
2. 点击左侧导航栏的 **「节点管理」**
3. 确认至少有一个节点显示为 **「在线」** 状态

### 步骤 2：配置代理（可选）

如果你的网络需要代理才能访问外网（Agent API 调用需要）：

1. 点击左侧导航栏的 **「代理管理」**
2. 点击 **「添加代理」**
3. 填写代理信息（类型、地址、端口）
4. 点击 **「测试」** 按钮，在弹出的对话框中输入目标网址（默认 `https://www.google.com`），验证代理能否实际访问目标地址。也可点击快捷按钮（Google / GitHub / OpenAI / Anthropic）快速切换常用目标

### 步骤 3：创建账号

1. 点击左侧导航栏的 **「账号管理」**
2. 点击 **「添加账号」**
3. 选择 Agent 类型（如 Qwen-Code）
4. 选择认证方式（OAuth / API Key）
5. 选择目标节点
6. 按提示完成认证

### 步骤 4：创建实例

1. 点击左侧导航栏的 **「实例管理」**
2. 点击 **「创建实例」**
3. 选择刚才创建的账号
4. 点击创建后，点击 **「启动」** 按钮
5. 等待实例状态变为 **「运行中」**

### 步骤 5：创建并执行任务

1. 点击左侧导航栏的 **「任务看板」**（首页）
2. 点击右上角 **「新建任务」**
3. 填写：
   - **任务名称**：如 "修复登录页面 bug"
   - **Agent 类型**：选择与账号匹配的类型
   - **选择实例**：选择运行中的实例
   - **任务提示词**：描述你希望 Agent 完成的任务
4. 点击 **「创建任务」**
5. 在看板中找到新任务卡片，点击展开详情
6. 点击 **「启动」** 按钮开始执行

### 步骤 6：监控执行

- 任务卡片会从「待处理」列移动到「运行中」列
- 点击任务查看详情，可以实时查看 Agent 的输出事件流
- 执行完成后，任务将移动到「已完成」或「失败」列

## 下一步

- [任务管理详解](./02-task-management.md) — 深入了解任务创建和管理
- [账号与实例管理](./03-account-instance.md) — 管理多个 Agent 账号
- [监控与运维](./06-monitoring.md) — 系统健康检查和指标监控
