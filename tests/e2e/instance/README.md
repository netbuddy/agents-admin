# Instance — 实例管理

## 覆盖功能

| 测试函数 | API 端点 | 验证内容 |
|----------|----------|----------|
| `TestInstance_CRUD` | `POST /api/v1/instances` | 创建实例（依赖节点） |
| `TestInstance_CRUD` | `GET /api/v1/instances/{id}` | 获取实例详情 |
| `TestInstance_CRUD` | `DELETE /api/v1/instances/{id}` | 删除实例 |
| `TestInstance_List` | `GET /api/v1/instances` | 实例列表查询 |
| `TestInstance_StartStop` | `POST /api/v1/instances/{id}/start` | 启动实例 |
| `TestInstance_StartStop` | `POST /api/v1/instances/{id}/stop` | 停止实例 |

## 前置条件

- 实例创建依赖已注册的节点（Node）
- 测试自动注册临时节点并在结束后清理

## 运行

```bash
API_BASE_URL=https://test-server:8080 go test -v ./tests/e2e/instance/...
```
