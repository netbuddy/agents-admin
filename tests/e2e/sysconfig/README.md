# SysConfig — 系统配置

## 覆盖功能

| 测试函数 | API 端点 | 验证内容 |
|----------|----------|----------|
| `TestSysConfig_Get` | `GET /api/v1/config` | 获取系统配置 |
| `TestSysConfig_Update` | `PUT /api/v1/config` | 更新系统配置（幂等写回） |

## 运行

```bash
API_BASE_URL=https://test-server:8080 go test -v ./tests/e2e/sysconfig/...
```
