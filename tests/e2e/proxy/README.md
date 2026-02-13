# Proxy — 代理管理

## 覆盖功能

| 测试函数 | API 端点 | 验证内容 |
|----------|----------|----------|
| `TestProxy_CRUD` | `POST /api/v1/proxies` | 创建 HTTP 代理 |
| `TestProxy_CRUD` | `GET /api/v1/proxies/{id}` | 获取代理详情 |
| `TestProxy_CRUD` | `PUT /api/v1/proxies/{id}` | 更新代理配置 |
| `TestProxy_CRUD` | `GET /api/v1/proxies` | 代理列表 |
| `TestProxy_CRUD` | `DELETE /api/v1/proxies/{id}` | 删除代理 |
| `TestProxy_Test` | `POST /api/v1/proxies/{id}/test` | 代理连通性测试 |

## 运行

```bash
API_BASE_URL=https://test-server:8080 go test -v ./tests/e2e/proxy/...
```
