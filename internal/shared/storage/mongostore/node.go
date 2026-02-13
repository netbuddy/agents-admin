package mongostore

import (
	"context"
	"time"

	"agents-admin/internal/shared/model"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ============================================================================
// NodeStore
// ============================================================================

func (s *Store) UpsertNode(ctx context.Context, node *model.Node) error {
	filter := bson.D{{Key: "_id", Value: node.ID}}
	update := bson.D{{Key: "$set", Value: node}}
	opts := options.UpdateOne().SetUpsert(true)
	_, err := s.col(ColNodes).UpdateOne(ctx, filter, update, opts)
	return wrapError(err)
}

func (s *Store) UpsertNodeHeartbeat(ctx context.Context, node *model.Node) error {
	filter := bson.D{{Key: "_id", Value: node.ID}}

	// 心跳不覆盖管理员设置的 status，仅更新心跳时间和容量等
	setOnInsert := bson.D{
		{Key: "_id", Value: node.ID},
		{Key: "status", Value: node.Status},
		{Key: "created_at", Value: node.CreatedAt},
	}
	set := bson.D{
		{Key: "last_heartbeat", Value: node.LastHeartbeat},
		{Key: "labels", Value: node.Labels},
		{Key: "capacity", Value: node.Capacity},
		{Key: "hostname", Value: node.Hostname},
		{Key: "ips", Value: node.IPs},
		{Key: "updated_at", Value: time.Now()},
	}

	// 仅对非行政状态的节点更新 status
	// 使用 $set + $setOnInsert 模式
	update := bson.D{
		{Key: "$set", Value: set},
		{Key: "$setOnInsert", Value: setOnInsert},
	}
	opts := options.UpdateOne().SetUpsert(true)
	_, err := s.col(ColNodes).UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return wrapError(err)
	}

	// 对非行政状态的节点，单独更新 status
	if !node.IsAdminStatus() {
		statusFilter := bson.D{
			{Key: "_id", Value: node.ID},
			{Key: "status", Value: bson.D{{Key: "$nin", Value: bson.A{
				"draining", "maintenance", "terminated", "starting", "unknown", "unhealthy",
			}}}},
		}
		statusUpdate := bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: node.Status}}}}
		_, _ = s.col(ColNodes).UpdateOne(ctx, statusFilter, statusUpdate)
	}

	return nil
}

func (s *Store) DeactivateStaleNodes(ctx context.Context, activeNodeID string, hostname string) error {
	if hostname == "" {
		return nil
	}
	filter := bson.D{
		{Key: "_id", Value: bson.D{{Key: "$ne", Value: activeNodeID}}},
		{Key: "hostname", Value: hostname},
		{Key: "status", Value: bson.D{{Key: "$nin", Value: bson.A{
			"offline", "terminated",
		}}}},
	}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "status", Value: "offline"},
		{Key: "updated_at", Value: time.Now()},
	}}}
	_, err := s.col(ColNodes).UpdateMany(ctx, filter, update)
	return wrapError(err)
}

func (s *Store) GetNode(ctx context.Context, id string) (*model.Node, error) {
	return findOne[model.Node](ctx, s.col(ColNodes), bson.D{{Key: "_id", Value: id}})
}

func (s *Store) ListAllNodes(ctx context.Context) ([]*model.Node, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	return findMany[model.Node](ctx, s.col(ColNodes), bson.D{}, opts)
}

func (s *Store) ListOnlineNodes(ctx context.Context) ([]*model.Node, error) {
	filter := bson.D{{Key: "status", Value: "online"}}
	return findMany[model.Node](ctx, s.col(ColNodes), filter)
}

func (s *Store) DeleteNode(ctx context.Context, id string) error {
	return deleteByID(ctx, s.col(ColNodes), id)
}

func (s *Store) CreateNodeProvision(ctx context.Context, p *model.NodeProvision) error {
	return insertOne(ctx, s.col(ColNodeProvisions), p)
}

func (s *Store) UpdateNodeProvision(ctx context.Context, p *model.NodeProvision) error {
	return updateFields(ctx, s.col(ColNodeProvisions), p.ID, bson.D{
		{Key: "status", Value: p.Status},
		{Key: "error_message", Value: p.ErrorMessage},
		{Key: "version", Value: p.Version},
		{Key: "updated_at", Value: time.Now()},
	})
}

func (s *Store) GetNodeProvision(ctx context.Context, id string) (*model.NodeProvision, error) {
	return findOne[model.NodeProvision](ctx, s.col(ColNodeProvisions), bson.D{{Key: "_id", Value: id}})
}

func (s *Store) ListNodeProvisions(ctx context.Context) ([]*model.NodeProvision, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	return findMany[model.NodeProvision](ctx, s.col(ColNodeProvisions), bson.D{}, opts)
}
