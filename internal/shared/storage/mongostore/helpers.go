package mongostore

import (
	"context"
	"errors"

	"agents-admin/internal/shared/storage"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// wrapError 将 MongoDB 错误转换为领域错误
func wrapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return storage.ErrNotFound
	}
	if mongo.IsDuplicateKeyError(err) {
		return storage.ErrDuplicate
	}
	return err
}

// findOne 查找单个文档并解码到 result
// 文档不存在时返回 (nil, nil)，与 SQL 实现的 sql.ErrNoRows → (nil, nil) 行为一致
func findOne[T any](ctx context.Context, col *mongo.Collection, filter bson.D) (*T, error) {
	var result T
	err := col.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, wrapError(err)
	}
	return &result, nil
}

// findMany 查找多个文档
func findMany[T any](ctx context.Context, col *mongo.Collection, filter bson.D, opts ...options.Lister[options.FindOptions]) ([]*T, error) {
	cursor, err := col.Find(ctx, filter, opts...)
	if err != nil {
		return nil, wrapError(err)
	}
	defer cursor.Close(ctx)

	var results []*T
	for cursor.Next(ctx) {
		var item T
		if err := cursor.Decode(&item); err != nil {
			return nil, err
		}
		results = append(results, &item)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	if results == nil {
		results = []*T{}
	}
	return results, nil
}

// insertOne 插入单个文档
func insertOne(ctx context.Context, col *mongo.Collection, doc interface{}) error {
	_, err := col.InsertOne(ctx, doc)
	return wrapError(err)
}

// deleteByID 按 _id 删除
func deleteByID(ctx context.Context, col *mongo.Collection, id string) error {
	res, err := col.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})
	if err != nil {
		return wrapError(err)
	}
	if res.DeletedCount == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// updateFields 按 _id 更新指定字段
func updateFields(ctx context.Context, col *mongo.Collection, id string, update bson.D) error {
	res, err := col.UpdateOne(ctx, bson.D{{Key: "_id", Value: id}}, bson.D{{Key: "$set", Value: update}})
	if err != nil {
		return wrapError(err)
	}
	if res.MatchedCount == 0 {
		return storage.ErrNotFound
	}
	return nil
}
