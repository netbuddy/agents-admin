package mongostore

import (
	"context"
	"time"

	"agents-admin/internal/shared/model"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ============================================================================
// TemplateStore
// ============================================================================

func (s *Store) CreateTaskTemplate(ctx context.Context, tmpl *model.TaskTemplate) error {
	return insertOne(ctx, s.col(ColTaskTemplates), tmpl)
}

func (s *Store) GetTaskTemplate(ctx context.Context, id string) (*model.TaskTemplate, error) {
	return findOne[model.TaskTemplate](ctx, s.col(ColTaskTemplates), bson.D{{Key: "_id", Value: id}})
}

func (s *Store) ListTaskTemplates(ctx context.Context, category string) ([]*model.TaskTemplate, error) {
	filter := bson.D{}
	if category != "" {
		filter = append(filter, bson.E{Key: "category", Value: category})
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	return findMany[model.TaskTemplate](ctx, s.col(ColTaskTemplates), filter, opts)
}

func (s *Store) DeleteTaskTemplate(ctx context.Context, id string) error {
	return deleteByID(ctx, s.col(ColTaskTemplates), id)
}

func (s *Store) CreateAgentTemplate(ctx context.Context, tmpl *model.AgentTemplate) error {
	return insertOne(ctx, s.col(ColAgentTemplates), tmpl)
}

func (s *Store) GetAgentTemplate(ctx context.Context, id string) (*model.AgentTemplate, error) {
	return findOne[model.AgentTemplate](ctx, s.col(ColAgentTemplates), bson.D{{Key: "_id", Value: id}})
}

func (s *Store) ListAgentTemplates(ctx context.Context, category string) ([]*model.AgentTemplate, error) {
	filter := bson.D{}
	if category != "" {
		filter = append(filter, bson.E{Key: "category", Value: category})
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	return findMany[model.AgentTemplate](ctx, s.col(ColAgentTemplates), filter, opts)
}

func (s *Store) UpdateAgentTemplate(ctx context.Context, tmpl *model.AgentTemplate) error {
	tmpl.UpdatedAt = time.Now()
	filter := bson.D{{Key: "_id", Value: tmpl.ID}}
	update := bson.D{{Key: "$set", Value: tmpl}}
	res, err := s.col(ColAgentTemplates).UpdateOne(ctx, filter, update)
	if err != nil {
		return wrapError(err)
	}
	if res.MatchedCount == 0 {
		return wrapError(nil)
	}
	return nil
}

func (s *Store) DeleteAgentTemplate(ctx context.Context, id string) error {
	return deleteByID(ctx, s.col(ColAgentTemplates), id)
}
