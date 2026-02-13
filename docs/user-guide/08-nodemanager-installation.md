# Node Manager 安装指南

> 本文介绍如何安装和配置 Node Manager，包括 deb 包安装和二进制直接运行两种方式。

## 概述

Node Manager 是一个**自包含的单二进制程序**，内嵌了 Web 配置向导。首次运行时，它会启动一个 Web 页面引导你完成全部配置；配置保存后自动进入工作模式，Web 页面关闭。

**两种部署方式：**

| 方式 | 适用场景 | 配置路径 | 配置后重启 |
|------|---------|---------|----------|
| **deb 包安装** | Ubuntu/Debian | `/etc/agents-admin/` | systemd 自动重启 |
| **二进制安装** | 所有 Linux | `/etc/agents-admin/` | systemd 自动重启 |

> **说明**：两种安装方式使用相同的配置路径和 systemd 服务管理，差别仅在于二进制位于 `/usr/local/bin/`（手动）或 `/usr/bin/`（deb）。

## 方式一：deb 包安装（推荐）

### 1. 构建 deb 包

在开发机上：

```bash
# 构建 Linux 二进制 + deb 包
make deb
```

生成文件：`dist/agents-admin-node-manager_0.9.0_amd64.deb`

### 2. 传输到目标服务器

```bash
scp dist/agents-admin-node-manager_0.9.0_amd64.deb user@<目标IP>:/tmp/
```

### 3. 安装

```bash
sudo dpkg -i /tmp/agents-admin-node-manager_0.9.0_amd64.deb
```

安装后自动完成以下操作：
- 创建系统用户 `agents-admin`
- 创建目录 `/var/lib/agents-admin`、`/var/log/agents-admin`、`/etc/agents-admin`
- 注册并启动 systemd 服务
- 服务自动进入 **Setup 模式**（首次运行，无配置文件）

### 4. 查看 Setup 向导地址

```bash
sudo journalctl -u agents-admin-node-manager -n 20 --no-pager
```

输出示例：
```
====================================================
  Node Manager Setup Wizard
====================================================
  Config will be saved to: /etc/agents-admin/agents-admin.yaml

  Access URLs:
    http://192.168.213.24:15700/setup?token=f0fbbb9d...
    http://localhost:15700/setup?token=f0fbbb9d...

  Token: f0fbbb9d8d6dab0b31eb0718c85eba5c330acd8e7ef9ae9f6eb68ab800e68d07
  Timeout: 30 minutes
====================================================
```

### 5. 浏览器访问配置向导

复制日志中的 **Access URL**（包含 token 参数），在浏览器中打开。

> **安全说明**：URL 中包含一次性 Token，只有持有 Token 才能访问配置页面。Token 仅打印在终端/journal 日志中。

### 6. 完成配置（2 步）

#### Step 1: 连接 API Server
- 输入 **API Server 地址**（如 `https://192.168.213.22:8080`）
- 点击 **「连接并配置」**

系统自动完成以下操作（Bootstrap）：
- 连接 API Server 验证可达性
- 自动下载 CA 证书（HTTPS 模式下，TOFU 信任首次使用）
- 自动生成确定性 Node ID（基于 machine-id）
- 自动检测可写的工作空间目录

#### Step 2: 确认并应用
- 查看完整配置摘要（节点 ID、API Server、工作空间、TLS 状态）
- 点击 **「保存并启动」** 保存配置

配置保存后：
1. 配置文件写入 `/etc/agents-admin/agents-admin.yaml`
2. 进程退出，systemd 自动重启
3. 重启后读取配置文件，进入 **Worker 模式**
4. Web 配置页面不再可用

### 7. 验证运行状态

```bash
# 查看服务状态
sudo systemctl status agents-admin-node-manager

# 查看日志
sudo journalctl -u agents-admin-node-manager -f
```

正常日志示例：
```
Starting NodeManager...
Loaded config from /etc/agents-admin/agents-admin.yaml
TLS enabled, CA: /etc/agents-admin/certs/ca.pem
Node ID: fbc535e1-fea5-5f3c-99ea-df15b4961e8d
API Server: https://192.168.213.22:8080
HTTP-Only mode: task polling via API Server
[nodemanager] started: fbc535e1-fea5-5f3c-99ea-df15b4961e8d
```

## 方式二：二进制安装

### 1. 构建二进制

```bash
make build
# 或仅构建 Linux 版本
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/nodemanager ./cmd/nodemanager
```

### 2. 传输到目标服务器

```bash
scp bin/nodemanager-linux-amd64 user@<目标IP>:/tmp/agents-admin-node-manager
```

### 3. 安装到系统路径

```bash
# 添加执行权限
sudo chmod +x /tmp/agents-admin-node-manager

# 移动到系统 PATH
sudo mv /tmp/agents-admin-node-manager /usr/local/bin/

# 验证安装
agents-admin-node-manager --help
```

### 4. 首次运行（以 root 权限）

```bash
sudo agents-admin-node-manager
```

程序自动检测到无配置文件，启动 Setup 向导。按照终端输出的 URL 在浏览器中完成配置。

> **为什么需要 sudo？**
> 以 root 运行时，Setup 向导会自动完成：
> - 创建 `agents-admin` 系统用户
> - 创建 `/etc/agents-admin/`、`/var/lib/agents-admin/`、`/var/log/agents-admin/` 目录
> - 将配置写入 `/etc/agents-admin/agents-admin.yaml`
> - 自动安装并启用 systemd 服务

### 5. 配置完成后

配置保存后，systemd 服务已自动安装，直接启动：

```bash
sudo systemctl start agents-admin-node-manager
```

## 管理操作

### 查看配置文件

```bash
# 两种安装方式相同
sudo cat /etc/agents-admin/agents-admin.yaml
```

### 重新配置

如需修改配置，有两种方式：

**方式 A：手动编辑配置文件**

```bash
sudo vim /etc/agents-admin/agents-admin.yaml
sudo systemctl restart agents-admin-node-manager
```

**方式 B：重新运行配置向导**

```bash
# deb 安装场景
sudo systemctl stop agents-admin-node-manager
sudo agents-admin-node-manager --reconfigure
# 配置完成后 Ctrl+C，重新启动服务
sudo systemctl start agents-admin-node-manager
```

### 卸载

```bash
# deb 安装
sudo dpkg --purge agents-admin-node-manager
```

## 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--config <dir>` | 搜索 `configs/` | 配置文件目录 |
| `--setup-port <port>` | `15700` | Setup 向导监听端口 |
| `--setup-listen <addr>` | `0.0.0.0` | Setup 向导监听地址 |
| `--reconfigure` | false | 强制重新进入配置向导 |

### 环境变量覆盖

可通过 `/etc/agents-admin/agents-admin.env` 设置环境变量覆盖 YAML 配置：

| 变量 | 说明 |
|------|------|
| `API_SERVER_URL` | 覆盖 `node.api_server_url` |
| `NODE_ID` | 覆盖 `node.id` |
| `WORKSPACE_DIR` | 覆盖 `node.workspace_dir` |
| `SETUP_PORT` | 覆盖 `--setup-port` |
| `SETUP_LISTEN` | 覆盖 `--setup-listen` |
| `TLS_CA_FILE` | 覆盖 `tls.ca_file` |

> 默认情况下 env 文件中的变量都是注释状态，YAML 配置优先。

### deb 场景修改 Setup 端口

```bash
# 方法 1: 编辑环境变量文件
sudo vim /etc/agents-admin/agents-admin.env
# 取消注释并修改：SETUP_PORT=15800

# 方法 2: systemd override
sudo systemctl edit agents-admin-node-manager
# 添加：
# [Service]
# ExecStart=
# ExecStart=/usr/bin/agents-admin-node-manager --config /etc/agents-admin --setup-port 15800
```

## 安全说明

### Setup Token

- Setup 向导启动时生成 **64 字符随机 Token**
- Token 仅打印到终端/systemd journal
- 所有请求必须携带 Token（URL 参数、Cookie 或 Authorization header）
- 首次通过 Token 访问后设置 HttpOnly Cookie（1 小时有效）
- **30 分钟无操作**，Setup 向导自动退出

### 配置完成后

- HTTP 服务器关闭，**不暴露任何端口**
- 配置文件权限为 `0640`（owner + group 可读）

## 配置文件格式

```yaml
# Node Manager Configuration
# Generated by Setup Wizard

node:
    id: fbc535e1-fea5-5f3c-99ea-df15b4961e8d
    api_server_url: https://192.168.213.22:8080
    workspace_dir: /var/lib/agents-admin/workspaces
    labels:
        os: linux
tls:
    enabled: true
    ca_file: /etc/agents-admin/certs/ca.pem
```

| 字段 | 说明 |
|------|------|
| `node.id` | 节点唯一标识（确定性 UUID，基于 machine-id 自动生成） |
| `node.api_server_url` | API Server 地址 |
| `node.workspace_dir` | Agent 工作空间根目录 |
| `node.labels` | 节点标签（用于调度匹配） |
| `tls.enabled` | 是否启用 TLS 客户端验证 |
| `tls.ca_file` | CA 证书文件路径 |

> **说明**：Node Manager 采用 HTTP-Only 架构，通过 HTTP 轮询 API Server 获取任务，无需配置 Redis。

## 故障排查

### Setup 向导无法访问

```bash
# 检查服务是否运行
sudo systemctl status agents-admin-node-manager

# 检查端口是否监听
ss -tlnp | grep 15700

# 查看完整日志获取 Token
sudo journalctl -u agents-admin-node-manager --no-pager | grep token
```

### 配置完成后服务不断重启

```bash
# 查看重启原因
sudo journalctl -u agents-admin-node-manager -n 50

# 常见原因：
# 1. API Server 地址错误 → 手动编辑配置文件修正
# 2. 配置文件权限错误 → sudo chown agents-admin:agents-admin /etc/agents-admin/agents-admin.yaml
```

### 重新开始配置

```bash
# 删除配置文件，服务重启后自动进入 Setup 模式
sudo rm /etc/agents-admin/agents-admin.yaml
sudo systemctl restart agents-admin-node-manager
```
