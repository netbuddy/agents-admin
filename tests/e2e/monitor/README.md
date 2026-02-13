# Monitor — 监控与可观测性

## 覆盖功能

| 测试函数 | API 端点 | 验证内容 |
|----------|----------|----------|
| `TestMonitor_Stats` | `GET /api/v1/monitor/stats` | 系统统计数据 |
| `TestMonitor_Workflows` | `GET /api/v1/monitor/workflows` | 工作流监控列表 |

## 说明

- WebSocket 端点 (`/ws/monitor`, `/ws/runs/{id}/events`) 需要 WebSocket 客户端测试，不在此 HTTP E2E 范围内
- 工作流详情 (`GET /api/v1/monitor/workflows/{type}/{id}`) 和事件 (`GET /api/v1/monitor/workflows/{type}/{id}/events`) 需要已有工作流数据

## 运行

```bash
API_BASE_URL=https://test-server:8080 go test -v ./tests/e2e/monitor/...
```
