package mongostore

import (
	"context"
	"time"

	"agents-admin/internal/shared/model"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ============================================================================
// ProxyStore
// ============================================================================

func (s *Store) CreateProxy(ctx context.Context, proxy *model.Proxy) error {
	return insertOne(ctx, s.col(ColProxies), proxy)
}

func (s *Store) GetProxy(ctx context.Context, id string) (*model.Proxy, error) {
	return findOne[model.Proxy](ctx, s.col(ColProxies), bson.D{{Key: "_id", Value: id}})
}

func (s *Store) ListProxies(ctx context.Context) ([]*model.Proxy, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	return findMany[model.Proxy](ctx, s.col(ColProxies), bson.D{}, opts)
}

func (s *Store) GetDefaultProxy(ctx context.Context) (*model.Proxy, error) {
	filter := bson.D{{Key: "is_default", Value: true}}
	return findOne[model.Proxy](ctx, s.col(ColProxies), filter)
}

func (s *Store) UpdateProxy(ctx context.Context, proxy *model.Proxy) error {
	proxy.UpdatedAt = time.Now()
	filter := bson.D{{Key: "_id", Value: proxy.ID}}
	update := bson.D{{Key: "$set", Value: proxy}}
	res, err := s.col(ColProxies).UpdateOne(ctx, filter, update)
	if err != nil {
		return wrapError(err)
	}
	if res.MatchedCount == 0 {
		return wrapError(nil)
	}
	return nil
}

func (s *Store) SetDefaultProxy(ctx context.Context, id string) error {
	// 先清除所有默认
	if err := s.ClearDefaultProxy(ctx); err != nil {
		return err
	}
	return updateFields(ctx, s.col(ColProxies), id, bson.D{
		{Key: "is_default", Value: true},
		{Key: "updated_at", Value: time.Now()},
	})
}

func (s *Store) ClearDefaultProxy(ctx context.Context) error {
	filter := bson.D{{Key: "is_default", Value: true}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "is_default", Value: false}}}}
	_, err := s.col(ColProxies).UpdateMany(ctx, filter, update)
	return err
}

func (s *Store) DeleteProxy(ctx context.Context, id string) error {
	return deleteByID(ctx, s.col(ColProxies), id)
}
