package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/justestif/specto/internal/core"
	"github.com/justestif/specto/internal/database"
)

// PgSyncLogStore implements SyncLogStore using sqlc-generated queries.
type PgSyncLogStore struct {
	q Querier
}

// NewSyncLogStore creates a new SyncLogStore backed by PostgreSQL.
func NewSyncLogStore(q Querier) *PgSyncLogStore {
	return &PgSyncLogStore{q: q}
}

var _ core.SyncLogStore = (*PgSyncLogStore)(nil)

func (s *PgSyncLogStore) Begin(ctx context.Context, userID uuid.UUID, plugin string) (uuid.UUID, error) {
	row, err := s.q.CreateSyncLog(ctx, database.CreateSyncLogParams{
		UserID: uuidToPgx(userID),
		Plugin: plugin,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("creating sync log: %w", err)
	}

	return pgxToUUID(row.ID), nil
}

func (s *PgSyncLogStore) Complete(ctx context.Context, logID uuid.UUID, result core.SyncLogResult) error {
	_, err := s.q.CompleteSyncLog(ctx, database.CompleteSyncLogParams{
		ID:           uuidToPgx(logID),
		ItemsAdded:   int4(result.ItemsAdded),
		ItemsSkipped: int4(result.ItemsSkipped),
		ItemsUpdated: int4(result.ItemsUpdated),
		Status:       "completed",
		ErrorCode:    textPtr(result.ErrorCode),
		ErrorMessage: textPtr(result.ErrorMessage),
		DurationMs:   int4(result.DurationMs),
	})
	if err != nil {
		return fmt.Errorf("completing sync log: %w", err)
	}
	return nil
}

func (s *PgSyncLogStore) Fail(ctx context.Context, logID uuid.UUID, result core.SyncLogResult) error {
	_, err := s.q.CompleteSyncLog(ctx, database.CompleteSyncLogParams{
		ID:           uuidToPgx(logID),
		ItemsAdded:   int4(result.ItemsAdded),
		ItemsSkipped: int4(result.ItemsSkipped),
		ItemsUpdated: int4(result.ItemsUpdated),
		Status:       "failed",
		ErrorCode:    textPtr(result.ErrorCode),
		ErrorMessage: textPtr(result.ErrorMessage),
		DurationMs:   int4(result.DurationMs),
	})
	if err != nil {
		return fmt.Errorf("failing sync log: %w", err)
	}
	return nil
}

func (s *PgSyncLogStore) List(ctx context.Context, userID uuid.UUID, plugin string, limit int32) ([]core.SyncLogEntry, error) {
	rows, err := s.q.ListSyncLogs(ctx, database.ListSyncLogsParams{
		UserID: uuidToPgx(userID),
		Plugin: plugin,
		Limit:  limit,
	})
	if err != nil {
		return nil, fmt.Errorf("listing sync logs: %w", err)
	}

	entries := make([]core.SyncLogEntry, len(rows))
	for i, row := range rows {
		entries[i] = syncLogFromDB(row)
	}
	return entries, nil
}
