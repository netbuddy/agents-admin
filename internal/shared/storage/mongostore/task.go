package mongostore

import (
	"context"
	"encoding/json"

	"agents-admin/internal/shared/model"
	"agents-admin/internal/shared/storagetypes"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ============================================================================
// TaskStore
// ============================================================================

func (s *Store) CreateTask(ctx context.Context, task *model.Task) error {
	return insertOne(ctx, s.col(ColTasks), task)
}

func (s *Store) GetTask(ctx context.Context, id string) (*model.Task, error) {
	return findOne[model.Task](ctx, s.col(ColTasks), bson.D{{Key: "_id", Value: id}})
}

func (s *Store) ListTasks(ctx context.Context, status string, limit, offset int) ([]*model.Task, error) {
	filter := bson.D{}
	if status != "" {
		filter = append(filter, bson.E{Key: "status", Value: status})
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	if offset > 0 {
		opts.SetSkip(int64(offset))
	}
	return findMany[model.Task](ctx, s.col(ColTasks), filter, opts)
}

func (s *Store) ListTasksWithFilter(ctx context.Context, tf storagetypes.TaskFilter) ([]*model.Task, int, error) {
	filter := bson.D{}
	if tf.Status != "" {
		filter = append(filter, bson.E{Key: "status", Value: tf.Status})
	}
	if tf.Search != "" {
		filter = append(filter, bson.E{Key: "name", Value: bson.D{{Key: "$regex", Value: tf.Search}, {Key: "$options", Value: "i"}}})
	}
	if !tf.Since.IsZero() {
		filter = append(filter, bson.E{Key: "created_at", Value: bson.D{{Key: "$gte", Value: tf.Since}}})
	}
	if !tf.Until.IsZero() {
		filter = append(filter, bson.E{Key: "created_at", Value: bson.D{{Key: "$lte", Value: tf.Until}}})
	}

	// Count total
	total, err := s.col(ColTasks).CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if tf.Limit > 0 {
		opts.SetLimit(int64(tf.Limit))
	}
	if tf.Offset > 0 {
		opts.SetSkip(int64(tf.Offset))
	}

	tasks, err := findMany[model.Task](ctx, s.col(ColTasks), filter, opts)
	if err != nil {
		return nil, 0, err
	}
	return tasks, int(total), nil
}

func (s *Store) UpdateTaskStatus(ctx context.Context, id string, status model.TaskStatus) error {
	return updateFields(ctx, s.col(ColTasks), id, bson.D{{Key: "status", Value: status}})
}

func (s *Store) DeleteTask(ctx context.Context, id string) error {
	return deleteByID(ctx, s.col(ColTasks), id)
}

func (s *Store) UpdateTaskContext(ctx context.Context, id string, taskContext json.RawMessage) error {
	return updateFields(ctx, s.col(ColTasks), id, bson.D{{Key: "context", Value: taskContext}})
}

func (s *Store) ListSubTasks(ctx context.Context, parentID string) ([]*model.Task, error) {
	filter := bson.D{{Key: "parent_id", Value: parentID}}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})
	return findMany[model.Task](ctx, s.col(ColTasks), filter, opts)
}

func (s *Store) GetTaskTree(ctx context.Context, rootID string) ([]*model.Task, error) {
	// 获取根任务
	root, err := s.GetTask(ctx, rootID)
	if err != nil {
		return nil, err
	}
	result := []*model.Task{root}

	// BFS 获取子任务
	queue := []string{rootID}
	for len(queue) > 0 {
		parentID := queue[0]
		queue = queue[1:]
		children, err := s.ListSubTasks(ctx, parentID)
		if err != nil {
			return nil, err
		}
		for _, child := range children {
			result = append(result, child)
			queue = append(queue, child.ID)
		}
	}
	return result, nil
}
