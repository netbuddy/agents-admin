# Health — 健康检查与系统可观测性

## 覆盖功能

| 测试函数 | API 端点 | 验证内容 |
|----------|----------|----------|
| `TestHealth_Check` | `GET /health` | 服务健康检查返回 `{"status":"ok"}` |
| `TestHealth_Metrics` | `GET /metrics` | Prometheus 指标端点可用且非空 |
| `TestHealth_NodeBootstrap` | `GET /api/v1/node-bootstrap` | 节点引导配置端点可达 |

## 运行

```bash
API_BASE_URL=https://test-server:8080 go test -v ./tests/e2e/health/...
```
