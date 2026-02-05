package operation

import (
	"encoding/json"
	"log"
	"net/http"

	"agents-admin/internal/shared/model"
)

// updateActionRequest Action 状态更新请求
type updateActionRequest struct {
	Status   string          `json:"status"`
	Phase    string          `json:"phase,omitempty"`
	Message  string          `json:"message,omitempty"`
	Progress int             `json:"progress"`
	Result   json.RawMessage `json:"result,omitempty"`
	Error    string          `json:"error,omitempty"`
}

// GetAction 获取 Action 详情
//
// GET /api/v1/actions/{id}
func (h *Handler) GetAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	action, err := h.store.GetActionWithOperation(ctx, id)
	if err != nil {
		log.Printf("[operation] GetActionWithOperation error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get action")
		return
	}
	if action == nil {
		writeError(w, http.StatusNotFound, "action not found")
		return
	}

	writeJSON(w, http.StatusOK, action)
}

// UpdateAction 更新 Action 状态（由节点管理器调用）
//
// PATCH /api/v1/actions/{id}
// Body: {"status": "running|waiting|success|failed|timeout", "progress": 50, "result": {...}}
func (h *Handler) UpdateAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	var req updateActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// 获取 Action（含 Operation）
	action, err := h.store.GetActionWithOperation(ctx, id)
	if err != nil {
		log.Printf("[operation] GetActionWithOperation error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get action")
		return
	}
	if action == nil {
		writeError(w, http.StatusNotFound, "action not found")
		return
	}

	// 如果 Action 已终态，不允许更新
	if action.Status.IsTerminal() {
		writeError(w, http.StatusConflict, "action is already in terminal state")
		return
	}

	newStatus := model.ActionStatus(req.Status)
	newPhase := model.ActionPhase(req.Phase)

	// 更新 Action
	if err := h.store.UpdateActionStatus(ctx, id, newStatus, newPhase, req.Message, req.Progress, req.Result, req.Error); err != nil {
		log.Printf("[operation] UpdateActionStatus error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update action")
		return
	}

	// 如果到达终态，更新 Operation 状态并处理结果
	if newStatus.IsTerminal() {
		var opStatus model.OperationStatus
		switch newStatus {
		case model.ActionStatusSuccess:
			opStatus = model.OperationStatusCompleted
		default:
			opStatus = model.OperationStatusFailed
		}
		if err := h.store.UpdateOperationStatus(ctx, action.OperationID, opStatus); err != nil {
			log.Printf("[operation] UpdateOperationStatus error: %v", err)
		}

		// 处理操作结果（成功时创建 Account 等）
		if newStatus == model.ActionStatusSuccess && action.Operation != nil {
			h.handleActionSuccess(ctx, action.Operation, req.Result)
		}
	} else if newStatus == model.ActionStatusRunning {
		// 首次开始执行时更新 Operation 为 in_progress
		if action.Operation != nil && action.Operation.Status == model.OperationStatusPending {
			if err := h.store.UpdateOperationStatus(ctx, action.OperationID, model.OperationStatusInProgress); err != nil {
				log.Printf("[operation] UpdateOperationStatus error: %v", err)
			}
		}
	}

	log.Printf("[operation] Action %s updated: status=%s phase=%s (progress=%d)", id, req.Status, req.Phase, req.Progress)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":      id,
		"status":  req.Status,
		"phase":   req.Phase,
		"message": req.Message,
	})
}
