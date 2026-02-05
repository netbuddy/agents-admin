package operation

import (
	"log"
	"net/http"
	"strconv"
)

// ListOperations 列出操作
//
// GET /api/v1/operations
// 可选查询参数: type, status, limit, offset
func (h *Handler) ListOperations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	opType := r.URL.Query().Get("type")
	status := r.URL.Query().Get("status")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = 50
	}

	ops, err := h.store.ListOperations(ctx, opType, status, limit, offset)
	if err != nil {
		log.Printf("[operation] ListOperations error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list operations")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"operations": ops})
}

// GetOperation 获取操作详情（含 Actions）
//
// GET /api/v1/operations/{id}
func (h *Handler) GetOperation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	op, err := h.store.GetOperation(ctx, id)
	if err != nil {
		log.Printf("[operation] GetOperation error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get operation")
		return
	}
	if op == nil {
		writeError(w, http.StatusNotFound, "operation not found")
		return
	}

	// 加载关联的 Actions
	actions, err := h.store.ListActionsByOperation(ctx, id)
	if err != nil {
		log.Printf("[operation] ListActionsByOperation error: %v", err)
	}
	op.Actions = actions

	writeJSON(w, http.StatusOK, op)
}
