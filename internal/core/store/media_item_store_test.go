package store

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/justestif/specto/internal/core"
	"github.com/justestif/specto/internal/database"
)

func TestMediaItemStore_Create(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	dur := 3 * time.Minute

	mock := &mockQuerier{
		createMediaItemFn: func(_ context.Context, arg database.CreateMediaItemParams) (database.MediaItem, error) {
			// Verify conversion
			if arg.Platform != "spotify" {
				t.Errorf("Platform: want 'spotify', got %q", arg.Platform)
			}
			if arg.Type != "music" {
				t.Errorf("Type: want 'music', got %q", arg.Type)
			}
			if arg.Title != "Test Song" {
				t.Errorf("Title: want 'Test Song', got %q", arg.Title)
			}
			if !arg.Creator.Valid || arg.Creator.String != "Test Artist" {
				t.Errorf("Creator: want 'Test Artist', got %+v", arg.Creator)
			}
			if arg.ExternalID != "ext-123" {
				t.Errorf("ExternalID: want 'ext-123', got %q", arg.ExternalID)
			}
			if !arg.Duration.Valid {
				t.Error("Duration: expected valid interval")
			}
			if arg.RawMetadata == nil {
				t.Error("RawMetadata: expected non-nil")
			}

			return database.MediaItem{ID: uuidToPgx(itemID)}, nil
		},
	}

	store := NewMediaItemStore(mock)
	id, err := store.Create(context.Background(), userID, core.MediaItem{
		Platform:    "spotify",
		Type:        core.MediaMusic,
		Title:       "Test Song",
		Creator:     "Test Artist",
		ConsumedAt:  time.Now().UTC(),
		Duration:    &dur,
		ExternalID:  "ext-123",
		RawMetadata: map[string]any{"key": "value"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != itemID {
		t.Fatalf("expected ID %v, got %v", itemID, id)
	}
}

func TestMediaItemStore_Create_NullOptionals(t *testing.T) {
	mock := &mockQuerier{
		createMediaItemFn: func(_ context.Context, arg database.CreateMediaItemParams) (database.MediaItem, error) {
			if arg.Creator.Valid {
				t.Error("Creator: expected invalid for empty string")
			}
			if arg.Url.Valid {
				t.Error("URL: expected invalid for empty string")
			}
			if arg.Duration.Valid {
				t.Error("Duration: expected invalid for nil")
			}
			if arg.RawMetadata != nil {
				t.Errorf("RawMetadata: expected nil, got %v", arg.RawMetadata)
			}
			return database.MediaItem{ID: uuidToPgx(uuid.New())}, nil
		},
	}

	store := NewMediaItemStore(mock)
	_, err := store.Create(context.Background(), uuid.New(), core.MediaItem{
		Platform:   "netflix",
		Type:       core.MediaVideo,
		Title:      "Test Movie",
		ExternalID: "movie-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMediaItemStore_Get(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	mock := &mockQuerier{
		getMediaItemByIDFn: func(_ context.Context, arg database.GetMediaItemByIDParams) (database.MediaItem, error) {
			if pgxToUUID(arg.ID) != itemID {
				t.Errorf("ID: want %v, got %v", itemID, pgxToUUID(arg.ID))
			}
			if pgxToUUID(arg.UserID) != userID {
				t.Errorf("UserID: want %v, got %v", userID, pgxToUUID(arg.UserID))
			}
			return database.MediaItem{
				ID:         uuidToPgx(itemID),
				UserID:     uuidToPgx(userID),
				Platform:   "youtube",
				Type:       "video",
				Title:      "Test Video",
				Creator:    pgtype.Text{String: "Test Channel", Valid: true},
				ConsumedAt: pgtype.Timestamptz{Time: now, Valid: true},
				ExternalID: "vid-1",
			}, nil
		},
	}

	store := NewMediaItemStore(mock)
	item, err := store.Get(context.Background(), userID, itemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Platform != "youtube" {
		t.Errorf("Platform: want 'youtube', got %q", item.Platform)
	}
	if item.Title != "Test Video" {
		t.Errorf("Title: want 'Test Video', got %q", item.Title)
	}
}

func TestMediaItemStore_List(t *testing.T) {
	userID := uuid.New()
	from := time.Now().UTC().Add(-24 * time.Hour)
	to := time.Now().UTC()

	mock := &mockQuerier{
		listMediaItemsFn: func(_ context.Context, arg database.ListMediaItemsParams) ([]database.MediaItem, error) {
			if arg.Limit != 50 {
				t.Errorf("Limit: want 50, got %d", arg.Limit)
			}
			if arg.Offset != 10 {
				t.Errorf("Offset: want 10, got %d", arg.Offset)
			}
			return []database.MediaItem{
				{Platform: "spotify", Type: "music", Title: "Song 1", ExternalID: "s1"},
				{Platform: "spotify", Type: "music", Title: "Song 2", ExternalID: "s2"},
			}, nil
		},
	}

	store := NewMediaItemStore(mock)
	items, err := store.List(context.Background(), userID, from, to, 50, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Title != "Song 1" {
		t.Errorf("first item title: want 'Song 1', got %q", items[0].Title)
	}
}

func TestMediaItemStore_UpdateEnrichmentStatus(t *testing.T) {
	itemID := uuid.New()

	mock := &mockQuerier{
		updateEnrichmentStatusFn: func(_ context.Context, arg database.UpdateEnrichmentStatusParams) error {
			if pgxToUUID(arg.ID) != itemID {
				t.Errorf("ID: want %v, got %v", itemID, pgxToUUID(arg.ID))
			}
			if arg.EnrichmentStatus != "enriched" {
				t.Errorf("Status: want 'enriched', got %q", arg.EnrichmentStatus)
			}
			return nil
		},
	}

	store := NewMediaItemStore(mock)
	err := store.UpdateEnrichmentStatus(context.Background(), itemID, "enriched")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMediaItemStore_ListPendingEnrichment(t *testing.T) {
	mock := &mockQuerier{
		listPendingEnrichmentFn: func(_ context.Context, limit int32) ([]database.MediaItem, error) {
			if limit != 20 {
				t.Errorf("Limit: want 20, got %d", limit)
			}
			return []database.MediaItem{
				{Platform: "spotify", Type: "music", Title: "Pending 1", ExternalID: "p1", EnrichmentStatus: "pending"},
			}, nil
		},
	}

	store := NewMediaItemStore(mock)
	items, err := store.ListPendingEnrichment(context.Background(), 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

func TestMediaItemStore_Create_MetadataMarshal(t *testing.T) {
	var capturedMeta []byte

	mock := &mockQuerier{
		createMediaItemFn: func(_ context.Context, arg database.CreateMediaItemParams) (database.MediaItem, error) {
			capturedMeta = arg.RawMetadata
			return database.MediaItem{ID: uuidToPgx(uuid.New())}, nil
		},
	}

	store := NewMediaItemStore(mock)
	meta := map[string]any{"artist_id": "abc", "popularity": float64(85)}
	_, err := store.Create(context.Background(), uuid.New(), core.MediaItem{
		Platform:    "spotify",
		Type:        core.MediaMusic,
		Title:       "Song",
		ExternalID:  "s1",
		RawMetadata: meta,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(capturedMeta, &decoded); err != nil {
		t.Fatalf("failed to unmarshal captured metadata: %v", err)
	}
	if decoded["artist_id"] != "abc" {
		t.Errorf("artist_id: want 'abc', got %v", decoded["artist_id"])
	}
}
