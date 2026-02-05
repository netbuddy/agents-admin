// Package repository Skill 相关的存储操作
package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"agents-admin/internal/shared/model"
)

// CreateSkill 创建技能
func (s *Store) CreateSkill(ctx context.Context, skill *model.Skill) error {
	tagsJSON, _ := json.Marshal(skill.Tags)

	query := s.rebind(`
		INSERT INTO skills (id, name, category, level, description, instructions, tools, examples, parameters, source, author_id, registry_id, version, is_builtin, tags, use_count, rating, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`)
	_, err := s.db.ExecContext(ctx, query,
		skill.ID, skill.Name, skill.Category, skill.Level, skill.Description, skill.Instructions,
		skill.Tools, skill.Examples, skill.Parameters, skill.Source, skill.AuthorID, skill.RegistryID,
		skill.Version, skill.IsBuiltin, tagsJSON, skill.UseCount, skill.Rating, skill.CreatedAt, skill.UpdatedAt)
	return err
}

// GetSkill 获取技能
func (s *Store) GetSkill(ctx context.Context, id string) (*model.Skill, error) {
	query := s.rebind(`SELECT id, name, category, level, description, instructions, tools, examples, parameters, source, author_id, registry_id, version, is_builtin, tags, use_count, rating, created_at, updated_at
			  FROM skills WHERE id = $1`)
	skill := &model.Skill{}
	var tagsJSON, tools, examples, params *[]byte
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&skill.ID, &skill.Name, &skill.Category, &skill.Level, &skill.Description, &skill.Instructions,
		&tools, &examples, &params, &skill.Source, &skill.AuthorID, &skill.RegistryID,
		&skill.Version, &skill.IsBuiltin, &tagsJSON, &skill.UseCount, &skill.Rating, &skill.CreatedAt, &skill.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if tools != nil {
		skill.Tools = *tools
	}
	if examples != nil {
		skill.Examples = *examples
	}
	if params != nil {
		skill.Parameters = *params
	}
	if tagsJSON != nil {
		json.Unmarshal(*tagsJSON, &skill.Tags)
	}
	return skill, nil
}

// ListSkills 列出技能
func (s *Store) ListSkills(ctx context.Context, category string) ([]*model.Skill, error) {
	var query string
	var args []interface{}

	if category != "" {
		query = s.rebind(`SELECT id, name, category, level, description, instructions, tools, examples, parameters, source, author_id, registry_id, version, is_builtin, tags, use_count, rating, created_at, updated_at
				 FROM skills WHERE category = $1 ORDER BY name`)
		args = []interface{}{category}
	} else {
		query = `SELECT id, name, category, level, description, instructions, tools, examples, parameters, source, author_id, registry_id, version, is_builtin, tags, use_count, rating, created_at, updated_at
				 FROM skills ORDER BY name`
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []*model.Skill
	for rows.Next() {
		skill := &model.Skill{}
		var tagsJSON, tools, examples, params *[]byte
		if err := rows.Scan(&skill.ID, &skill.Name, &skill.Category, &skill.Level, &skill.Description, &skill.Instructions,
			&tools, &examples, &params, &skill.Source, &skill.AuthorID, &skill.RegistryID,
			&skill.Version, &skill.IsBuiltin, &tagsJSON, &skill.UseCount, &skill.Rating, &skill.CreatedAt, &skill.UpdatedAt); err != nil {
			return nil, err
		}
		if tools != nil {
			skill.Tools = *tools
		}
		if examples != nil {
			skill.Examples = *examples
		}
		if params != nil {
			skill.Parameters = *params
		}
		if tagsJSON != nil {
			json.Unmarshal(*tagsJSON, &skill.Tags)
		}
		skills = append(skills, skill)
	}
	return skills, rows.Err()
}

// DeleteSkill 删除技能
func (s *Store) DeleteSkill(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, s.rebind(`DELETE FROM skills WHERE id = $1`), id)
	return err
}
