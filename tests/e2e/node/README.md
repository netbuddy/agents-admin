# Node — 节点管理

## 覆盖功能

| 测试函数 | API 端点 | 验证内容 |
|----------|----------|----------|
| `TestNode_HeartbeatAndList` | `POST /api/v1/nodes/heartbeat` | 节点心跳注册（含标签、容量） |
| `TestNode_HeartbeatAndList` | `GET /api/v1/nodes/{id}` | 节点详情获取 |
| `TestNode_HeartbeatAndList` | `GET /api/v1/nodes` | 节点列表查询 |
| `TestNode_EnvConfig` | `GET /api/v1/nodes/{id}/env-config` | 节点环境配置读取 |
| `TestNode_Provisions` | `GET /api/v1/node-provisions` | 节点预配置列表 |
| `TestNode_Delete` | `DELETE /api/v1/nodes/{id}` | 节点删除与确认 |

## 运行

```bash
API_BASE_URL=https://test-server:8080 go test -v ./tests/e2e/node/...
```
