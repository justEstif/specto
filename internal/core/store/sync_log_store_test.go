package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/justestif/specto/internal/core"
	"github.com/justestif/specto/internal/database"
)

func TestSyncLogStore_Begin(t *testing.T) {
	userID := uuid.New()
	logID := uuid.New()

	mock := &mockQuerier{
		createSyncLogFn: func(_ context.Context, arg database.CreateSyncLogParams) (database.SyncLog, error) {
			if arg.Plugin != "spotify" {
				t.Errorf("Plugin: want 'spotify', got %q", arg.Plugin)
			}
			return database.SyncLog{
				ID:     uuidToPgx(logID),
				UserID: uuidToPgx(userID),
				Plugin: "spotify",
				Status: "running",
			}, nil
		},
	}

	store := NewSyncLogStore(mock)
	id, err := store.Begin(context.Background(), userID, "spotify")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != logID {
		t.Fatalf("expected log ID %v, got %v", logID, id)
	}
}

func TestSyncLogStore_Complete(t *testing.T) {
	logID := uuid.New()

	mock := &mockQuerier{
		completeSyncLogFn: func(_ context.Context, arg database.CompleteSyncLogParams) (database.SyncLog, error) {
			if pgxToUUID(arg.ID) != logID {
				t.Errorf("ID: want %v, got %v", logID, pgxToUUID(arg.ID))
			}
			if arg.Status != "completed" {
				t.Errorf("Status: want 'completed', got %q", arg.Status)
			}
			if int4Val(arg.ItemsAdded) != 10 {
				t.Errorf("ItemsAdded: want 10, got %d", int4Val(arg.ItemsAdded))
			}
			if int4Val(arg.ItemsSkipped) != 2 {
				t.Errorf("ItemsSkipped: want 2, got %d", int4Val(arg.ItemsSkipped))
			}
			if int4Val(arg.ItemsUpdated) != 3 {
				t.Errorf("ItemsUpdated: want 3, got %d", int4Val(arg.ItemsUpdated))
			}
			if int4Val(arg.DurationMs) != 5000 {
				t.Errorf("DurationMs: want 5000, got %d", int4Val(arg.DurationMs))
			}
			return database.SyncLog{}, nil
		},
	}

	store := NewSyncLogStore(mock)
	err := store.Complete(context.Background(), logID, core.SyncLogResult{
		ItemsAdded:   10,
		ItemsSkipped: 2,
		ItemsUpdated: 3,
		DurationMs:   5000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSyncLogStore_Fail(t *testing.T) {
	logID := uuid.New()
	errCode := "auth_expired"
	errMsg := "token expired"

	mock := &mockQuerier{
		completeSyncLogFn: func(_ context.Context, arg database.CompleteSyncLogParams) (database.SyncLog, error) {
			if arg.Status != "failed" {
				t.Errorf("Status: want 'failed', got %q", arg.Status)
			}
			if !arg.ErrorCode.Valid || arg.ErrorCode.String != errCode {
				t.Errorf("ErrorCode: want %q, got %+v", errCode, arg.ErrorCode)
			}
			if !arg.ErrorMessage.Valid || arg.ErrorMessage.String != errMsg {
				t.Errorf("ErrorMessage: want %q, got %+v", errMsg, arg.ErrorMessage)
			}
			return database.SyncLog{}, nil
		},
	}

	store := NewSyncLogStore(mock)
	err := store.Fail(context.Background(), logID, core.SyncLogResult{
		ItemsAdded:   5,
		ErrorCode:    &errCode,
		ErrorMessage: &errMsg,
		DurationMs:   2000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSyncLogStore_List(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC()

	mock := &mockQuerier{
		listSyncLogsFn: func(_ context.Context, arg database.ListSyncLogsParams) ([]database.SyncLog, error) {
			if arg.Plugin != "spotify" {
				t.Errorf("Plugin: want 'spotify', got %q", arg.Plugin)
			}
			if arg.Limit != 10 {
				t.Errorf("Limit: want 10, got %d", arg.Limit)
			}
			return []database.SyncLog{
				{
					ID:           uuidToPgx(uuid.New()),
					UserID:       uuidToPgx(userID),
					Plugin:       "spotify",
					StartedAt:    pgtype.Timestamptz{Time: now, Valid: true},
					CompletedAt:  pgtype.Timestamptz{Time: now.Add(time.Minute), Valid: true},
					ItemsAdded:   pgtype.Int4{Int32: 15, Valid: true},
					ItemsSkipped: pgtype.Int4{Int32: 0, Valid: true},
					ItemsUpdated: pgtype.Int4{Int32: 0, Valid: true},
					Status:       "completed",
					DurationMs:   pgtype.Int4{Int32: 60000, Valid: true},
				},
			}, nil
		},
	}

	store := NewSyncLogStore(mock)
	entries, err := store.List(context.Background(), userID, "spotify", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].ItemsAdded != 15 {
		t.Errorf("ItemsAdded: want 15, got %d", entries[0].ItemsAdded)
	}
	if entries[0].Status != "completed" {
		t.Errorf("Status: want 'completed', got %q", entries[0].Status)
	}
}
