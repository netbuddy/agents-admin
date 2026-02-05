// Package hitl 人在环路（HITL）领域 - HTTP 处理
//
// 本文件实现人在环路相关的 API 端点：
//   - 审批请求管理（ApprovalRequest）
//   - 审批决定处理（ApprovalDecision）
//   - 人工反馈提交（HumanFeedback）
//   - 干预操作执行（Intervention）
//   - 确认请求处理（Confirmation）
package hitl

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"agents-admin/internal/shared/model"
	"agents-admin/internal/shared/storage"
)

// Handler HITL 领域 HTTP 处理器
type Handler struct {
	store storage.PersistentStore
}

// NewHandler 创建 HITL 处理器
func NewHandler(store storage.PersistentStore) *Handler {
	return &Handler{store: store}
}

// RegisterRoutes 注册 HITL 相关路由
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// 审批请求
	mux.HandleFunc("GET /api/v1/runs/{id}/approvals", h.ListApprovalRequests)
	mux.HandleFunc("GET /api/v1/approvals/{id}", h.GetApprovalRequest)
	mux.HandleFunc("POST /api/v1/approvals/{id}/decision", h.CreateApprovalDecision)

	// 人工反馈
	mux.HandleFunc("GET /api/v1/runs/{id}/feedbacks", h.ListFeedbacks)
	mux.HandleFunc("POST /api/v1/runs/{id}/feedbacks", h.CreateFeedback)

	// 干预操作
	mux.HandleFunc("GET /api/v1/runs/{id}/interventions", h.ListInterventions)
	mux.HandleFunc("POST /api/v1/runs/{id}/interventions", h.CreateIntervention)

	// 确认请求
	mux.HandleFunc("GET /api/v1/runs/{id}/confirmations", h.ListConfirmations)
	mux.HandleFunc("GET /api/v1/confirmations/{id}", h.GetConfirmation)
	mux.HandleFunc("POST /api/v1/confirmations/{id}/resolve", h.ResolveConfirmation)

	// HITL 汇总
	mux.HandleFunc("GET /api/v1/runs/{id}/hitl/pending", h.GetPendingHITLItems)
}

// ============================================================================
// 请求结构体
// ============================================================================

type createApprovalDecisionRequest struct {
	Decision     string `json:"decision"`
	Comment      string `json:"comment,omitempty"`
	Instructions string `json:"instructions,omitempty"`
}

type createFeedbackRequest struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type createInterventionRequest struct {
	Action     string          `json:"action"`
	Reason     string          `json:"reason,omitempty"`
	Parameters json.RawMessage `json:"parameters,omitempty"`
}

type resolveConfirmationRequest struct {
	Confirmed      bool   `json:"confirmed"`
	SelectedOption string `json:"selected_option,omitempty"`
}

// ============================================================================
// ApprovalRequest 接口
// ============================================================================

// ListApprovalRequests 列出审批请求
// GET /api/v1/runs/{id}/approvals
func (h *Handler) ListApprovalRequests(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	status := r.URL.Query().Get("status")

	approvals, err := h.store.ListApprovalRequests(r.Context(), runID, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list approval requests")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"approvals": approvals,
		"count":     len(approvals),
	})
}

// GetApprovalRequest 获取审批请求详情
// GET /api/v1/approvals/{id}
func (h *Handler) GetApprovalRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	approval, err := h.store.GetApprovalRequest(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get approval request")
		return
	}
	if approval == nil {
		writeError(w, http.StatusNotFound, "approval request not found")
		return
	}
	writeJSON(w, http.StatusOK, approval)
}

// CreateApprovalDecision 处理审批请求（批准或拒绝）
// POST /api/v1/approvals/{id}/decision
func (h *Handler) CreateApprovalDecision(w http.ResponseWriter, r *http.Request) {
	requestID := r.PathValue("id")

	var req createApprovalDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Decision != "approve" && req.Decision != "reject" {
		writeError(w, http.StatusBadRequest, "decision must be 'approve' or 'reject'")
		return
	}

	approval, err := h.store.GetApprovalRequest(r.Context(), requestID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get approval request")
		return
	}
	if approval == nil {
		writeError(w, http.StatusNotFound, "approval request not found")
		return
	}

	if approval.Status != model.ApprovalStatusPending {
		writeError(w, http.StatusConflict, "approval request already processed")
		return
	}
	if approval.IsExpired() {
		writeError(w, http.StatusConflict, "approval request has expired")
		return
	}

	now := time.Now()
	decision := &model.ApprovalDecision{
		ID:           generateID("decision"),
		RequestID:    requestID,
		Decision:     req.Decision,
		DecidedBy:    "user",
		Comment:      req.Comment,
		Instructions: req.Instructions,
		CreatedAt:    now,
	}

	if err := h.store.CreateApprovalDecision(r.Context(), decision); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create decision")
		return
	}

	var newStatus model.ApprovalStatus
	if req.Decision == "approve" {
		newStatus = model.ApprovalStatusApproved
	} else {
		newStatus = model.ApprovalStatusRejected
	}

	if err := h.store.UpdateApprovalRequestStatus(r.Context(), requestID, newStatus); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update request status")
		return
	}

	writeJSON(w, http.StatusOK, decision)
}

// ============================================================================
// HumanFeedback 接口
// ============================================================================

// ListFeedbacks 列出人工反馈
// GET /api/v1/runs/{id}/feedbacks
func (h *Handler) ListFeedbacks(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")

	feedbacks, err := h.store.ListFeedbacks(r.Context(), runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list feedbacks")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"feedbacks": feedbacks,
		"count":     len(feedbacks),
	})
}

// CreateFeedback 提交人工反馈
// POST /api/v1/runs/{id}/feedbacks
func (h *Handler) CreateFeedback(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")

	run, err := h.store.GetRun(r.Context(), runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get run")
		return
	}
	if run == nil {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}

	var req createFeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	feedbackType := model.FeedbackType(req.Type)
	if feedbackType != model.FeedbackTypeGuidance &&
		feedbackType != model.FeedbackTypeCorrection &&
		feedbackType != model.FeedbackTypeClarification {
		writeError(w, http.StatusBadRequest, "invalid feedback type")
		return
	}

	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}

	now := time.Now()
	feedback := &model.HumanFeedback{
		ID:        generateID("feedback"),
		RunID:     runID,
		Type:      feedbackType,
		Content:   req.Content,
		CreatedBy: "user",
		CreatedAt: now,
	}

	if err := h.store.CreateFeedback(r.Context(), feedback); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create feedback")
		return
	}

	writeJSON(w, http.StatusCreated, feedback)
}

// ============================================================================
// Intervention 接口
// ============================================================================

// ListInterventions 列出干预记录
// GET /api/v1/runs/{id}/interventions
func (h *Handler) ListInterventions(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")

	interventions, err := h.store.ListInterventions(r.Context(), runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list interventions")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"interventions": interventions,
		"count":         len(interventions),
	})
}

// CreateIntervention 创建干预操作
// POST /api/v1/runs/{id}/interventions
func (h *Handler) CreateIntervention(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")

	run, err := h.store.GetRun(r.Context(), runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get run")
		return
	}
	if run == nil {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}

	var req createInterventionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	action := model.InterventionAction(req.Action)
	validActions := map[model.InterventionAction]bool{
		model.InterventionActionPause:  true,
		model.InterventionActionResume: true,
		model.InterventionActionCancel: true,
		model.InterventionActionModify: true,
	}
	if !validActions[action] {
		writeError(w, http.StatusBadRequest, "invalid action")
		return
	}

	switch action {
	case model.InterventionActionPause:
		if run.Status != model.RunStatusRunning {
			writeError(w, http.StatusConflict, "can only pause running runs")
			return
		}
	case model.InterventionActionResume:
		if run.Status != model.RunStatusPaused {
			writeError(w, http.StatusConflict, "can only resume paused runs")
			return
		}
	case model.InterventionActionCancel:
		if run.Status != model.RunStatusRunning && run.Status != model.RunStatusQueued && run.Status != model.RunStatusPaused {
			writeError(w, http.StatusConflict, "cannot cancel this run")
			return
		}
	}

	now := time.Now()
	intervention := &model.Intervention{
		ID:         generateID("intervention"),
		RunID:      runID,
		Action:     action,
		Reason:     req.Reason,
		Parameters: req.Parameters,
		CreatedBy:  "user",
		CreatedAt:  now,
	}

	if err := h.store.CreateIntervention(r.Context(), intervention); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create intervention")
		return
	}

	var newStatus model.RunStatus
	switch action {
	case model.InterventionActionPause:
		newStatus = model.RunStatusPaused
	case model.InterventionActionResume:
		newStatus = model.RunStatusRunning
	case model.InterventionActionCancel:
		newStatus = model.RunStatusCancelled
	}

	if newStatus != "" {
		if err := h.store.UpdateRunStatus(r.Context(), runID, newStatus, nil); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update run status")
			return
		}
	}

	intervention.ExecutedAt = &now
	h.store.UpdateInterventionExecuted(r.Context(), intervention.ID)

	writeJSON(w, http.StatusCreated, intervention)
}

// ============================================================================
// Confirmation 接口
// ============================================================================

// ListConfirmations 列出确认请求
// GET /api/v1/runs/{id}/confirmations
func (h *Handler) ListConfirmations(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	status := r.URL.Query().Get("status")

	confirmations, err := h.store.ListConfirmations(r.Context(), runID, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list confirmations")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"confirmations": confirmations,
		"count":         len(confirmations),
	})
}

// GetConfirmation 获取确认请求详情
// GET /api/v1/confirmations/{id}
func (h *Handler) GetConfirmation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	confirmation, err := h.store.GetConfirmation(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get confirmation")
		return
	}
	if confirmation == nil {
		writeError(w, http.StatusNotFound, "confirmation not found")
		return
	}
	writeJSON(w, http.StatusOK, confirmation)
}

// ResolveConfirmation 处理确认请求
// POST /api/v1/confirmations/{id}/resolve
func (h *Handler) ResolveConfirmation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req resolveConfirmationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	confirmation, err := h.store.GetConfirmation(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get confirmation")
		return
	}
	if confirmation == nil {
		writeError(w, http.StatusNotFound, "confirmation not found")
		return
	}

	if confirmation.Status != model.ConfirmStatusPending {
		writeError(w, http.StatusConflict, "confirmation already processed")
		return
	}

	var newStatus model.ConfirmStatus
	if req.Confirmed {
		newStatus = model.ConfirmStatusConfirmed
	} else {
		newStatus = model.ConfirmStatusCancelled
	}

	selectedOption := req.SelectedOption
	if err := h.store.UpdateConfirmationStatus(r.Context(), id, newStatus, &selectedOption); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update confirmation")
		return
	}

	confirmation.Status = newStatus
	confirmation.SelectedOption = &selectedOption
	now := time.Now()
	confirmation.ResolvedAt = &now

	writeJSON(w, http.StatusOK, confirmation)
}

// ============================================================================
// HITL 汇总
// ============================================================================

// GetPendingHITLItems 获取 Run 的所有待处理 HITL 项目
// GET /api/v1/runs/{id}/hitl/pending
func (h *Handler) GetPendingHITLItems(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")

	approvals, _ := h.store.ListApprovalRequests(r.Context(), runID, "pending")
	confirmations, _ := h.store.ListConfirmations(r.Context(), runID, "pending")

	total := len(approvals) + len(confirmations)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"approvals":     approvals,
		"confirmations": confirmations,
		"total":         total,
	})
}

// ============================================================================
// 工具函数
// ============================================================================

func generateID(prefix string) string {
	b := make([]byte, 6)
	rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
