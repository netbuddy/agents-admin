# Template — 模板与配置资源

## 覆盖功能

本目录覆盖五类模板/配置资源的完整 CRUD：

### Task Templates（任务模板）

| 测试函数 | API 端点 | 验证内容 |
|----------|----------|----------|
| `TestTaskTemplate_CRUD` | `POST /api/v1/task-templates` | 创建任务模板 |
| `TestTaskTemplate_CRUD` | `GET /api/v1/task-templates/{id}` | 获取模板详情 |
| `TestTaskTemplate_CRUD` | `GET /api/v1/task-templates` | 模板列表 |

### Agent Templates（Agent 模板）

| 测试函数 | API 端点 | 验证内容 |
|----------|----------|----------|
| `TestAgentTemplate_CRUD` | `POST /api/v1/agent-templates` | 创建 Agent 模板 |
| `TestAgentTemplate_CRUD` | `GET /api/v1/agent-templates/{id}` | 获取模板详情 |
| `TestAgentTemplate_CRUD` | `GET /api/v1/agent-templates` | 模板列表 |

### Skills（技能）

| 测试函数 | API 端点 | 验证内容 |
|----------|----------|----------|
| `TestSkill_CRUD` | `POST /api/v1/skills` | 创建技能 |
| `TestSkill_CRUD` | `GET /api/v1/skills/{id}` | 获取技能详情 |
| `TestSkill_CRUD` | `GET /api/v1/skills` | 技能列表 |

### MCP Servers

| 测试函数 | API 端点 | 验证内容 |
|----------|----------|----------|
| `TestMCPServer_CRUD` | `POST /api/v1/mcp-servers` | 创建 MCP Server |
| `TestMCPServer_CRUD` | `GET /api/v1/mcp-servers/{id}` | 获取详情 |
| `TestMCPServer_CRUD` | `GET /api/v1/mcp-servers` | 列表 |

### Security Policies（安全策略）

| 测试函数 | API 端点 | 验证内容 |
|----------|----------|----------|
| `TestSecurityPolicy_CRUD` | `POST /api/v1/security-policies` | 创建安全策略 |
| `TestSecurityPolicy_CRUD` | `GET /api/v1/security-policies/{id}` | 获取详情 |
| `TestSecurityPolicy_CRUD` | `GET /api/v1/security-policies` | 列表 |

## 运行

```bash
API_BASE_URL=https://test-server:8080 go test -v ./tests/e2e/template/...
```
