// Package repository HITL (Human-in-the-Loop) 相关的存储操作
package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"agents-admin/internal/shared/model"
)

// ============================================================================
// ApprovalRequest 操作
// ============================================================================

// CreateApprovalRequest 创建审批请求
func (s *Store) CreateApprovalRequest(ctx context.Context, req *model.ApprovalRequest) error {
	query := s.rebind(`
		INSERT INTO approval_requests (id, run_id, type, status, operation, reason, context, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`)
	_, err := s.db.ExecContext(ctx, query,
		req.ID, req.RunID, req.Type, req.Status, req.Operation, req.Reason, req.Context, req.ExpiresAt, req.CreatedAt)
	return err
}

// GetApprovalRequest 获取审批请求
func (s *Store) GetApprovalRequest(ctx context.Context, id string) (*model.ApprovalRequest, error) {
	query := s.rebind(`SELECT id, run_id, type, status, operation, reason, context, expires_at, created_at, resolved_at
			  FROM approval_requests WHERE id = $1`)
	req := &model.ApprovalRequest{}
	var ctxJSON *[]byte
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&req.ID, &req.RunID, &req.Type, &req.Status, &req.Operation, &req.Reason,
		&ctxJSON, &req.ExpiresAt, &req.CreatedAt, &req.ResolvedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if ctxJSON != nil {
		req.Context = *ctxJSON
	}
	return req, nil
}

// ListApprovalRequests 列出 Run 的审批请求
func (s *Store) ListApprovalRequests(ctx context.Context, runID string, status string) ([]*model.ApprovalRequest, error) {
	var query string
	var args []interface{}

	if status != "" {
		query = s.rebind(`SELECT id, run_id, type, status, operation, reason, context, expires_at, created_at, resolved_at
				 FROM approval_requests WHERE run_id = $1 AND status = $2 ORDER BY created_at DESC`)
		args = []interface{}{runID, status}
	} else {
		query = s.rebind(`SELECT id, run_id, type, status, operation, reason, context, expires_at, created_at, resolved_at
				 FROM approval_requests WHERE run_id = $1 ORDER BY created_at DESC`)
		args = []interface{}{runID}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*model.ApprovalRequest
	for rows.Next() {
		req := &model.ApprovalRequest{}
		var ctxJSON *[]byte
		if err := rows.Scan(&req.ID, &req.RunID, &req.Type, &req.Status, &req.Operation, &req.Reason,
			&ctxJSON, &req.ExpiresAt, &req.CreatedAt, &req.ResolvedAt); err != nil {
			return nil, err
		}
		if ctxJSON != nil {
			req.Context = *ctxJSON
		}
		requests = append(requests, req)
	}
	return requests, rows.Err()
}

// UpdateApprovalRequestStatus 更新审批请求状态
func (s *Store) UpdateApprovalRequestStatus(ctx context.Context, id string, status model.ApprovalStatus) error {
	nowExpr := s.now()
	query := s.rebind(`UPDATE approval_requests SET status = $1, resolved_at = ` + nowExpr + ` WHERE id = $2`)
	_, err := s.db.ExecContext(ctx, query, status, id)
	return err
}

// ============================================================================
// ApprovalDecision 操作
// ============================================================================

// CreateApprovalDecision 创建审批决定
func (s *Store) CreateApprovalDecision(ctx context.Context, decision *model.ApprovalDecision) error {
	query := s.rebind(`
		INSERT INTO approval_decisions (id, request_id, decision, decided_by, comment, instructions, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`)
	_, err := s.db.ExecContext(ctx, query,
		decision.ID, decision.RequestID, decision.Decision, decision.DecidedBy,
		decision.Comment, decision.Instructions, decision.CreatedAt)
	return err
}

// ============================================================================
// HumanFeedback 操作
// ============================================================================

// CreateFeedback 创建人工反馈
func (s *Store) CreateFeedback(ctx context.Context, feedback *model.HumanFeedback) error {
	query := s.rebind(`
		INSERT INTO human_feedbacks (id, run_id, type, content, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`)
	_, err := s.db.ExecContext(ctx, query,
		feedback.ID, feedback.RunID, feedback.Type, feedback.Content, feedback.CreatedBy, feedback.CreatedAt)
	return err
}

// ListFeedbacks 列出 Run 的人工反馈
func (s *Store) ListFeedbacks(ctx context.Context, runID string) ([]*model.HumanFeedback, error) {
	query := s.rebind(`SELECT id, run_id, type, content, created_by, created_at, processed_at
			  FROM human_feedbacks WHERE run_id = $1 ORDER BY created_at DESC`)
	rows, err := s.db.QueryContext(ctx, query, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feedbacks []*model.HumanFeedback
	for rows.Next() {
		fb := &model.HumanFeedback{}
		if err := rows.Scan(&fb.ID, &fb.RunID, &fb.Type, &fb.Content, &fb.CreatedBy, &fb.CreatedAt, &fb.ProcessedAt); err != nil {
			return nil, err
		}
		feedbacks = append(feedbacks, fb)
	}
	return feedbacks, rows.Err()
}

// MarkFeedbackProcessed 标记反馈已处理
func (s *Store) MarkFeedbackProcessed(ctx context.Context, id string) error {
	nowExpr := s.now()
	query := s.rebind(`UPDATE human_feedbacks SET processed_at = ` + nowExpr + ` WHERE id = $1`)
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// ============================================================================
// Intervention 操作
// ============================================================================

// CreateIntervention 创建干预
func (s *Store) CreateIntervention(ctx context.Context, intervention *model.Intervention) error {
	query := s.rebind(`
		INSERT INTO interventions (id, run_id, action, reason, parameters, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`)
	_, err := s.db.ExecContext(ctx, query,
		intervention.ID, intervention.RunID, intervention.Action, intervention.Reason,
		intervention.Parameters, intervention.CreatedBy, intervention.CreatedAt)
	return err
}

// ListInterventions 列出 Run 的干预记录
func (s *Store) ListInterventions(ctx context.Context, runID string) ([]*model.Intervention, error) {
	query := s.rebind(`SELECT id, run_id, action, reason, parameters, created_by, created_at, executed_at
			  FROM interventions WHERE run_id = $1 ORDER BY created_at DESC`)
	rows, err := s.db.QueryContext(ctx, query, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var interventions []*model.Intervention
	for rows.Next() {
		i := &model.Intervention{}
		var params *[]byte
		if err := rows.Scan(&i.ID, &i.RunID, &i.Action, &i.Reason, &params, &i.CreatedBy, &i.CreatedAt, &i.ExecutedAt); err != nil {
			return nil, err
		}
		if params != nil {
			i.Parameters = *params
		}
		interventions = append(interventions, i)
	}
	return interventions, rows.Err()
}

// UpdateInterventionExecuted 标记干预已执行
func (s *Store) UpdateInterventionExecuted(ctx context.Context, id string) error {
	nowExpr := s.now()
	query := s.rebind(`UPDATE interventions SET executed_at = ` + nowExpr + ` WHERE id = $1`)
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// ============================================================================
// Confirmation 操作
// ============================================================================

// CreateConfirmation 创建确认请求
func (s *Store) CreateConfirmation(ctx context.Context, confirmation *model.Confirmation) error {
	optionsJSON, _ := json.Marshal(confirmation.Options)
	query := s.rebind(`
		INSERT INTO confirmations (id, run_id, type, message, status, options, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`)
	_, err := s.db.ExecContext(ctx, query,
		confirmation.ID, confirmation.RunID, confirmation.Type, confirmation.Message,
		confirmation.Status, optionsJSON, confirmation.CreatedAt)
	return err
}

// GetConfirmation 获取确认请求
func (s *Store) GetConfirmation(ctx context.Context, id string) (*model.Confirmation, error) {
	query := s.rebind(`SELECT id, run_id, type, message, status, options, selected_option, created_at, resolved_at
			  FROM confirmations WHERE id = $1`)
	c := &model.Confirmation{}
	var optionsJSON []byte
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.RunID, &c.Type, &c.Message, &c.Status, &optionsJSON, &c.SelectedOption, &c.CreatedAt, &c.ResolvedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(optionsJSON) > 0 {
		json.Unmarshal(optionsJSON, &c.Options)
	}
	return c, nil
}

// ListConfirmations 列出 Run 的确认请求
func (s *Store) ListConfirmations(ctx context.Context, runID string, status string) ([]*model.Confirmation, error) {
	var query string
	var args []interface{}

	if status != "" {
		query = s.rebind(`SELECT id, run_id, type, message, status, options, selected_option, created_at, resolved_at
				 FROM confirmations WHERE run_id = $1 AND status = $2 ORDER BY created_at DESC`)
		args = []interface{}{runID, status}
	} else {
		query = s.rebind(`SELECT id, run_id, type, message, status, options, selected_option, created_at, resolved_at
				 FROM confirmations WHERE run_id = $1 ORDER BY created_at DESC`)
		args = []interface{}{runID}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var confirmations []*model.Confirmation
	for rows.Next() {
		c := &model.Confirmation{}
		var optionsJSON []byte
		if err := rows.Scan(&c.ID, &c.RunID, &c.Type, &c.Message, &c.Status, &optionsJSON, &c.SelectedOption, &c.CreatedAt, &c.ResolvedAt); err != nil {
			return nil, err
		}
		if len(optionsJSON) > 0 {
			json.Unmarshal(optionsJSON, &c.Options)
		}
		confirmations = append(confirmations, c)
	}
	return confirmations, rows.Err()
}

// UpdateConfirmationStatus 更新确认请求状态
func (s *Store) UpdateConfirmationStatus(ctx context.Context, id string, status model.ConfirmStatus, selectedOption *string) error {
	nowExpr := s.now()
	query := s.rebind(`UPDATE confirmations SET status = $1, selected_option = $2, resolved_at = ` + nowExpr + ` WHERE id = $3`)
	_, err := s.db.ExecContext(ctx, query, status, selectedOption, id)
	return err
}
