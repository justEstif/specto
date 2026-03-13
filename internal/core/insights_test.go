package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// --- mock InsightsStore ---

type mockInsightsStore struct {
	platformBreakdownFn func(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]PlatformBreakdownEntry, error)
	tagDistributionFn   func(ctx context.Context, userID uuid.UUID, from, to time.Time, limit int32) ([]TagDistributionEntry, error)
	listMediaItemsFn    func(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int32) ([]MediaItem, error)
}

var _ InsightsStore = (*mockInsightsStore)(nil)

func (m *mockInsightsStore) PlatformBreakdown(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]PlatformBreakdownEntry, error) {
	if m.platformBreakdownFn != nil {
		return m.platformBreakdownFn(ctx, userID, from, to)
	}
	return nil, nil
}

func (m *mockInsightsStore) TagDistribution(ctx context.Context, userID uuid.UUID, from, to time.Time, limit int32) ([]TagDistributionEntry, error) {
	if m.tagDistributionFn != nil {
		return m.tagDistributionFn(ctx, userID, from, to, limit)
	}
	return nil, nil
}

func (m *mockInsightsStore) ListMediaItems(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int32) ([]MediaItem, error) {
	if m.listMediaItemsFn != nil {
		return m.listMediaItemsFn(ctx, userID, from, to, limit, offset)
	}
	return nil, nil
}

func (m *mockInsightsStore) PlatformBreakdownFiltered(ctx context.Context, userID uuid.UUID, from, to time.Time, _ InsightsFilter) ([]PlatformBreakdownEntry, error) {
	return m.PlatformBreakdown(ctx, userID, from, to)
}

func (m *mockInsightsStore) TagDistributionFiltered(ctx context.Context, userID uuid.UUID, from, to time.Time, limit int32, _ InsightsFilter) ([]TagDistributionEntry, error) {
	return m.TagDistribution(ctx, userID, from, to, limit)
}

func (m *mockInsightsStore) ListMediaItemsFiltered(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int32, _ InsightsFilter) ([]MediaItem, error) {
	return m.ListMediaItems(ctx, userID, from, to, limit, offset)
}

// --- test helpers ---

var (
	testUserID = uuid.New()
	testFrom   = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	testTo     = time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC)
)

func dur(d time.Duration) *time.Duration { return &d }

// --- GetSummary tests ---

func TestGetSummary_Basic(t *testing.T) {
	store := &mockInsightsStore{
		platformBreakdownFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]PlatformBreakdownEntry, error) {
			return []PlatformBreakdownEntry{
				{Platform: "spotify", MediaType: "music", Count: 100, TotalDurationSec: 36000},
				{Platform: "spotify", MediaType: "podcast", Count: 20, TotalDurationSec: 14400},
				{Platform: "youtube", MediaType: "video", Count: 50, TotalDurationSec: 18000},
			}, nil
		},
	}
	svc := NewInsightsService(store)
	summary, err := svc.GetSummary(context.Background(), testUserID, testFrom, testTo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.TotalItems != 170 {
		t.Errorf("TotalItems = %d, want 170", summary.TotalItems)
	}
	if summary.TotalDurationSec != 68400 {
		t.Errorf("TotalDurationSec = %d, want 68400", summary.TotalDurationSec)
	}
	if summary.TopPlatform != "spotify" {
		t.Errorf("TopPlatform = %q, want %q", summary.TopPlatform, "spotify")
	}
	if summary.TopMediaType != "music" {
		t.Errorf("TopMediaType = %q, want %q", summary.TopMediaType, "music")
	}
}

func TestGetSummary_Empty(t *testing.T) {
	store := &mockInsightsStore{
		platformBreakdownFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]PlatformBreakdownEntry, error) {
			return []PlatformBreakdownEntry{}, nil
		},
	}
	svc := NewInsightsService(store)
	summary, err := svc.GetSummary(context.Background(), testUserID, testFrom, testTo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.TotalItems != 0 {
		t.Errorf("TotalItems = %d, want 0", summary.TotalItems)
	}
	if summary.TopPlatform != "" {
		t.Errorf("TopPlatform = %q, want empty", summary.TopPlatform)
	}
}

func TestGetSummary_StoreError(t *testing.T) {
	store := &mockInsightsStore{
		platformBreakdownFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]PlatformBreakdownEntry, error) {
			return nil, errors.New("db connection failed")
		},
	}
	svc := NewInsightsService(store)
	_, err := svc.GetSummary(context.Background(), testUserID, testFrom, testTo)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetSummary_InvalidDateRange(t *testing.T) {
	svc := NewInsightsService(&mockInsightsStore{})

	// to before from
	_, err := svc.GetSummary(context.Background(), testUserID, testTo, testFrom)
	if err == nil {
		t.Fatal("expected error for reversed date range")
	}

	// zero from
	_, err = svc.GetSummary(context.Background(), testUserID, time.Time{}, testTo)
	if err == nil {
		t.Fatal("expected error for zero from")
	}

	// zero to
	_, err = svc.GetSummary(context.Background(), testUserID, testFrom, time.Time{})
	if err == nil {
		t.Fatal("expected error for zero to")
	}
}

// --- GetTimeline tests ---

func TestGetTimeline_DailyBuckets(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 3, 23, 59, 59, 0, time.UTC)

	store := &mockInsightsStore{
		listMediaItemsFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time, _, _ int32) ([]MediaItem, error) {
			return []MediaItem{
				{ConsumedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), Duration: dur(30 * time.Minute)},
				{ConsumedAt: time.Date(2026, 1, 1, 14, 0, 0, 0, time.UTC), Duration: dur(45 * time.Minute)},
				{ConsumedAt: time.Date(2026, 1, 3, 8, 0, 0, 0, time.UTC), Duration: dur(60 * time.Minute)},
			}, nil
		},
	}
	svc := NewInsightsService(store)
	timeline, err := svc.GetTimeline(context.Background(), testUserID, BucketDay, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 3 daily buckets: Jan 1, 2, 3
	if len(timeline) != 3 {
		t.Fatalf("len(timeline) = %d, want 3", len(timeline))
	}

	// Jan 1: 2 items, 4500 sec (30+45 min)
	if timeline[0].Count != 2 {
		t.Errorf("Jan 1 count = %d, want 2", timeline[0].Count)
	}
	if timeline[0].TotalDurationSec != 4500 {
		t.Errorf("Jan 1 duration = %d, want 4500", timeline[0].TotalDurationSec)
	}

	// Jan 2: 0 items (empty bucket)
	if timeline[1].Count != 0 {
		t.Errorf("Jan 2 count = %d, want 0", timeline[1].Count)
	}

	// Jan 3: 1 item, 3600 sec
	if timeline[2].Count != 1 {
		t.Errorf("Jan 3 count = %d, want 1", timeline[2].Count)
	}
	if timeline[2].TotalDurationSec != 3600 {
		t.Errorf("Jan 3 duration = %d, want 3600", timeline[2].TotalDurationSec)
	}
}

func TestGetTimeline_WeeklyBuckets(t *testing.T) {
	// 2026-01-05 is a Monday, 2026-01-18 is a Sunday
	from := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 18, 23, 59, 59, 0, time.UTC)

	store := &mockInsightsStore{
		listMediaItemsFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time, _, _ int32) ([]MediaItem, error) {
			return []MediaItem{
				{ConsumedAt: time.Date(2026, 1, 6, 10, 0, 0, 0, time.UTC)},  // week of Jan 5
				{ConsumedAt: time.Date(2026, 1, 13, 14, 0, 0, 0, time.UTC)}, // week of Jan 12
			}, nil
		},
	}
	svc := NewInsightsService(store)
	timeline, err := svc.GetTimeline(context.Background(), testUserID, BucketWeek, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 2 weekly buckets: week of Jan 5, week of Jan 12
	if len(timeline) != 2 {
		t.Fatalf("len(timeline) = %d, want 2", len(timeline))
	}

	expectedFirst := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	if !timeline[0].Bucket.Equal(expectedFirst) {
		t.Errorf("first bucket = %v, want %v", timeline[0].Bucket, expectedFirst)
	}
	if timeline[0].Count != 1 {
		t.Errorf("first bucket count = %d, want 1", timeline[0].Count)
	}
}

func TestGetTimeline_MonthlyBuckets(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC)

	store := &mockInsightsStore{
		listMediaItemsFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time, _, _ int32) ([]MediaItem, error) {
			return []MediaItem{
				{ConsumedAt: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)},
				{ConsumedAt: time.Date(2026, 3, 10, 14, 0, 0, 0, time.UTC)},
			}, nil
		},
	}
	svc := NewInsightsService(store)
	timeline, err := svc.GetTimeline(context.Background(), testUserID, BucketMonth, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 3 months: Jan, Feb, Mar
	if len(timeline) != 3 {
		t.Fatalf("len(timeline) = %d, want 3", len(timeline))
	}
	if timeline[0].Count != 1 {
		t.Errorf("Jan count = %d, want 1", timeline[0].Count)
	}
	if timeline[1].Count != 0 {
		t.Errorf("Feb count = %d, want 0", timeline[1].Count)
	}
	if timeline[2].Count != 1 {
		t.Errorf("Mar count = %d, want 1", timeline[2].Count)
	}
}

func TestGetTimeline_Empty(t *testing.T) {
	store := &mockInsightsStore{
		listMediaItemsFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time, _, _ int32) ([]MediaItem, error) {
			return []MediaItem{}, nil
		},
	}
	svc := NewInsightsService(store)
	timeline, err := svc.GetTimeline(context.Background(), testUserID, BucketDay, testFrom, testTo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Even with no items, we should get empty buckets for every day
	if len(timeline) != 31 {
		t.Errorf("len(timeline) = %d, want 31 (days in Jan)", len(timeline))
	}
	for i, e := range timeline {
		if e.Count != 0 {
			t.Errorf("bucket %d count = %d, want 0", i, e.Count)
		}
	}
}

func TestGetTimeline_InvalidBucket(t *testing.T) {
	svc := NewInsightsService(&mockInsightsStore{})
	_, err := svc.GetTimeline(context.Background(), testUserID, "quarter", testFrom, testTo)
	if err == nil {
		t.Fatal("expected error for invalid bucket")
	}
}

func TestGetTimeline_StoreError(t *testing.T) {
	store := &mockInsightsStore{
		listMediaItemsFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time, _, _ int32) ([]MediaItem, error) {
			return nil, errors.New("timeout")
		},
	}
	svc := NewInsightsService(store)
	_, err := svc.GetTimeline(context.Background(), testUserID, BucketDay, testFrom, testTo)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetTimeline_Pagination(t *testing.T) {
	// Verify the service paginates through multiple pages
	callCount := 0
	store := &mockInsightsStore{
		listMediaItemsFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time, limit, offset int32) ([]MediaItem, error) {
			callCount++
			if offset == 0 {
				// Return a full page to trigger pagination
				items := make([]MediaItem, limit)
				for i := range items {
					items[i] = MediaItem{ConsumedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)}
				}
				return items, nil
			}
			// Second page: empty
			return []MediaItem{}, nil
		},
	}
	svc := NewInsightsService(store)

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 1, 23, 59, 59, 0, time.UTC)
	_, err := svc.GetTimeline(context.Background(), testUserID, BucketDay, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount < 2 {
		t.Errorf("expected at least 2 store calls for pagination, got %d", callCount)
	}
}

// --- GetPlatformBreakdown tests ---

func TestGetPlatformBreakdown_Basic(t *testing.T) {
	expected := []PlatformBreakdownEntry{
		{Platform: "spotify", MediaType: "music", Count: 100, TotalDurationSec: 36000},
		{Platform: "youtube", MediaType: "video", Count: 50, TotalDurationSec: 18000},
	}
	store := &mockInsightsStore{
		platformBreakdownFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]PlatformBreakdownEntry, error) {
			return expected, nil
		},
	}
	svc := NewInsightsService(store)
	result, err := svc.GetPlatformBreakdown(context.Background(), testUserID, testFrom, testTo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}
	if result[0].Platform != "spotify" {
		t.Errorf("result[0].Platform = %q, want %q", result[0].Platform, "spotify")
	}
}

func TestGetPlatformBreakdown_InvalidDateRange(t *testing.T) {
	svc := NewInsightsService(&mockInsightsStore{})
	_, err := svc.GetPlatformBreakdown(context.Background(), testUserID, testTo, testFrom)
	if err == nil {
		t.Fatal("expected error for reversed date range")
	}
}

// --- GetTagDistribution tests ---

func TestGetTagDistribution_Basic(t *testing.T) {
	expected := []TagDistributionEntry{
		{Name: "rock", Category: "genre", Count: 45},
		{Name: "electronic", Category: "genre", Count: 30},
		{Name: "nostalgia", Category: "mood", Count: 20},
	}
	store := &mockInsightsStore{
		tagDistributionFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time, limit int32) ([]TagDistributionEntry, error) {
			if limit != 10 {
				t.Errorf("limit = %d, want 10", limit)
			}
			return expected, nil
		},
	}
	svc := NewInsightsService(store)
	result, err := svc.GetTagDistribution(context.Background(), testUserID, testFrom, testTo, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("len(result) = %d, want 3", len(result))
	}
	if result[0].Name != "rock" {
		t.Errorf("result[0].Name = %q, want %q", result[0].Name, "rock")
	}
}

func TestGetTagDistribution_DefaultLimit(t *testing.T) {
	var gotLimit int32
	store := &mockInsightsStore{
		tagDistributionFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time, limit int32) ([]TagDistributionEntry, error) {
			gotLimit = limit
			return nil, nil
		},
	}
	svc := NewInsightsService(store)

	// Passing 0 should default to 50
	_, err := svc.GetTagDistribution(context.Background(), testUserID, testFrom, testTo, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotLimit != 50 {
		t.Errorf("limit passed to store = %d, want 50 (default)", gotLimit)
	}
}

func TestGetTagDistribution_NegativeLimit(t *testing.T) {
	var gotLimit int32
	store := &mockInsightsStore{
		tagDistributionFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time, limit int32) ([]TagDistributionEntry, error) {
			gotLimit = limit
			return nil, nil
		},
	}
	svc := NewInsightsService(store)

	_, err := svc.GetTagDistribution(context.Background(), testUserID, testFrom, testTo, -5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotLimit != 50 {
		t.Errorf("limit passed to store = %d, want 50 (default)", gotLimit)
	}
}

func TestGetTagDistribution_InvalidDateRange(t *testing.T) {
	svc := NewInsightsService(&mockInsightsStore{})
	_, err := svc.GetTagDistribution(context.Background(), testUserID, testTo, testFrom, 10)
	if err == nil {
		t.Fatal("expected error for reversed date range")
	}
}

// --- helper function tests ---

func TestTruncateToBucket_Day(t *testing.T) {
	ts := time.Date(2026, 3, 15, 14, 30, 0, 0, time.UTC)
	got := truncateToBucket(ts, BucketDay)
	want := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("truncateToBucket(day) = %v, want %v", got, want)
	}
}

func TestTruncateToBucket_Week(t *testing.T) {
	// 2026-03-12 is a Thursday
	ts := time.Date(2026, 3, 12, 14, 30, 0, 0, time.UTC)
	got := truncateToBucket(ts, BucketWeek)
	want := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC) // Monday
	if !got.Equal(want) {
		t.Errorf("truncateToBucket(week) = %v, want %v", got, want)
	}
}

func TestTruncateToBucket_Week_Sunday(t *testing.T) {
	// 2026-03-15 is a Sunday
	ts := time.Date(2026, 3, 15, 14, 30, 0, 0, time.UTC)
	got := truncateToBucket(ts, BucketWeek)
	want := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC) // Monday
	if !got.Equal(want) {
		t.Errorf("truncateToBucket(week, sunday) = %v, want %v", got, want)
	}
}

func TestTruncateToBucket_Month(t *testing.T) {
	ts := time.Date(2026, 3, 15, 14, 30, 0, 0, time.UTC)
	got := truncateToBucket(ts, BucketMonth)
	want := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("truncateToBucket(month) = %v, want %v", got, want)
	}
}

func TestMaxKey_Basic(t *testing.T) {
	m := map[string]int64{"a": 10, "b": 20, "c": 5}
	got := maxKey(m)
	if got != "b" {
		t.Errorf("maxKey = %q, want %q", got, "b")
	}
}

func TestMaxKey_Tie(t *testing.T) {
	// On tie, should pick lexicographically smaller key for determinism
	m := map[string]int64{"b": 10, "a": 10}
	got := maxKey(m)
	if got != "a" {
		t.Errorf("maxKey (tie) = %q, want %q", got, "a")
	}
}

func TestMaxKey_Empty(t *testing.T) {
	got := maxKey(map[string]int64{})
	if got != "" {
		t.Errorf("maxKey (empty) = %q, want empty", got)
	}
}
