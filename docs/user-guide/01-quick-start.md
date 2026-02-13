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
| **开发模式** | `make run-api-dev` + `make run-web` | `https://localhost:8080` | 日常开发，前端支持热更新 |
| **生产模式** | `make release-linux` 后运行二进制 | `https://<IP>:8080` | 部署、演示，前端嵌入后端 |

> **⚠️ 注意区别**：
> - `make run-api-dev` 使用 `-tags dev` 编译，**不嵌入前端**，Go 反向代理 Next.js `:3002`
> - `make run-api` 不带 dev tag，会尝试从 `web/out/` 加载嵌入的前端（需先执行 `make web-build`）
> - 浏览器统一通过 Go HTTPS 入口访问（开发 `:8080`，生产 `:443` 或 `:8080`）

## 步骤 1：启动基础设施

```bash
# 启动 MongoDB + Redis + MinIO
make dev-up
```

等待服务就绪后，终端会显示服务状态。

> **MongoDB 为默认数据库**，schema-less 设计，**无需手动初始化数据库**。程序启动时自动创建集合和索引。
>
> 如需使用 PostgreSQL，请参考 [配置指南](./10-configuration.md) 切换 `database.driver`，并手动执行 `deployments/init-db.sql`。

## 步骤 2：启动服务

### 开发模式（推荐日常使用）

```bash
# 终端 1：启动前端 Dev Server（支持热更新）
make run-web

# 终端 2：启动 API Server（HTTPS，自动代理前端到 Next.js :3002）
make run-api-dev

# 终端 3：启动 NodeManager
make run-nodemanager
```

浏览器访问：`https://localhost:8080`（Go 统一 HTTPS 入口，API + 前端代理）

### 生产模式

```bash
# 1. 构建生产版本（前端静态导出 + 嵌入 Go 二进制）
make release-linux

# 2. 首次运行 — 自动进入 Setup Wizard（默认端口 15800）
sudo ./bin/api-server-linux-amd64

# 3. 在浏览器中打开终端输出的 Setup URL，按向导完成配置
#    向导支持一键部署基础设施（Docker Compose）或连接已有服务
```

#### Setup Wizard 命令行参数

| 参数 | 环境变量 | 默认值 | 说明 |
|------|---------|--------|------|
| `--setup-port` | `SETUP_PORT` | `15800` | Setup 向导监听端口 |
| `--setup-listen` | `SETUP_LISTEN` | `0.0.0.0` | Setup 向导监听地址 |
| `--reconfigure` | — | `false` | 强制重新进入配置向导 |
| `--config` | — | 自动搜索 | 配置文件目录 |

```bash
# 示例：自定义端口（避免端口冲突）
sudo ./bin/api-server-linux-amd64 --setup-port 9999

# 示例：通过环境变量指定
sudo SETUP_PORT=9999 ./bin/api-server-linux-amd64

# 示例：强制重新配置
sudo ./bin/api-server-linux-amd64 --reconfigure
```

> **Node Manager** 也支持相同的参数（默认端口 `15700`）：
> ```bash
> sudo ./bin/nodemanager-linux-amd64 --setup-port 15701
> ```

配置完成后，systemd 服务自动管理。详见 [API Server 安装指南](./09-apiserver-installation.md)。

## 第一次使用

完成部署后，按以下步骤体验完整流程：

### 步骤 1：确认节点在线

1. 打开浏览器访问 `https://localhost:8080`（Go 统一 HTTPS 入口，开发和生产相同）
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
