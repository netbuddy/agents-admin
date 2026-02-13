package mongostore

import (
	"context"

	"agents-admin/internal/shared/model"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ============================================================================
// SecurityPolicyStore
// ============================================================================

func (s *Store) CreateSecurityPolicy(ctx context.Context, policy *model.SecurityPolicyEntity) error {
	return insertOne(ctx, s.col(ColSecurityPolicies), policy)
}

func (s *Store) GetSecurityPolicy(ctx context.Context, id string) (*model.SecurityPolicyEntity, error) {
	return findOne[model.SecurityPolicyEntity](ctx, s.col(ColSecurityPolicies), bson.D{{Key: "_id", Value: id}})
}

func (s *Store) ListSecurityPolicies(ctx context.Context, category string) ([]*model.SecurityPolicyEntity, error) {
	filter := bson.D{}
	if category != "" {
		filter = append(filter, bson.E{Key: "category", Value: category})
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	return findMany[model.SecurityPolicyEntity](ctx, s.col(ColSecurityPolicies), filter, opts)
}

func (s *Store) DeleteSecurityPolicy(ctx context.Context, id string) error {
	return deleteByID(ctx, s.col(ColSecurityPolicies), id)
}
