// Package repository Task 相关的存储操作
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"agents-admin/internal/shared/model"
)

// CreateTask 创建任务
func (s *Store) CreateTask(ctx context.Context, task *model.Task) error {
	promptJSON, _ := json.Marshal(task.Prompt)
	workspaceJSON, _ := json.Marshal(task.Workspace)
	securityJSON, _ := json.Marshal(task.Security)
	labelsJSON, _ := json.Marshal(task.Labels)
	contextJSON, _ := json.Marshal(task.Context)

	spec := map[string]interface{}{
		"prompt": task.Prompt,
		"type":   task.Type,
	}
	if task.Workspace != nil {
		spec["workspace"] = task.Workspace
	}
	if task.Security != nil {
		spec["security"] = task.Security
	}
	if task.Labels != nil {
		spec["labels"] = task.Labels
	}
	specJSON, _ := json.Marshal(spec)

	query := s.rebind(`
		INSERT INTO tasks (id, parent_id, name, status, spec, type, prompt, workspace, security, labels, context, template_id, agent_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`)
	_, err := s.db.ExecContext(ctx, query,
		task.ID, task.ParentID, task.Name, task.Status, specJSON, task.Type, promptJSON,
		workspaceJSON, securityJSON, labelsJSON, contextJSON,
		task.TemplateID, task.AgentID, task.CreatedAt, task.UpdatedAt)
	return err
}

// GetTask 获取任务
func (s *Store) GetTask(ctx context.Context, id string) (*model.Task, error) {
	query := s.rebind(`SELECT id, parent_id, name, status, type, prompt, workspace, security, labels, context, template_id, agent_id, created_at, updated_at FROM tasks WHERE id = $1`)
	task := &model.Task{}
	var promptJSON, workspaceJSON, securityJSON, labelsJSON, contextJSON []byte
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID, &task.ParentID, &task.Name, &task.Status, &task.Type, &promptJSON,
		&workspaceJSON, &securityJSON, &labelsJSON, &contextJSON,
		&task.TemplateID, &task.AgentID, &task.CreatedAt, &task.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	unmarshalJSONFields(task, promptJSON, workspaceJSON, securityJSON, labelsJSON, contextJSON)
	return task, nil
}

// scanTask 辅助函数：从数据库行扫描 Task
func scanTask(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.Task, error) {
	task := &model.Task{}
	var promptJSON, workspaceJSON, securityJSON, labelsJSON, contextJSON []byte
	err := scanner.Scan(
		&task.ID, &task.ParentID, &task.Name, &task.Status, &task.Type, &promptJSON,
		&workspaceJSON, &securityJSON, &labelsJSON, &contextJSON,
		&task.TemplateID, &task.AgentID, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		return nil, err
	}
	unmarshalJSONFields(task, promptJSON, workspaceJSON, securityJSON, labelsJSON, contextJSON)
	return task, nil
}

// unmarshalJSONFields 反序列化 Task 的 JSON 字段
func unmarshalJSONFields(task *model.Task, promptJSON, workspaceJSON, securityJSON, labelsJSON, contextJSON []byte) {
	if len(promptJSON) > 0 && string(promptJSON) != "null" {
		json.Unmarshal(promptJSON, &task.Prompt)
	}
	if len(workspaceJSON) > 0 && string(workspaceJSON) != "null" {
		json.Unmarshal(workspaceJSON, &task.Workspace)
	}
	if len(securityJSON) > 0 && string(securityJSON) != "null" {
		json.Unmarshal(securityJSON, &task.Security)
	}
	if len(labelsJSON) > 0 && string(labelsJSON) != "null" {
		json.Unmarshal(labelsJSON, &task.Labels)
	}
	if len(contextJSON) > 0 && string(contextJSON) != "null" {
		json.Unmarshal(contextJSON, &task.Context)
	}
}

// ListTasks 列出任务
func (s *Store) ListTasks(ctx context.Context, status string, limit, offset int) ([]*model.Task, error) {
	var query string
	var args []interface{}

	if status != "" {
		query = s.rebind(`SELECT id, parent_id, name, status, type, prompt, workspace, security, labels, context, template_id, agent_id, created_at, updated_at 
				 FROM tasks WHERE status = $1 
				 ORDER BY created_at DESC LIMIT $2 OFFSET $3`)
		args = []interface{}{status, limit, offset}
	} else {
		query = s.rebind(`SELECT id, parent_id, name, status, type, prompt, workspace, security, labels, context, template_id, agent_id, created_at, updated_at 
				 FROM tasks ORDER BY created_at DESC LIMIT $1 OFFSET $2`)
		args = []interface{}{limit, offset}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*model.Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

// UpdateTaskStatus 更新任务状态
func (s *Store) UpdateTaskStatus(ctx context.Context, id string, status model.TaskStatus) error {
	query := s.rebind(`UPDATE tasks SET status = $1 WHERE id = $2`)
	_, err := s.db.ExecContext(ctx, query, status, id)
	return err
}

// DeleteTask 删除任务
func (s *Store) DeleteTask(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, s.rebind(`DELETE FROM events WHERE run_id IN (SELECT id FROM runs WHERE task_id = $1)`), id)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, s.rebind(`DELETE FROM artifacts WHERE run_id IN (SELECT id FROM runs WHERE task_id = $1)`), id)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, s.rebind(`DELETE FROM runs WHERE task_id = $1`), id)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, s.rebind(`DELETE FROM tasks WHERE id = $1`), id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// UpdateTaskContext 更新任务上下文
func (s *Store) UpdateTaskContext(ctx context.Context, id string, taskContext json.RawMessage) error {
	query := s.rebind(`UPDATE tasks SET context = $1, updated_at = $2 WHERE id = $3`)
	_, err := s.db.ExecContext(ctx, query, taskContext, time.Now(), id)
	return err
}

// ListSubTasks 列出子任务
func (s *Store) ListSubTasks(ctx context.Context, parentID string) ([]*model.Task, error) {
	query := s.rebind(`SELECT id, parent_id, name, status, type, prompt, workspace, security, labels, context, template_id, agent_id, created_at, updated_at 
			  FROM tasks WHERE parent_id = $1 ORDER BY created_at ASC`)
	rows, err := s.db.QueryContext(ctx, query, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*model.Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

// GetTaskTree 获取任务树
func (s *Store) GetTaskTree(ctx context.Context, rootID string) ([]*model.Task, error) {
	if !s.dialect.SupportsRecursiveCTE() {
		return nil, fmt.Errorf("recursive CTE not supported by current database dialect")
	}

	query := s.rebind(`
		WITH RECURSIVE task_tree AS (
			SELECT id, parent_id, name, status, type, prompt, workspace, security, labels, context, template_id, agent_id, created_at, updated_at, 0 as depth
			FROM tasks WHERE id = $1
			UNION ALL
			SELECT t.id, t.parent_id, t.name, t.status, t.type, t.prompt, t.workspace, t.security, t.labels, t.context, t.template_id, t.agent_id, t.created_at, t.updated_at, tt.depth + 1
			FROM tasks t
			INNER JOIN task_tree tt ON t.parent_id = tt.id
		)
		SELECT id, parent_id, name, status, type, prompt, workspace, security, labels, context, template_id, agent_id, created_at, updated_at
		FROM task_tree ORDER BY depth, created_at ASC
	`)
	rows, err := s.db.QueryContext(ctx, query, rootID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*model.Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}
