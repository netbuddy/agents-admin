# Terminal — 终端会话管理

## 覆盖功能

| 测试函数 | API 端点 | 验证内容 |
|----------|----------|----------|
| `TestTerminal_CRUD` | `POST /api/v1/terminal-sessions` | 创建终端会话（依赖节点） |
| `TestTerminal_CRUD` | `GET /api/v1/terminal-sessions/{id}` | 获取会话详情 |
| `TestTerminal_CRUD` | `DELETE /api/v1/terminal-sessions/{id}` | 删除会话 |
| `TestTerminal_List` | `GET /api/v1/terminal-sessions` | 会话列表查询 |

## 前置条件

- 终端会话创建依赖已注册的节点（Node）
- 测试自动注册临时节点并在结束后清理

## 运行

```bash
API_BASE_URL=https://test-server:8080 go test -v ./tests/e2e/terminal/...
```
