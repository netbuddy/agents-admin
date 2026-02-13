# Auth — 认证与授权

## 覆盖功能

| 测试函数 | API 端点 | 验证内容 |
|----------|----------|----------|
| `TestAuth_Me` | `GET /api/v1/auth/me` | 登录后可获取当前用户信息 |
| `TestAuth_RefreshToken` | `POST /api/v1/auth/refresh` | Cookie 令牌刷新 |
| `TestAuth_ChangePassword` | `PUT /api/v1/auth/password` | 密码修改接口可达 |
| `TestAuth_UnauthorizedAccess` | `GET /api/v1/tasks` | 未认证请求被 401 拒绝 |

## 说明

- 所有 E2E 子包共享 `TestMain` 中的自动登录逻辑（Cookie-based）
- 登录凭据通过环境变量 `ADMIN_EMAIL` / `ADMIN_PASSWORD` 配置
- `TestAuth_UnauthorizedAccess` 创建无 Cookie 客户端验证认证中间件

## 运行

```bash
API_BASE_URL=https://test-server:8080 go test -v ./tests/e2e/auth/...
```
