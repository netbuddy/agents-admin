package mongostore

import (
	"context"
	"time"

	"agents-admin/internal/shared/model"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ============================================================================
// RunStore
// ============================================================================

func (s *Store) CreateRun(ctx context.Context, run *model.Run) error {
	return insertOne(ctx, s.col(ColRuns), run)
}

func (s *Store) GetRun(ctx context.Context, id string) (*model.Run, error) {
	return findOne[model.Run](ctx, s.col(ColRuns), bson.D{{Key: "_id", Value: id}})
}

func (s *Store) ListRunsByTask(ctx context.Context, taskID string) ([]*model.Run, error) {
	filter := bson.D{{Key: "task_id", Value: taskID}}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	return findMany[model.Run](ctx, s.col(ColRuns), filter, opts)
}

func (s *Store) ListRunsByNode(ctx context.Context, nodeID string) ([]*model.Run, error) {
	filter := bson.D{
		{Key: "node_id", Value: nodeID},
		{Key: "status", Value: bson.D{{Key: "$in", Value: bson.A{"queued", "assigned", "running"}}}},
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	return findMany[model.Run](ctx, s.col(ColRuns), filter, opts)
}

func (s *Store) ListRunningRuns(ctx context.Context, limit int) ([]*model.Run, error) {
	filter := bson.D{{Key: "status", Value: "running"}}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	return findMany[model.Run](ctx, s.col(ColRuns), filter, opts)
}

func (s *Store) ListQueuedRuns(ctx context.Context, limit int) ([]*model.Run, error) {
	filter := bson.D{{Key: "status", Value: "queued"}}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	return findMany[model.Run](ctx, s.col(ColRuns), filter, opts)
}

func (s *Store) ListStaleQueuedRuns(ctx context.Context, threshold time.Duration) ([]*model.Run, error) {
	cutoff := time.Now().Add(-threshold)
	filter := bson.D{
		{Key: "status", Value: "queued"},
		{Key: "created_at", Value: bson.D{{Key: "$lt", Value: cutoff}}},
	}
	return findMany[model.Run](ctx, s.col(ColRuns), filter)
}

func (s *Store) ResetRunToQueued(ctx context.Context, id string) error {
	return updateFields(ctx, s.col(ColRuns), id, bson.D{
		{Key: "status", Value: "queued"},
		{Key: "node_id", Value: nil},
	})
}

func (s *Store) UpdateRunStatus(ctx context.Context, id string, status model.RunStatus, nodeID *string) error {
	update := bson.D{
		{Key: "status", Value: status},
		{Key: "updated_at", Value: time.Now()},
	}
	if nodeID != nil {
		update = append(update, bson.E{Key: "node_id", Value: *nodeID})
	}
	if status == model.RunStatusRunning {
		now := time.Now()
		update = append(update, bson.E{Key: "started_at", Value: now})
	}
	if status == model.RunStatusDone || status == model.RunStatusFailed ||
		status == model.RunStatusCancelled || status == model.RunStatusTimeout {
		now := time.Now()
		update = append(update, bson.E{Key: "finished_at", Value: now})
	}
	return updateFields(ctx, s.col(ColRuns), id, update)
}

func (s *Store) UpdateRunError(ctx context.Context, id string, errMsg string) error {
	return updateFields(ctx, s.col(ColRuns), id, bson.D{
		{Key: "error", Value: errMsg},
		{Key: "updated_at", Value: time.Now()},
	})
}

func (s *Store) DeleteRun(ctx context.Context, id string) error {
	return deleteByID(ctx, s.col(ColRuns), id)
}
