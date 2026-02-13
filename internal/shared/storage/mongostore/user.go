package mongostore

import (
	"context"
	"time"

	"agents-admin/internal/shared/model"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ============================================================================
// UserStore
// ============================================================================

func (s *Store) CreateUser(ctx context.Context, user *model.User) error {
	return insertOne(ctx, s.col(ColUsers), user)
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	return findOne[model.User](ctx, s.col(ColUsers), bson.D{{Key: "email", Value: email}})
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	return findOne[model.User](ctx, s.col(ColUsers), bson.D{{Key: "_id", Value: id}})
}

func (s *Store) UpdateUserPassword(ctx context.Context, id, passwordHash string) error {
	return updateFields(ctx, s.col(ColUsers), id, bson.D{
		{Key: "password_hash", Value: passwordHash},
		{Key: "updated_at", Value: time.Now()},
	})
}

func (s *Store) ListUsers(ctx context.Context) ([]*model.User, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	return findMany[model.User](ctx, s.col(ColUsers), bson.D{}, opts)
}
