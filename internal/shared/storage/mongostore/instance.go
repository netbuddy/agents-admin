package mongostore

import (
	"context"
	"time"

	"agents-admin/internal/shared/model"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ============================================================================
// AgentInstanceStore
// ============================================================================

func (s *Store) CreateAgentInstance(ctx context.Context, instance *model.Instance) error {
	return insertOne(ctx, s.col(ColAgents), instance)
}

func (s *Store) GetAgentInstance(ctx context.Context, id string) (*model.Instance, error) {
	return findOne[model.Instance](ctx, s.col(ColAgents), bson.D{{Key: "_id", Value: id}})
}

func (s *Store) ListAgentInstances(ctx context.Context) ([]*model.Instance, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	return findMany[model.Instance](ctx, s.col(ColAgents), bson.D{}, opts)
}

func (s *Store) ListAgentInstancesByNode(ctx context.Context, nodeID string) ([]*model.Instance, error) {
	filter := bson.D{{Key: "node_id", Value: nodeID}}
	return findMany[model.Instance](ctx, s.col(ColAgents), filter)
}

func (s *Store) ListPendingAgentInstances(ctx context.Context, nodeID string) ([]*model.Instance, error) {
	filter := bson.D{
		{Key: "node_id", Value: nodeID},
		{Key: "status", Value: bson.D{{Key: "$in", Value: bson.A{"pending", "creating", "stopping"}}}},
	}
	return findMany[model.Instance](ctx, s.col(ColAgents), filter)
}

func (s *Store) UpdateAgentInstance(ctx context.Context, id string, status model.InstanceStatus, containerName *string) error {
	update := bson.D{
		{Key: "status", Value: status},
		{Key: "updated_at", Value: time.Now()},
	}
	if containerName != nil {
		update = append(update, bson.E{Key: "container_name", Value: *containerName})
	}
	return updateFields(ctx, s.col(ColAgents), id, update)
}

func (s *Store) DeleteAgentInstance(ctx context.Context, id string) error {
	return deleteByID(ctx, s.col(ColAgents), id)
}
