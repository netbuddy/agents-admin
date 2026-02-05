package operation

import (
	"context"
	"encoding/json"
	"log"

	"agents-admin/internal/shared/model"
)

// handleActionSuccess 处理 Action 成功后的结果
//
// 根据 Operation 类型分发到对应的子 Handler：
//   - 认证操作（oauth/device_code）：auth.HandleAuthSuccess → 创建 Account
//   - API Key：同步完成，无需额外处理
//   - 运行时操作：未来扩展
func (h *Handler) handleActionSuccess(ctx context.Context, op *model.Operation, resultJSON json.RawMessage) {
	switch op.Type {
	case model.OperationTypeOAuth, model.OperationTypeDeviceCode:
		h.authHandler.HandleAuthSuccess(ctx, op, resultJSON)
	case model.OperationTypeAPIKey:
		// API Key 在 CreateOperation 中已同步完成，无需额外处理
	// 未来扩展：
	// case model.OperationTypeRuntimeCreate, ...:
	//     h.runtimeHandler.HandleRuntimeSuccess(ctx, op, resultJSON)
	default:
		log.Printf("[operation] Unhandled success for operation type: %s", op.Type)
	}
}
