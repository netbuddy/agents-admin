package operation

import (
	"encoding/json"
	"fmt"
	"net/http"

	"agents-admin/internal/shared/model"
)

// createOperationRequest 创建操作的请求体
type createOperationRequest struct {
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config"`
	NodeID string          `json:"node_id"`
}

// CreateOperation 创建系统操作（统一入口，按类型分发到子 Handler）
//
// POST /api/v1/operations
// Body: {"type": "oauth|api_key|device_code", "config": {...}, "node_id": "node-001"}
func (h *Handler) CreateOperation(w http.ResponseWriter, r *http.Request) {
	var req createOperationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	opType := model.OperationType(req.Type)

	// 根据类型分发到子 Handler
	switch opType {
	case model.OperationTypeOAuth, model.OperationTypeDeviceCode:
		h.authHandler.CreateAuthOperation(w, r, opType, req.Config, req.NodeID)
	case model.OperationTypeAPIKey:
		h.authHandler.CreateAPIKeyOperation(w, r, req.Config, req.NodeID)
	// 未来扩展：
	// case model.OperationTypeRuntimeCreate, model.OperationTypeRuntimeStart, ...:
	//     h.runtimeHandler.CreateRuntimeOperation(w, r, opType, req.Config, req.NodeID)
	default:
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unsupported operation type: %s", req.Type))
	}
}
