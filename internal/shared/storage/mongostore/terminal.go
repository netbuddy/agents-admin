package mongostore

import (
	"context"
	"time"

	"agents-admin/internal/shared/model"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ============================================================================
// TerminalSessionStore
// ============================================================================

func (s *Store) CreateTerminalSession(ctx context.Context, session *model.TerminalSession) error {
	return insertOne(ctx, s.col(ColTerminalSessions), session)
}

func (s *Store) GetTerminalSession(ctx context.Context, id string) (*model.TerminalSession, error) {
	return findOne[model.TerminalSession](ctx, s.col(ColTerminalSessions), bson.D{{Key: "_id", Value: id}})
}

func (s *Store) ListTerminalSessions(ctx context.Context) ([]*model.TerminalSession, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	return findMany[model.TerminalSession](ctx, s.col(ColTerminalSessions), bson.D{}, opts)
}

func (s *Store) ListTerminalSessionsByNode(ctx context.Context, nodeID string) ([]*model.TerminalSession, error) {
	filter := bson.D{{Key: "node_id", Value: nodeID}}
	return findMany[model.TerminalSession](ctx, s.col(ColTerminalSessions), filter)
}

func (s *Store) ListPendingTerminalSessions(ctx context.Context, nodeID string) ([]*model.TerminalSession, error) {
	filter := bson.D{
		{Key: "node_id", Value: nodeID},
		{Key: "status", Value: "pending"},
	}
	return findMany[model.TerminalSession](ctx, s.col(ColTerminalSessions), filter)
}

func (s *Store) UpdateTerminalSession(ctx context.Context, id string, status model.TerminalSessionStatus, port *int, url *string) error {
	update := bson.D{{Key: "status", Value: status}}
	if port != nil {
		update = append(update, bson.E{Key: "port", Value: *port})
	}
	if url != nil {
		update = append(update, bson.E{Key: "url", Value: *url})
	}
	return updateFields(ctx, s.col(ColTerminalSessions), id, update)
}

func (s *Store) DeleteTerminalSession(ctx context.Context, id string) error {
	return deleteByID(ctx, s.col(ColTerminalSessions), id)
}

func (s *Store) CleanupExpiredTerminalSessions(ctx context.Context) (int64, error) {
	filter := bson.D{{Key: "expires_at", Value: bson.D{{Key: "$lt", Value: time.Now()}}}}
	res, err := s.col(ColTerminalSessions).DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}
	return res.DeletedCount, nil
}
