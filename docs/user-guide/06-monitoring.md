# 监控与运维

> 本文介绍系统监控、健康检查和运维相关功能。

## 健康检查

### API 端点

```bash
curl -k https://localhost:8080/health
# 响应: {"status": "ok"}
```

用于负载均衡器健康检查或监控告警。

## 工作流监控

### 访问监控页面

1. 点击左侧导航栏的 **「工作流监控」**
2. 页面显示当前系统中的工作流列表
3. 支持实时 WebSocket 推送更新

### 监控内容

| 指标 | 说明 |
|------|------|
| **工作流列表** | 所有认证/执行工作流及其状态 |
| **工作流详情** | 单个工作流的步骤和事件 |
| **系统统计** | 任务数、Run 数、节点数等汇总 |

### 系统统计

通过 API 获取系统整体统计：

```bash
curl -k https://localhost:8080/api/v1/monitor/stats
```

返回任务、Run、节点的数量统计。

## Prometheus 指标

### 端点

```bash
curl -k https://localhost:8080/metrics
```

### 可用指标

系统暴露标准的 Prometheus 指标，包括：

| 指标 | 类型 | 说明 |
|------|------|------|
| `http_requests_total` | Counter | HTTP 请求总数（按方法、路径、状态码） |
| `http_request_duration_seconds` | Histogram | HTTP 请求延迟分布 |
| `go_*` | Gauge | Go 运行时指标（goroutine、内存等） |

### Grafana 集成

可使用 Makefile 启动完整监控栈：

```bash
make monitoring-up
```

访问地址：
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3001 (admin/admin)

## WebSocket 实时监控

### Run 事件流

连接到 Run 的实时事件 WebSocket：

```
wss://localhost:8080/ws/runs/{runId}/events?from_seq=0
```

消息格式：
```json
{"type": "event", "data": {"seq": 1, "type": "message", "timestamp": "...", "payload": {...}}}
{"type": "status", "data": {"status": "done", "finished_at": "..."}}
```

### 全局监控 WebSocket

连接到全局监控 WebSocket：

```
wss://localhost:8080/ws/monitor
```

接收系统级别的实时事件推送。

## 日志

### API Server 日志

API Server 日志输出到 stdout，包含：

- `[run.create.*]` — Run 创建相关日志
- `[node.heartbeat]` — 节点心跳日志
- `[auth]` — 认证操作日志
- `[instance]` — 实例管理日志
- `[proxy]` — 代理管理日志

### NodeManager 日志

NodeManager 日志包含：

- Agent 容器启动/停止
- 事件上报
- 任务领取和执行

## 运维操作

### 启动/停止服务

```bash
# 启动基础设施
make dev-up

# 停止基础设施
make dev-down

# 启动 API Server
make run-api

# 停止 API Server
make stop-api

# 启动 NodeManager
make run-nodemanager

# 停止 NodeManager
make stop-nodemanager
```

### 热加载开发

```bash
# API Server 热加载（修改代码自动重启）
make watch-api

# NodeManager 热加载
make watch-nodemanager
```

### 数据库迁移

迁移文件位于 `deployments/migrations/` 目录，仅适用于 **PostgreSQL**。MongoDB 无需迁移（schema-less，索引在启动时自动创建）。

## API 参考

| 操作 | 方法 | 路径 |
|------|------|------|
| 健康检查 | GET | `/health` |
| Prometheus 指标 | GET | `/metrics` |
| 工作流列表 | GET | `/api/v1/monitor/workflows` |
| 工作流详情 | GET | `/api/v1/monitor/workflows/{type}/{id}` |
| 工作流事件 | GET | `/api/v1/monitor/workflows/{type}/{id}/events` |
| 系统统计 | GET | `/api/v1/monitor/stats` |
| Run 事件 WS | GET | `/ws/runs/{id}/events` |
| 全局监控 WS | GET | `/ws/monitor` |
