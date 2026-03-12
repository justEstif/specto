package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/justestif/specto/internal/database"
)

func TestPgInsightsStore_PlatformBreakdown(t *testing.T) {
	mq := &mockQuerier{
		platformBreakdownFn: func(_ context.Context, arg database.PlatformBreakdownParams) ([]database.PlatformBreakdownRow, error) {
			// Verify params are converted correctly
			if !arg.UserID.Valid {
				t.Error("expected valid UserID")
			}
			if !arg.ConsumedAt.Valid {
				t.Error("expected valid ConsumedAt (from)")
			}
			return []database.PlatformBreakdownRow{
				{Platform: "spotify", Type: "music", Count: 100, TotalDurationSec: 36000},
				{Platform: "youtube", Type: "video", Count: 50, TotalDurationSec: 18000},
			}, nil
		},
	}

	store := NewInsightsStore(mq)
	userID := uuid.New()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)

	entries, err := store.PlatformBreakdown(context.Background(), userID, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}
	if entries[0].Platform != "spotify" {
		t.Errorf("entries[0].Platform = %q, want %q", entries[0].Platform, "spotify")
	}
	if entries[0].MediaType != "music" {
		t.Errorf("entries[0].MediaType = %q, want %q", entries[0].MediaType, "music")
	}
	if entries[0].Count != 100 {
		t.Errorf("entries[0].Count = %d, want 100", entries[0].Count)
	}
	if entries[0].TotalDurationSec != 36000 {
		t.Errorf("entries[0].TotalDurationSec = %d, want 36000", entries[0].TotalDurationSec)
	}
}

func TestPgInsightsStore_PlatformBreakdown_Empty(t *testing.T) {
	mq := &mockQuerier{
		platformBreakdownFn: func(_ context.Context, _ database.PlatformBreakdownParams) ([]database.PlatformBreakdownRow, error) {
			return []database.PlatformBreakdownRow{}, nil
		},
	}

	store := NewInsightsStore(mq)
	entries, err := store.PlatformBreakdown(context.Background(), uuid.New(), time.Now(), time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("len(entries) = %d, want 0", len(entries))
	}
}

func TestPgInsightsStore_TagDistribution(t *testing.T) {
	mq := &mockQuerier{
		tagDistributionFn: func(_ context.Context, arg database.TagDistributionParams) ([]database.TagDistributionRow, error) {
			if arg.Limit != 20 {
				t.Errorf("limit = %d, want 20", arg.Limit)
			}
			return []database.TagDistributionRow{
				{Name: "rock", Category: pgtype.Text{String: "genre", Valid: true}, Count: 45},
				{Name: "electronic", Category: pgtype.Text{String: "genre", Valid: true}, Count: 30},
				{Name: "unknown", Category: pgtype.Text{}, Count: 10}, // NULL category
			}, nil
		},
	}

	store := NewInsightsStore(mq)
	entries, err := store.TagDistribution(context.Background(), uuid.New(), time.Now(), time.Now(), 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("len(entries) = %d, want 3", len(entries))
	}
	if entries[0].Name != "rock" || entries[0].Category != "genre" {
		t.Errorf("entries[0] = {%q, %q}, want {rock, genre}", entries[0].Name, entries[0].Category)
	}
	if entries[2].Category != "" {
		t.Errorf("entries[2].Category = %q, want empty (NULL)", entries[2].Category)
	}
}

func TestPgInsightsStore_ListMediaItems(t *testing.T) {
	mq := &mockQuerier{
		listMediaItemsFn: func(_ context.Context, arg database.ListMediaItemsParams) ([]database.MediaItem, error) {
			if arg.Limit != 100 {
				t.Errorf("limit = %d, want 100", arg.Limit)
			}
			if arg.Offset != 0 {
				t.Errorf("offset = %d, want 0", arg.Offset)
			}
			return []database.MediaItem{
				{
					Platform:   "spotify",
					Type:       "music",
					Title:      "Test Song",
					ExternalID: "ext-1",
					ConsumedAt: pgtype.Timestamptz{Time: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC), Valid: true},
				},
			}, nil
		},
	}

	store := NewInsightsStore(mq)
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)

	items, err := store.ListMediaItems(context.Background(), uuid.New(), from, to, 100, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Title != "Test Song" {
		t.Errorf("items[0].Title = %q, want %q", items[0].Title, "Test Song")
	}
}
