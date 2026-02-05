// Package repository Event 相关的存储操作
package repository

import (
	"context"

	"agents-admin/internal/shared/model"
)

// CreateEvents 批量创建事件
func (s *Store) CreateEvents(ctx context.Context, events []*model.Event) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		s.rebind(`INSERT INTO events (run_id, seq, type, timestamp, payload, raw) VALUES ($1, $2, $3, $4, $5, $6)`))
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range events {
		_, err := stmt.ExecContext(ctx, e.RunID, e.Seq, e.Type, e.Timestamp, e.Payload, e.Raw)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// CountEventsByRun 统计 Run 的事件数量
func (s *Store) CountEventsByRun(ctx context.Context, runID string) (int, error) {
	query := s.rebind(`SELECT COUNT(1) FROM events WHERE run_id = $1`)
	var cnt int
	if err := s.db.QueryRowContext(ctx, query, runID).Scan(&cnt); err != nil {
		return 0, err
	}
	return cnt, nil
}

// GetEventsByRun 获取 Run 的事件
func (s *Store) GetEventsByRun(ctx context.Context, runID string, fromSeq int, limit int) ([]*model.Event, error) {
	query := s.rebind(`SELECT id, run_id, seq, type, timestamp, payload, raw 
			  FROM events WHERE run_id = $1 AND seq > $2 ORDER BY seq ASC LIMIT $3`)
	rows, err := s.db.QueryContext(ctx, query, runID, fromSeq, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*model.Event
	for rows.Next() {
		e := &model.Event{}
		var payload *[]byte
		if err := rows.Scan(&e.ID, &e.RunID, &e.Seq, &e.Type, &e.Timestamp, &payload, &e.Raw); err != nil {
			return nil, err
		}
		if payload != nil {
			e.Payload = *payload
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
