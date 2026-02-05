package operation

import (
	"log"
	"net/http"

	"agents-admin/internal/shared/model"
)

// GetNodeActions 获取分配给节点的待处理 Action（节点管理器轮询）
//
// GET /api/v1/nodes/{id}/actions
// 可选查询参数: status（默认 assigned）
//
// 返回包含 Operation 信息的 Action 列表，节点管理器根据
// Operation.type 分发到不同的 Handler（auth/runtime/...）
func (h *Handler) GetNodeActions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	nodeID := r.PathValue("id")

	status := r.URL.Query().Get("status")
	if status == "" {
		status = string(model.ActionStatusAssigned)
	}

	actions, err := h.store.ListActionsByNode(ctx, nodeID, status)
	if err != nil {
		log.Printf("[operation] ListActionsByNode error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list actions")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"actions": actions})
}
