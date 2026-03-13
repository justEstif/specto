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

// --- mock media item store ---

type mockMediaItemStore struct {
	listFn                   func(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int32) ([]core.MediaItem, error)
	createFn                 func(ctx context.Context, userID uuid.UUID, item core.MediaItem) (uuid.UUID, error)
	getFn                    func(ctx context.Context, userID, itemID uuid.UUID) (*core.MediaItem, error)
	getByExternalIDFn        func(ctx context.Context, userID uuid.UUID, platform, externalID string) (*core.MediaItem, uuid.UUID, error)
	updateEnrichmentStatusFn func(ctx context.Context, itemID uuid.UUID, status string) error
	listPendingEnrichmentFn  func(ctx context.Context, limit int32) ([]core.MediaItem, error)
}

func (m *mockMediaItemStore) List(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int32) ([]core.MediaItem, error) {
	if m.listFn != nil {
		return m.listFn(ctx, userID, from, to, limit, offset)
	}
	return nil, nil
}
func (m *mockMediaItemStore) Create(ctx context.Context, userID uuid.UUID, item core.MediaItem) (uuid.UUID, error) {
	if m.createFn != nil {
		return m.createFn(ctx, userID, item)
	}
	return uuid.New(), nil
}
func (m *mockMediaItemStore) Get(ctx context.Context, userID, itemID uuid.UUID) (*core.MediaItem, error) {
	if m.getFn != nil {
		return m.getFn(ctx, userID, itemID)
	}
	return nil, nil
}
func (m *mockMediaItemStore) GetByExternalID(ctx context.Context, userID uuid.UUID, platform, externalID string) (*core.MediaItem, uuid.UUID, error) {
	if m.getByExternalIDFn != nil {
		return m.getByExternalIDFn(ctx, userID, platform, externalID)
	}
	return nil, uuid.Nil, nil
}
func (m *mockMediaItemStore) UpdateEnrichmentStatus(ctx context.Context, itemID uuid.UUID, status string) error {
	if m.updateEnrichmentStatusFn != nil {
		return m.updateEnrichmentStatusFn(ctx, itemID, status)
	}
	return nil
}
func (m *mockMediaItemStore) ListPendingEnrichment(ctx context.Context, limit int32) ([]core.MediaItem, error) {
	if m.listPendingEnrichmentFn != nil {
		return m.listPendingEnrichmentFn(ctx, limit)
	}
	return nil, nil
}

func TestTimeline(t *testing.T) {
	duration := 3 * time.Minute
	items := []core.MediaItem{
		{
			Platform:   "spotify",
			Type:       core.MediaMusic,
			Title:      "Breathe",
			Creator:    "Pink Floyd",
			ConsumedAt: time.Now().Add(-1 * time.Hour),
			Duration:   &duration,
			ExternalID: "spotify:track:abc",
			URL:        "https://open.spotify.com/track/abc",
		},
	}

	mediaStore := &mockMediaItemStore{
		listFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time, _, _ int32) ([]core.MediaItem, error) {
			return items, nil
		},
	}

	application := &app.App{MediaItems: mediaStore}
	h := handlers.New(application)

	userID := uuid.New()
	req := authenticatedRequest("GET", "/api/v1/timeline?limit=10&offset=0", userID)
	w := httptest.NewRecorder()

	h.Timeline(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	data := resp["data"].([]any)
	if len(data) != 1 {
		t.Fatalf("expected 1 item, got %d", len(data))
	}

	item := data[0].(map[string]any)
	if item["title"] != "Breathe" {
		t.Errorf("expected title Breathe, got %s", item["title"])
	}
	if item["platform"] != "spotify" {
		t.Errorf("expected platform spotify, got %s", item["platform"])
	}
	if int(item["duration_seconds"].(float64)) != 180 {
		t.Errorf("expected duration_seconds 180, got %v", item["duration_seconds"])
	}

	meta := resp["meta"].(map[string]any)
	if int(meta["limit"].(float64)) != 10 {
		t.Errorf("expected limit 10, got %v", meta["limit"])
	}
}

func TestTimelineUnauthenticated(t *testing.T) {
	application := &app.App{MediaItems: &mockMediaItemStore{}}
	h := handlers.New(application)
	req := httptest.NewRequest("GET", "/api/v1/timeline", nil)
	w := httptest.NewRecorder()

	h.Timeline(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestTimelineInvalidFromDate(t *testing.T) {
	application := &app.App{MediaItems: &mockMediaItemStore{}}
	h := handlers.New(application)

	userID := uuid.New()
	req := authenticatedRequest("GET", "/api/v1/timeline?from=not-a-date", userID)
	w := httptest.NewRecorder()

	h.Timeline(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTimelineDefaultPagination(t *testing.T) {
	var capturedLimit, capturedOffset int32
	mediaStore := &mockMediaItemStore{
		listFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time, limit, offset int32) ([]core.MediaItem, error) {
			capturedLimit = limit
			capturedOffset = offset
			return nil, nil
		},
	}

	application := &app.App{MediaItems: mediaStore}
	h := handlers.New(application)

	userID := uuid.New()
	req := authenticatedRequest("GET", "/api/v1/timeline", userID)
	w := httptest.NewRecorder()

	h.Timeline(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if capturedLimit != 50 {
		t.Errorf("expected default limit 50, got %d", capturedLimit)
	}
	if capturedOffset != 0 {
		t.Errorf("expected default offset 0, got %d", capturedOffset)
	}
}

func TestTimelineLimitCapped(t *testing.T) {
	var capturedLimit int32
	mediaStore := &mockMediaItemStore{
		listFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time, limit, _ int32) ([]core.MediaItem, error) {
			capturedLimit = limit
			return nil, nil
		},
	}

	application := &app.App{MediaItems: mediaStore}
	h := handlers.New(application)

	userID := uuid.New()
	req := authenticatedRequest("GET", "/api/v1/timeline?limit=999", userID)
	w := httptest.NewRecorder()

	h.Timeline(w, req)

	if capturedLimit != 100 {
		t.Errorf("expected limit capped at 100, got %d", capturedLimit)
	}
}
