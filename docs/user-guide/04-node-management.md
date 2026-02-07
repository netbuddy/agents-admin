# 节点管理

> 本文介绍如何管理 Agent 执行节点。

## 概述

节点是运行 Agent 的物理或虚拟机器。每个节点上运行一个 **NodeManager** 进程，负责：

- 定期向 API Server 发送心跳
- 领取并执行调度到本节点的任务
- 管理 Docker 容器（实例启停）
- 上报执行事件

## 节点注册

节点无需手动创建，**NodeManager 启动后会自动注册**。

### 启动 NodeManager

```bash
# 基本启动
API_SERVER_URL="http://localhost:8080" \
NODE_ID="node-01" \
WORKSPACE_DIR="/tmp/agents-workspaces" \
go run ./cmd/nodemanager

# 或使用 Makefile
make run-nodemanager
```

### 环境变量

| 变量 | 必填 | 说明 | 默认值 |
|------|------|------|--------|
| `API_SERVER_URL` | 是 | API Server 地址 | `http://localhost:8080` |
| `NODE_ID` | 是 | 节点唯一标识 | `dev-node-01` |
| `WORKSPACE_DIR` | 否 | 工作空间目录 | `/tmp/agents-workspaces` |

## 查看节点列表

1. 点击左侧导航栏的 **「节点管理」**
2. 页面显示所有注册过的节点
3. 每个节点卡片显示：ID、状态、标签、容量、最近心跳时间

## 节点状态

| 状态 | 说明 |
|------|------|
| **在线** (online) | 节点正常运行，定期发送心跳 |
| **离线** (offline) | 节点停止发送心跳，超时判定为离线 |
| **维护中** (draining) | 节点正在排空任务，不接受新任务 |

### 状态判定规则

- 心跳间隔：NodeManager 每 **30 秒** 发送一次心跳
- 超时判定：超过 **90 秒** 无心跳则判定为离线

## 节点操作

### 查看节点详情

1. 点击节点卡片查看详情
2. 可以看到：
   - 节点基本信息（ID、状态、创建时间）
   - 标签（labels）：如 `os=linux`, `arch=amd64`
   - 容量（capacity）：`max_concurrent` 最大并发数
   - 当前执行的 Run 列表

### 配置节点环境

1. 在节点详情中，点击 **「环境配置」**
2. 可配置：
   - **代理设置**：为节点上的 Agent 容器配置网络代理
   - **环境变量**：自定义环境变量
3. 点击 **「保存」** 应用配置

### 测试代理连接

1. 在节点环境配置中
2. 点击 **「测试代理」** 按钮
3. 系统会在节点上测试代理连接是否正常

### 更新节点

1. 在节点卡片上点击编辑
2. 可更新节点的标签和状态
3. 将节点设为 `draining` 状态可停止接受新任务

### 删除节点

1. 确保节点上没有运行中的任务
2. 点击 **删除按钮**
3. 确认删除

## 调度机制

### 调度流程

```
1. 用户创建 Run
2. Run 加入调度队列（Redis Streams）
3. Scheduler 从队列中取出 Run
4. Scheduler 根据标签匹配选择节点
5. 选择负载最低的节点
6. 检查节点容量是否允许
7. 将 Run 分配到目标节点
8. NodeManager 领取并执行
```

### 调度策略

| 策略 | 说明 |
|------|------|
| **标签匹配** | 任务标签需与节点标签匹配 |
| **负载均衡** | 优先选择当前负载最低的节点 |
| **容量检查** | 不超过节点的 `max_concurrent` 限制 |

## API 参考

| 操作 | 方法 | 路径 |
|------|------|------|
| 节点心跳 | POST | `/api/v1/nodes/heartbeat` |
| 列出节点 | GET | `/api/v1/nodes` |
| 获取节点 | GET | `/api/v1/nodes/{id}` |
| 更新节点 | PATCH | `/api/v1/nodes/{id}` |
| 删除节点 | DELETE | `/api/v1/nodes/{id}` |
| 节点 Run 列表 | GET | `/api/v1/nodes/{id}/runs` |
| 获取环境配置 | GET | `/api/v1/nodes/{id}/env-config` |
| 更新环境配置 | PUT | `/api/v1/nodes/{id}/env-config` |
| 测试代理 | POST | `/api/v1/nodes/{id}/env-config/test-proxy` |
