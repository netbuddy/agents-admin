// Package repository Template 相关的存储操作
package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"agents-admin/internal/shared/model"
)

// ============================================================================
// TaskTemplate 操作
// ============================================================================

// CreateTaskTemplate 创建任务模板
func (s *Store) CreateTaskTemplate(ctx context.Context, tmpl *model.TaskTemplate) error {
	promptJSON, _ := json.Marshal(tmpl.PromptTemplate)
	workspaceJSON, _ := json.Marshal(tmpl.DefaultWorkspace)
	securityJSON, _ := json.Marshal(tmpl.DefaultSecurity)
	labelsJSON, _ := json.Marshal(tmpl.DefaultLabels)
	varsJSON, _ := json.Marshal(tmpl.Variables)

	query := s.rebind(`
		INSERT INTO task_templates (id, name, type, description, prompt_template, default_workspace, default_security, default_labels, variables, is_builtin, category, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`)
	_, err := s.db.ExecContext(ctx, query,
		tmpl.ID, tmpl.Name, tmpl.Type, tmpl.Description, promptJSON, workspaceJSON,
		securityJSON, labelsJSON, varsJSON, tmpl.IsBuiltin, tmpl.Category, tmpl.CreatedAt, tmpl.UpdatedAt)
	return err
}

// GetTaskTemplate 获取任务模板
func (s *Store) GetTaskTemplate(ctx context.Context, id string) (*model.TaskTemplate, error) {
	query := s.rebind(`SELECT id, name, type, description, prompt_template, default_workspace, default_security, default_labels, variables, is_builtin, category, created_at, updated_at
			  FROM task_templates WHERE id = $1`)
	tmpl := &model.TaskTemplate{}
	var promptJSON, workspaceJSON, securityJSON, labelsJSON, varsJSON []byte
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&tmpl.ID, &tmpl.Name, &tmpl.Type, &tmpl.Description, &promptJSON, &workspaceJSON,
		&securityJSON, &labelsJSON, &varsJSON, &tmpl.IsBuiltin, &tmpl.Category, &tmpl.CreatedAt, &tmpl.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(promptJSON) > 0 {
		json.Unmarshal(promptJSON, &tmpl.PromptTemplate)
	}
	if len(workspaceJSON) > 0 {
		json.Unmarshal(workspaceJSON, &tmpl.DefaultWorkspace)
	}
	if len(securityJSON) > 0 {
		json.Unmarshal(securityJSON, &tmpl.DefaultSecurity)
	}
	if len(labelsJSON) > 0 {
		json.Unmarshal(labelsJSON, &tmpl.DefaultLabels)
	}
	if len(varsJSON) > 0 {
		json.Unmarshal(varsJSON, &tmpl.Variables)
	}
	return tmpl, nil
}

// ListTaskTemplates 列出任务模板
func (s *Store) ListTaskTemplates(ctx context.Context, category string) ([]*model.TaskTemplate, error) {
	var query string
	var args []interface{}

	if category != "" {
		query = s.rebind(`SELECT id, name, type, description, prompt_template, default_workspace, default_security, default_labels, variables, is_builtin, category, created_at, updated_at
				 FROM task_templates WHERE category = $1 ORDER BY name`)
		args = []interface{}{category}
	} else {
		query = `SELECT id, name, type, description, prompt_template, default_workspace, default_security, default_labels, variables, is_builtin, category, created_at, updated_at
				 FROM task_templates ORDER BY name`
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*model.TaskTemplate
	for rows.Next() {
		tmpl := &model.TaskTemplate{}
		var promptJSON, workspaceJSON, securityJSON, labelsJSON, varsJSON []byte
		if err := rows.Scan(&tmpl.ID, &tmpl.Name, &tmpl.Type, &tmpl.Description, &promptJSON, &workspaceJSON,
			&securityJSON, &labelsJSON, &varsJSON, &tmpl.IsBuiltin, &tmpl.Category, &tmpl.CreatedAt, &tmpl.UpdatedAt); err != nil {
			return nil, err
		}
		if len(promptJSON) > 0 {
			json.Unmarshal(promptJSON, &tmpl.PromptTemplate)
		}
		if len(workspaceJSON) > 0 {
			json.Unmarshal(workspaceJSON, &tmpl.DefaultWorkspace)
		}
		if len(securityJSON) > 0 {
			json.Unmarshal(securityJSON, &tmpl.DefaultSecurity)
		}
		if len(labelsJSON) > 0 {
			json.Unmarshal(labelsJSON, &tmpl.DefaultLabels)
		}
		if len(varsJSON) > 0 {
			json.Unmarshal(varsJSON, &tmpl.Variables)
		}
		templates = append(templates, tmpl)
	}
	return templates, rows.Err()
}

// DeleteTaskTemplate 删除任务模板
func (s *Store) DeleteTaskTemplate(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, s.rebind(`DELETE FROM task_templates WHERE id = $1`), id)
	return err
}

// ============================================================================
// AgentTemplate 操作
// ============================================================================

// CreateAgentTemplate 创建 Agent 模板
func (s *Store) CreateAgentTemplate(ctx context.Context, tmpl *model.AgentTemplate) error {
	personalityJSON, _ := json.Marshal(tmpl.Personality)
	skillsJSON, _ := json.Marshal(tmpl.Skills)
	mcpServersJSON, _ := json.Marshal(tmpl.MCPServers)

	query := s.rebind(`
		INSERT INTO agent_templates (id, name, type, role, description, personality, model, temperature, max_context, skills, mcp_servers, is_builtin, category, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`)
	_, err := s.db.ExecContext(ctx, query,
		tmpl.ID, tmpl.Name, tmpl.Type, tmpl.Role, tmpl.Description, personalityJSON,
		tmpl.Model, tmpl.Temperature, tmpl.MaxContext, skillsJSON, mcpServersJSON,
		tmpl.IsBuiltin, tmpl.Category, tmpl.CreatedAt, tmpl.UpdatedAt)
	return err
}

// GetAgentTemplate 获取 Agent 模板
func (s *Store) GetAgentTemplate(ctx context.Context, id string) (*model.AgentTemplate, error) {
	query := s.rebind(`SELECT id, name, type, role, description, personality, model, temperature, max_context, skills, mcp_servers, is_builtin, category, created_at, updated_at
			  FROM agent_templates WHERE id = $1`)
	tmpl := &model.AgentTemplate{}
	var personalityJSON, skillsJSON, mcpServersJSON []byte
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&tmpl.ID, &tmpl.Name, &tmpl.Type, &tmpl.Role, &tmpl.Description, &personalityJSON,
		&tmpl.Model, &tmpl.Temperature, &tmpl.MaxContext, &skillsJSON, &mcpServersJSON,
		&tmpl.IsBuiltin, &tmpl.Category, &tmpl.CreatedAt, &tmpl.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(personalityJSON) > 0 {
		json.Unmarshal(personalityJSON, &tmpl.Personality)
	}
	if len(skillsJSON) > 0 {
		json.Unmarshal(skillsJSON, &tmpl.Skills)
	}
	if len(mcpServersJSON) > 0 {
		json.Unmarshal(mcpServersJSON, &tmpl.MCPServers)
	}
	return tmpl, nil
}

// ListAgentTemplates 列出 Agent 模板
func (s *Store) ListAgentTemplates(ctx context.Context, category string) ([]*model.AgentTemplate, error) {
	var query string
	var args []interface{}

	if category != "" {
		query = s.rebind(`SELECT id, name, type, role, description, personality, model, temperature, max_context, skills, mcp_servers, is_builtin, category, created_at, updated_at
				 FROM agent_templates WHERE category = $1 ORDER BY name`)
		args = []interface{}{category}
	} else {
		query = `SELECT id, name, type, role, description, personality, model, temperature, max_context, skills, mcp_servers, is_builtin, category, created_at, updated_at
				 FROM agent_templates ORDER BY name`
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*model.AgentTemplate
	for rows.Next() {
		tmpl := &model.AgentTemplate{}
		var personalityJSON, skillsJSON, mcpServersJSON []byte
		if err := rows.Scan(&tmpl.ID, &tmpl.Name, &tmpl.Type, &tmpl.Role, &tmpl.Description, &personalityJSON,
			&tmpl.Model, &tmpl.Temperature, &tmpl.MaxContext, &skillsJSON, &mcpServersJSON,
			&tmpl.IsBuiltin, &tmpl.Category, &tmpl.CreatedAt, &tmpl.UpdatedAt); err != nil {
			return nil, err
		}
		if len(personalityJSON) > 0 {
			json.Unmarshal(personalityJSON, &tmpl.Personality)
		}
		if len(skillsJSON) > 0 {
			json.Unmarshal(skillsJSON, &tmpl.Skills)
		}
		if len(mcpServersJSON) > 0 {
			json.Unmarshal(mcpServersJSON, &tmpl.MCPServers)
		}
		templates = append(templates, tmpl)
	}
	return templates, rows.Err()
}

// UpdateAgentTemplate 更新 Agent 模板
func (s *Store) UpdateAgentTemplate(ctx context.Context, tmpl *model.AgentTemplate) error {
	personalityJSON, _ := json.Marshal(tmpl.Personality)
	skillsJSON, _ := json.Marshal(tmpl.Skills)
	mcpServersJSON, _ := json.Marshal(tmpl.MCPServers)

	query := s.rebind(`
		UPDATE agent_templates
		SET name = $1, type = $2, role = $3, description = $4, personality = $5,
		    model = $6, temperature = $7, max_context = $8, skills = $9, mcp_servers = $10,
		    category = $11, updated_at = $12
		WHERE id = $13
	`)
	_, err := s.db.ExecContext(ctx, query,
		tmpl.Name, tmpl.Type, tmpl.Role, tmpl.Description, personalityJSON,
		tmpl.Model, tmpl.Temperature, tmpl.MaxContext, skillsJSON, mcpServersJSON,
		tmpl.Category, tmpl.UpdatedAt, tmpl.ID)
	return err
}

// DeleteAgentTemplate 删除 Agent 模板
func (s *Store) DeleteAgentTemplate(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, s.rebind(`DELETE FROM agent_templates WHERE id = $1`), id)
	return err
}
