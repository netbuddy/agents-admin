package mongostore

import (
	"context"

	"agents-admin/internal/shared/model"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ============================================================================
// MCPServerStore
// ============================================================================

func (s *Store) CreateMCPServer(ctx context.Context, server *model.MCPServer) error {
	return insertOne(ctx, s.col(ColMCPServers), server)
}

func (s *Store) GetMCPServer(ctx context.Context, id string) (*model.MCPServer, error) {
	return findOne[model.MCPServer](ctx, s.col(ColMCPServers), bson.D{{Key: "_id", Value: id}})
}

func (s *Store) ListMCPServers(ctx context.Context, source string) ([]*model.MCPServer, error) {
	filter := bson.D{}
	if source != "" {
		filter = append(filter, bson.E{Key: "source", Value: source})
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	return findMany[model.MCPServer](ctx, s.col(ColMCPServers), filter, opts)
}

func (s *Store) DeleteMCPServer(ctx context.Context, id string) error {
	return deleteByID(ctx, s.col(ColMCPServers), id)
}
