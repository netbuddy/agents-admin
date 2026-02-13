# Task — 任务管理

## 覆盖功能

| 测试函数 | API 端点 | 验证内容 |
|----------|----------|----------|
| `TestTask_CRUD` | `POST/GET/DELETE /api/v1/tasks` | 任务创建→读取→列表→删除完整生命周期 |
| `TestTask_Pagination` | `GET /api/v1/tasks?limit=&offset=` | 分页查询（limit/offset 参数） |
| `TestTask_SubTasks` | `GET /api/v1/tasks/{id}/subtasks` | 子任务列表查询 |
| `TestTask_SubTasks` | `GET /api/v1/tasks/{id}/tree` | 任务树结构查询 |
| `TestTask_NotFound` | `GET /api/v1/tasks/{id}` | 不存在任务返回 404 |

## 运行

```bash
API_BASE_URL=https://test-server:8080 go test -v ./tests/e2e/task/...
```
