package mongostore

import (
	"context"
	"time"

	"agents-admin/internal/shared/model"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ============================================================================
// HITLStore
// ============================================================================

func (s *Store) CreateApprovalRequest(ctx context.Context, req *model.ApprovalRequest) error {
	return insertOne(ctx, s.col(ColApprovalRequests), req)
}

func (s *Store) GetApprovalRequest(ctx context.Context, id string) (*model.ApprovalRequest, error) {
	return findOne[model.ApprovalRequest](ctx, s.col(ColApprovalRequests), bson.D{{Key: "_id", Value: id}})
}

func (s *Store) ListApprovalRequests(ctx context.Context, runID string, status string) ([]*model.ApprovalRequest, error) {
	filter := bson.D{}
	if runID != "" {
		filter = append(filter, bson.E{Key: "run_id", Value: runID})
	}
	if status != "" {
		filter = append(filter, bson.E{Key: "status", Value: status})
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	return findMany[model.ApprovalRequest](ctx, s.col(ColApprovalRequests), filter, opts)
}

func (s *Store) UpdateApprovalRequestStatus(ctx context.Context, id string, status model.ApprovalStatus) error {
	update := bson.D{{Key: "status", Value: status}}
	if status == model.ApprovalStatusApproved || status == model.ApprovalStatusRejected {
		now := time.Now()
		update = append(update, bson.E{Key: "resolved_at", Value: now})
	}
	return updateFields(ctx, s.col(ColApprovalRequests), id, update)
}

func (s *Store) CreateApprovalDecision(ctx context.Context, decision *model.ApprovalDecision) error {
	return insertOne(ctx, s.col(ColApprovalDecisions), decision)
}

func (s *Store) CreateFeedback(ctx context.Context, feedback *model.HumanFeedback) error {
	return insertOne(ctx, s.col(ColFeedbacks), feedback)
}

func (s *Store) ListFeedbacks(ctx context.Context, runID string) ([]*model.HumanFeedback, error) {
	filter := bson.D{{Key: "run_id", Value: runID}}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	return findMany[model.HumanFeedback](ctx, s.col(ColFeedbacks), filter, opts)
}

func (s *Store) MarkFeedbackProcessed(ctx context.Context, id string) error {
	now := time.Now()
	return updateFields(ctx, s.col(ColFeedbacks), id, bson.D{
		{Key: "processed", Value: true},
		{Key: "processed_at", Value: now},
	})
}

func (s *Store) CreateIntervention(ctx context.Context, intervention *model.Intervention) error {
	return insertOne(ctx, s.col(ColInterventions), intervention)
}

func (s *Store) ListInterventions(ctx context.Context, runID string) ([]*model.Intervention, error) {
	filter := bson.D{{Key: "run_id", Value: runID}}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	return findMany[model.Intervention](ctx, s.col(ColInterventions), filter, opts)
}

func (s *Store) UpdateInterventionExecuted(ctx context.Context, id string) error {
	now := time.Now()
	return updateFields(ctx, s.col(ColInterventions), id, bson.D{
		{Key: "executed", Value: true},
		{Key: "executed_at", Value: now},
	})
}

func (s *Store) CreateConfirmation(ctx context.Context, confirmation *model.Confirmation) error {
	return insertOne(ctx, s.col(ColConfirmations), confirmation)
}

func (s *Store) GetConfirmation(ctx context.Context, id string) (*model.Confirmation, error) {
	return findOne[model.Confirmation](ctx, s.col(ColConfirmations), bson.D{{Key: "_id", Value: id}})
}

func (s *Store) ListConfirmations(ctx context.Context, runID string, status string) ([]*model.Confirmation, error) {
	filter := bson.D{}
	if runID != "" {
		filter = append(filter, bson.E{Key: "run_id", Value: runID})
	}
	if status != "" {
		filter = append(filter, bson.E{Key: "status", Value: status})
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	return findMany[model.Confirmation](ctx, s.col(ColConfirmations), filter, opts)
}

func (s *Store) UpdateConfirmationStatus(ctx context.Context, id string, status model.ConfirmStatus, selectedOption *string) error {
	update := bson.D{
		{Key: "status", Value: status},
		{Key: "resolved_at", Value: time.Now()},
	}
	if selectedOption != nil {
		update = append(update, bson.E{Key: "selected_option", Value: *selectedOption})
	}
	return updateFields(ctx, s.col(ColConfirmations), id, update)
}
