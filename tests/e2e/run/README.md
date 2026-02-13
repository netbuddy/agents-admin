# Run — 执行管理与事件流

## 覆盖功能

| 测试函数 | API 端点 | 验证内容 |
|----------|----------|----------|
| `TestRun_Lifecycle` | `POST /api/v1/tasks/{id}/runs` | Run 创建→状态查询→取消完整流程 |
| `TestRun_Lifecycle` | `GET /api/v1/runs/{id}` | Run 详情获取与状态验证 |
| `TestRun_Lifecycle` | `POST /api/v1/runs/{id}/cancel` | Run 取消操作 |
| `TestRun_Events` | `POST /api/v1/runs/{id}/events` | 批量事件上报（run_started, message, tool_use_start） |
| `TestRun_Events` | `GET /api/v1/runs/{id}/events` | 事件获取与计数验证 |
| `TestRun_ListByTask` | `GET /api/v1/tasks/{id}/runs` | 按任务列出所有 Runs |

## 运行

```bash
API_BASE_URL=https://test-server:8080 go test -v ./tests/e2e/run/...
```
