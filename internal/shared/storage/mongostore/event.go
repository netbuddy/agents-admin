package mongostore

import (
	"context"

	"agents-admin/internal/shared/model"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ============================================================================
// EventStore
// ============================================================================

func (s *Store) CreateEvents(ctx context.Context, events []*model.Event) error {
	if len(events) == 0 {
		return nil
	}
	docs := make([]interface{}, len(events))
	for i, e := range events {
		docs[i] = e
	}
	_, err := s.col(ColEvents).InsertMany(ctx, docs)
	return wrapError(err)
}

func (s *Store) CountEventsByRun(ctx context.Context, runID string) (int, error) {
	count, err := s.col(ColEvents).CountDocuments(ctx, bson.D{{Key: "run_id", Value: runID}})
	return int(count), err
}

func (s *Store) GetEventsByRun(ctx context.Context, runID string, fromSeq int, limit int) ([]*model.Event, error) {
	filter := bson.D{{Key: "run_id", Value: runID}}
	if fromSeq > 0 {
		filter = append(filter, bson.E{Key: "seq", Value: bson.D{{Key: "$gte", Value: fromSeq}}})
	}
	opts := options.Find().SetSort(bson.D{{Key: "seq", Value: 1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	return findMany[model.Event](ctx, s.col(ColEvents), filter, opts)
}
