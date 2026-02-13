package mongostore

import (
	"context"

	"agents-admin/internal/shared/model"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ============================================================================
// SkillStore
// ============================================================================

func (s *Store) CreateSkill(ctx context.Context, skill *model.Skill) error {
	return insertOne(ctx, s.col(ColSkills), skill)
}

func (s *Store) GetSkill(ctx context.Context, id string) (*model.Skill, error) {
	return findOne[model.Skill](ctx, s.col(ColSkills), bson.D{{Key: "_id", Value: id}})
}

func (s *Store) ListSkills(ctx context.Context, category string) ([]*model.Skill, error) {
	filter := bson.D{}
	if category != "" {
		filter = append(filter, bson.E{Key: "category", Value: category})
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	return findMany[model.Skill](ctx, s.col(ColSkills), filter, opts)
}

func (s *Store) DeleteSkill(ctx context.Context, id string) error {
	return deleteByID(ctx, s.col(ColSkills), id)
}
