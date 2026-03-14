package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/justestif/specto/internal/app"
	"github.com/justestif/specto/internal/core"
	"github.com/justestif/specto/internal/handlers"
)

// --- mock insights store ---

type mockInsightsStore struct {
	platformBreakdownFn func(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]core.PlatformBreakdownEntry, error)
	tagDistributionFn   func(ctx context.Context, userID uuid.UUID, from, to time.Time, limit int32) ([]core.TagDistributionEntry, error)
	listMediaItemsFn    func(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int32) ([]core.MediaItem, error)
}

func (m *mockInsightsStore) PlatformBreakdown(ctx context.Context, userID uuid.UUID, from, to time.Time, _ core.InsightsFilter) ([]core.PlatformBreakdownEntry, error) {
	if m.platformBreakdownFn != nil {
		return m.platformBreakdownFn(ctx, userID, from, to)
	}
	return nil, nil
}
func (m *mockInsightsStore) TagDistribution(ctx context.Context, userID uuid.UUID, from, to time.Time, limit int32, _ core.InsightsFilter) ([]core.TagDistributionEntry, error) {
	if m.tagDistributionFn != nil {
		return m.tagDistributionFn(ctx, userID, from, to, limit)
	}
	return nil, nil
}
func (m *mockInsightsStore) ListMediaItems(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int32, _ core.InsightsFilter) ([]core.MediaItem, error) {
	if m.listMediaItemsFn != nil {
		return m.listMediaItemsFn(ctx, userID, from, to, limit, offset)
	}
	return nil, nil
}
func (m *mockInsightsStore) TagDistributionByCategory(_ context.Context, _ uuid.UUID, _, _ time.Time, _ int32, _ string, _ core.InsightsFilter) ([]core.TagDistributionEntry, error) {
	return nil, nil
}
func (m *mockInsightsStore) AttentionByType(_ context.Context, _ uuid.UUID, _, _ time.Time, _ *string) ([]core.AttentionByTypeEntry, error) {
	return nil, nil
}
func (m *mockInsightsStore) ConsumptionHeatmap(_ context.Context, _ uuid.UUID, _, _ time.Time, _ core.InsightsFilter) ([]core.HeatmapCell, error) {
	return nil, nil
}
func (m *mockInsightsStore) Crossover(_ context.Context, _ uuid.UUID, _, _ time.Time, _ int32, _ *string, _ core.InsightsFilter) ([]core.CrossoverEntry, error) {
	return nil, nil
}
func (m *mockInsightsStore) TopicTimeSeries(_ context.Context, _ uuid.UUID, _, _ time.Time, _, _ *string, _ core.InsightsFilter) ([]core.TopicTimeSeriesEntry, error) {
	return nil, nil
}
func (m *mockInsightsStore) TopicSpikes(_ context.Context, _ uuid.UUID, _, _ time.Time, _ time.Time, _ int32, _ core.InsightsFilter) ([]core.TopicSpikeEntry, error) {
	return nil, nil
}

func newInsightsTestHandler(insightsStore core.InsightsStore) *handlers.Handler {
	insights := core.NewInsightsService(insightsStore)
	application := &app.App{Insights: insights}
	return handlers.New(application)
}

func TestInsightsSummary(t *testing.T) {
	store := &mockInsightsStore{
		platformBreakdownFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]core.PlatformBreakdownEntry, error) {
			return []core.PlatformBreakdownEntry{
				{Platform: "spotify", MediaType: "music", Count: 100, TotalDurationSec: 50000},
				{Platform: "youtube", MediaType: "video", Count: 50, TotalDurationSec: 30000},
			}, nil
		},
	}

	h := newInsightsTestHandler(store)
	userID := uuid.New()
	req := authenticatedRequest("GET", "/api/v1/insights/summary", userID)
	w := httptest.NewRecorder()

	h.InsightsSummary(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	data := resp["data"].(map[string]any)
	if int(data["total_items"].(float64)) != 150 {
		t.Errorf("expected total_items 150, got %v", data["total_items"])
	}
	if data["top_platform"] != "spotify" {
		t.Errorf("expected top_platform spotify, got %s", data["top_platform"])
	}
}

func TestInsightsSummaryUnauthenticated(t *testing.T) {
	h := newInsightsTestHandler(&mockInsightsStore{})
	req := httptest.NewRequest("GET", "/api/v1/insights/summary", nil)
	w := httptest.NewRecorder()

	h.InsightsSummary(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestInsightsPlatformBreakdown(t *testing.T) {
	store := &mockInsightsStore{
		platformBreakdownFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]core.PlatformBreakdownEntry, error) {
			return []core.PlatformBreakdownEntry{
				{Platform: "spotify", MediaType: "music", Count: 100, TotalDurationSec: 50000},
			}, nil
		},
	}

	h := newInsightsTestHandler(store)
	userID := uuid.New()
	req := authenticatedRequest("GET", "/api/v1/insights/platform-breakdown", userID)
	w := httptest.NewRecorder()

	h.InsightsPlatformBreakdown(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	data := resp["data"].([]any)
	if len(data) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(data))
	}
	entry := data[0].(map[string]any)
	if entry["platform"] != "spotify" {
		t.Errorf("expected platform spotify, got %s", entry["platform"])
	}
	if int(entry["count"].(float64)) != 100 {
		t.Errorf("expected count 100, got %v", entry["count"])
	}
}

func TestInsightsTags(t *testing.T) {
	store := &mockInsightsStore{
		tagDistributionFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time, _ int32) ([]core.TagDistributionEntry, error) {
			return []core.TagDistributionEntry{
				{Name: "rock", Category: "genre", Count: 184},
				{Name: "science", Category: "topic", Count: 91},
			}, nil
		},
	}

	h := newInsightsTestHandler(store)
	userID := uuid.New()
	req := authenticatedRequest("GET", "/api/v1/insights/tags?limit=10", userID)
	w := httptest.NewRecorder()

	h.InsightsTags(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	data := resp["data"].([]any)
	if len(data) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(data))
	}
}

func TestInsightsTimeline(t *testing.T) {
	duration := 3 * time.Minute
	store := &mockInsightsStore{
		listMediaItemsFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time, _, _ int32) ([]core.MediaItem, error) {
			return []core.MediaItem{
				{
					Platform:   "spotify",
					Type:       core.MediaMusic,
					Title:      "Song",
					ConsumedAt: time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC),
					Duration:   &duration,
				},
			}, nil
		},
	}

	h := newInsightsTestHandler(store)
	userID := uuid.New()
	from := "2026-03-10T00:00:00Z"
	to := "2026-03-10T23:59:59Z"
	req := authenticatedRequest("GET", "/api/v1/insights/timeline?bucket=day&from="+from+"&to="+to, userID)
	w := httptest.NewRecorder()

	h.InsightsTimeline(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	data := resp["data"].([]any)
	if len(data) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(data))
	}
	bucket := data[0].(map[string]any)
	if int(bucket["count"].(float64)) != 1 {
		t.Errorf("expected count 1, got %v", bucket["count"])
	}
}

func TestInsightsTimelineInvalidBucket(t *testing.T) {
	store := &mockInsightsStore{
		listMediaItemsFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time, _, _ int32) ([]core.MediaItem, error) {
			return nil, nil
		},
	}

	h := newInsightsTestHandler(store)
	userID := uuid.New()
	req := authenticatedRequest("GET", "/api/v1/insights/timeline?bucket=century&from=2026-03-10T00:00:00Z&to=2026-03-10T23:59:59Z", userID)
	w := httptest.NewRecorder()

	h.InsightsTimeline(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
