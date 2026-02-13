package mongostore

import (
	"context"
	"encoding/json"
	"time"

	"agents-admin/internal/shared/model"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ============================================================================
// OperationStore
// ============================================================================

func (s *Store) CreateOperation(ctx context.Context, op *model.Operation) error {
	return insertOne(ctx, s.col(ColOperations), op)
}

func (s *Store) GetOperation(ctx context.Context, id string) (*model.Operation, error) {
	return findOne[model.Operation](ctx, s.col(ColOperations), bson.D{{Key: "_id", Value: id}})
}

func (s *Store) ListOperations(ctx context.Context, opType string, status string, limit, offset int) ([]*model.Operation, error) {
	filter := bson.D{}
	if opType != "" {
		filter = append(filter, bson.E{Key: "type", Value: opType})
	}
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
	return findMany[model.Operation](ctx, s.col(ColOperations), filter, opts)
}

func (s *Store) UpdateOperationStatus(ctx context.Context, id string, status model.OperationStatus) error {
	return updateFields(ctx, s.col(ColOperations), id, bson.D{
		{Key: "status", Value: status},
		{Key: "updated_at", Value: time.Now()},
	})
}

// ============================================================================
// ActionStore
// ============================================================================

func (s *Store) CreateAction(ctx context.Context, action *model.Action) error {
	return insertOne(ctx, s.col(ColActions), action)
}

func (s *Store) GetAction(ctx context.Context, id string) (*model.Action, error) {
	return findOne[model.Action](ctx, s.col(ColActions), bson.D{{Key: "_id", Value: id}})
}

func (s *Store) GetActionWithOperation(ctx context.Context, id string) (*model.Action, error) {
	action, err := s.GetAction(ctx, id)
	if err != nil {
		return nil, err
	}
	if action == nil {
		return nil, nil
	}
	op, err := s.GetOperation(ctx, action.OperationID)
	if err == nil {
		action.Operation = op
	}
	return action, nil
}

func (s *Store) ListActionsByOperation(ctx context.Context, operationID string) ([]*model.Action, error) {
	filter := bson.D{{Key: "operation_id", Value: operationID}}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})
	return findMany[model.Action](ctx, s.col(ColActions), filter, opts)
}

func (s *Store) ListActionsByNode(ctx context.Context, nodeID string, status string) ([]*model.Action, error) {
	// Actions don't have a direct node_id, we need to join through operations
	opFilter := bson.D{{Key: "node_id", Value: nodeID}}
	ops, err := findMany[model.Operation](ctx, s.col(ColOperations), opFilter)
	if err != nil {
		return nil, err
	}
	if len(ops) == 0 {
		return []*model.Action{}, nil
	}

	// Build operation lookup map for attaching to actions
	opMap := make(map[string]*model.Operation, len(ops))
	opIDs := make(bson.A, len(ops))
	for i, op := range ops {
		opIDs[i] = op.ID
		opMap[op.ID] = op
	}

	actionFilter := bson.D{{Key: "operation_id", Value: bson.D{{Key: "$in", Value: opIDs}}}}
	if status != "" {
		actionFilter = append(actionFilter, bson.E{Key: "status", Value: status})
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})
	actions, err := findMany[model.Action](ctx, s.col(ColActions), actionFilter, opts)
	if err != nil {
		return nil, err
	}

	// Attach Operation to each Action (equivalent to SQL JOIN)
	for _, action := range actions {
		action.Operation = opMap[action.OperationID]
	}
	return actions, nil
}

func (s *Store) UpdateActionStatus(ctx context.Context, id string, status model.ActionStatus, phase model.ActionPhase, message string, progress int, result json.RawMessage, errMsg string) error {
	update := bson.D{
		{Key: "status", Value: status},
		{Key: "phase", Value: phase},
		{Key: "message", Value: message},
		{Key: "progress", Value: progress},
		{Key: "error", Value: errMsg},
	}
	if result != nil {
		update = append(update, bson.E{Key: "result", Value: result})
	}
	if status == model.ActionStatusRunning {
		now := time.Now()
		update = append(update, bson.E{Key: "started_at", Value: now})
	}
	if status == model.ActionStatusSuccess || status == model.ActionStatusFailed {
		now := time.Now()
		update = append(update, bson.E{Key: "finished_at", Value: now})
	}
	return updateFields(ctx, s.col(ColActions), id, update)
}
