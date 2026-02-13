# Operation — 系统操作与账号管理

## 覆盖功能

### Operations（操作）

| 测试函数 | API 端点 | 验证内容 |
|----------|----------|----------|
| `TestOperation_List` | `GET /api/v1/operations` | 操作列表查询 |
| `TestOperation_Create` | `POST /api/v1/operations` | 创建操作（如 create_account） |
| `TestOperation_Create` | `GET /api/v1/operations/{id}` | 操作详情查询 |

### Accounts（账号，由 Operation 成功后创建）

| 测试函数 | API 端点 | 验证内容 |
|----------|----------|----------|
| `TestAccount_List` | `GET /api/v1/accounts` | 账号列表 |

### Agent Types（Agent 类型，只读）

| 测试函数 | API 端点 | 验证内容 |
|----------|----------|----------|
| `TestAgentType_List` | `GET /api/v1/agent-types` | Agent 类型列表 |

## 运行

```bash
API_BASE_URL=https://test-server:8080 go test -v ./tests/e2e/operation/...
```
