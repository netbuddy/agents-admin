# HITL — 人机协作（Human-in-the-Loop）

## 覆盖功能

| 测试函数 | API 端点 | 验证内容 |
|----------|----------|----------|
| `TestHITL_Approvals` | `GET /api/v1/runs/{id}/approvals` | 审批请求列表 |
| `TestHITL_Feedbacks` | `GET /api/v1/runs/{id}/feedbacks` | 人工反馈列表 |
| `TestHITL_Feedbacks` | `POST /api/v1/runs/{id}/feedbacks` | 创建人工反馈 |
| `TestHITL_Interventions` | `GET /api/v1/runs/{id}/interventions` | 干预操作列表 |
| `TestHITL_Confirmations` | `GET /api/v1/runs/{id}/confirmations` | 确认请求列表 |
| `TestHITL_PendingItems` | `GET /api/v1/runs/{id}/hitl/pending` | HITL 待处理汇总 |

## 说明

- 每个测试自动创建临时 Task + Run 作为关联上下文
- 审批决策 (`POST /api/v1/approvals/{id}/decision`) 和确认解决 (`POST /api/v1/confirmations/{id}/resolve`) 需要先有对应请求，此处仅验证列表接口可达

## 运行

```bash
API_BASE_URL=https://test-server:8080 go test -v ./tests/e2e/hitl/...
```
