package mongostore

import (
	"context"
	"errors"
	"time"

	"agents-admin/internal/shared/model"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ============================================================================
// AccountStore
// ============================================================================

func (s *Store) CreateAccount(ctx context.Context, account *model.Account) error {
	return insertOne(ctx, s.col(ColAccounts), account)
}

func (s *Store) GetAccount(ctx context.Context, id string) (*model.Account, error) {
	return findOne[model.Account](ctx, s.col(ColAccounts), bson.D{{Key: "_id", Value: id}})
}

func (s *Store) ListAccounts(ctx context.Context) ([]*model.Account, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	return findMany[model.Account](ctx, s.col(ColAccounts), bson.D{}, opts)
}

func (s *Store) UpdateAccountStatus(ctx context.Context, id string, status model.AccountStatus) error {
	return updateFields(ctx, s.col(ColAccounts), id, bson.D{
		{Key: "status", Value: status},
		{Key: "updated_at", Value: time.Now()},
	})
}

func (s *Store) UpdateAccountVolumeArchive(ctx context.Context, id string, archiveKey string) error {
	return updateFields(ctx, s.col(ColAccounts), id, bson.D{
		{Key: "volume_archive_key", Value: archiveKey},
		{Key: "updated_at", Value: time.Now()},
	})
}

func (s *Store) UpdateAccountVolume(ctx context.Context, id string, volumeName string) error {
	return updateFields(ctx, s.col(ColAccounts), id, bson.D{
		{Key: "volume_name", Value: volumeName},
		{Key: "updated_at", Value: time.Now()},
	})
}

func (s *Store) DeleteAccount(ctx context.Context, id string) error {
	return deleteByID(ctx, s.col(ColAccounts), id)
}

// ============================================================================
// AuthTaskStore
// ============================================================================

func (s *Store) CreateAuthTask(ctx context.Context, task *model.AuthTask) error {
	return insertOne(ctx, s.col(ColAuthTasks), task)
}

func (s *Store) GetAuthTask(ctx context.Context, id string) (*model.AuthTask, error) {
	return findOne[model.AuthTask](ctx, s.col(ColAuthTasks), bson.D{{Key: "_id", Value: id}})
}

func (s *Store) GetAuthTaskByAccountID(ctx context.Context, accountID string) (*model.AuthTask, error) {
	filter := bson.D{{Key: "account_id", Value: accountID}}
	opts := options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})
	var result model.AuthTask
	err := s.col(ColAuthTasks).FindOne(ctx, filter, opts).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, wrapError(err)
	}
	return &result, nil
}

func (s *Store) ListRecentAuthTasks(ctx context.Context, limit int) ([]*model.AuthTask, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	return findMany[model.AuthTask](ctx, s.col(ColAuthTasks), bson.D{}, opts)
}

func (s *Store) ListPendingAuthTasks(ctx context.Context, limit int) ([]*model.AuthTask, error) {
	filter := bson.D{{Key: "status", Value: bson.D{{Key: "$in", Value: bson.A{"pending", "assigned"}}}}}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	return findMany[model.AuthTask](ctx, s.col(ColAuthTasks), filter, opts)
}

func (s *Store) ListAuthTasksByNode(ctx context.Context, nodeID string) ([]*model.AuthTask, error) {
	filter := bson.D{{Key: "node_id", Value: nodeID}}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	return findMany[model.AuthTask](ctx, s.col(ColAuthTasks), filter, opts)
}

func (s *Store) UpdateAuthTaskAssignment(ctx context.Context, id string, nodeID string) error {
	return updateFields(ctx, s.col(ColAuthTasks), id, bson.D{
		{Key: "node_id", Value: nodeID},
		{Key: "status", Value: model.AuthTaskStatusAssigned},
		{Key: "updated_at", Value: time.Now()},
	})
}

func (s *Store) UpdateAuthTaskStatus(ctx context.Context, id string, status model.AuthTaskStatus, terminalPort *int, terminalURL *string, containerName *string, message *string) error {
	update := bson.D{
		{Key: "status", Value: status},
		{Key: "updated_at", Value: time.Now()},
	}
	if terminalPort != nil {
		update = append(update, bson.E{Key: "terminal_port", Value: *terminalPort})
	}
	if terminalURL != nil {
		update = append(update, bson.E{Key: "terminal_url", Value: *terminalURL})
	}
	if containerName != nil {
		update = append(update, bson.E{Key: "container_name", Value: *containerName})
	}
	if message != nil {
		update = append(update, bson.E{Key: "message", Value: *message})
	}
	return updateFields(ctx, s.col(ColAuthTasks), id, update)
}
