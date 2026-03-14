package core

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestWorker_TickProcessesPendingItems(t *testing.T) {
	itemID := uuid.New()
	userID := uuid.New()

	var claimedLimit int32
	var statusUpdates []string
	var tagsPersisted []string

	media := &mockMediaItemStore{
		listPendingEnrichmentFn: func(_ context.Context, limit int32) ([]MediaItem, error) {
			claimedLimit = limit
			return nil, nil
		},
	}

	// Override ClaimPendingItems
	claimFn := func(_ context.Context, limit int32, _ int32) ([]EnrichmentItem, error) {
		claimedLimit = limit
		return []EnrichmentItem{
			{
				ID:     itemID,
				UserID: userID,
				Item: MediaItem{
					Platform:   "spotify",
					Type:       MediaMusic,
					Title:      "Test Song",
					ExternalID: "ext-1",
				},
				Retries: 0,
			},
		}, nil
	}
	media.createFn = nil

	// Build a custom mock that tracks status updates
	mockMedia := &workerTestMediaStore{
		mockMediaItemStore: media,
		claimFn:            claimFn,
		updateStatusFn: func(_ context.Context, id uuid.UUID, status string) error {
			statusUpdates = append(statusUpdates, status)
			return nil
		},
		updateStatusRetriesFn: func(_ context.Context, id uuid.UUID, status string, retries int32) error {
			statusUpdates = append(statusUpdates, status)
			return nil
		},
	}

	tags := &mockTagStore{
		getOrCreateFn: func(_ context.Context, tag string) (uuid.UUID, error) {
			tagsPersisted = append(tagsPersisted, tag)
			return uuid.New(), nil
		},
	}

	// Coordinator that adds a tag
	provider := &mockEnrichmentProvider{
		name: "test",
		enrichFn: func(_ context.Context, items []MediaItem) ([]MediaItem, error) {
			for i := range items {
				items[i].Tags = []string{"rock"}
			}
			return items, nil
		},
	}
	coordinator := NewEnrichmentCoordinator([]EnrichmentProvider{provider}, nil, discardLogger())

	worker := NewEnrichmentWorker(coordinator, mockMedia, tags, discardLogger(), EnrichmentWorkerConfig{
		BatchSize:    10,
		MaxRetries:   3,
		PollInterval: time.Millisecond,
	})

	// Run one tick
	ctx := context.Background()
	worker.tick(ctx)

	// Verify items were claimed
	if claimedLimit != 10 {
		t.Errorf("expected claim limit 10, got %d", claimedLimit)
	}

	// Verify status transitions: enriching -> enriched
	if len(statusUpdates) < 2 {
		t.Fatalf("expected at least 2 status updates, got %d: %v", len(statusUpdates), statusUpdates)
	}
	if statusUpdates[0] != "enriching" {
		t.Errorf("expected first status 'enriching', got %q", statusUpdates[0])
	}
	if statusUpdates[len(statusUpdates)-1] != "enriched" {
		t.Errorf("expected last status 'enriched', got %q", statusUpdates[len(statusUpdates)-1])
	}

	// Verify tags were persisted
	if len(tagsPersisted) != 1 || tagsPersisted[0] != "rock" {
		t.Errorf("expected ['rock'] persisted, got %v", tagsPersisted)
	}
}

func TestWorker_EmptyBatch_NoOp(t *testing.T) {
	mockMedia := &workerTestMediaStore{
		mockMediaItemStore: &mockMediaItemStore{},
		claimFn: func(_ context.Context, _ int32, _ int32) ([]EnrichmentItem, error) {
			return nil, nil
		},
	}

	coordinator := NewEnrichmentCoordinator(nil, nil, discardLogger())
	worker := NewEnrichmentWorker(coordinator, mockMedia, &mockTagStore{}, discardLogger(), EnrichmentWorkerConfig{
		PollInterval: time.Millisecond,
	})

	// Should not panic or error
	worker.tick(context.Background())
}

func TestWorker_GracefulShutdown(t *testing.T) {
	mockMedia := &workerTestMediaStore{
		mockMediaItemStore: &mockMediaItemStore{},
		claimFn: func(_ context.Context, _ int32, _ int32) ([]EnrichmentItem, error) {
			return nil, nil
		},
	}

	coordinator := NewEnrichmentCoordinator(nil, nil, discardLogger())
	worker := NewEnrichmentWorker(coordinator, mockMedia, &mockTagStore{}, discardLogger(), EnrichmentWorkerConfig{
		PollInterval: 10 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		worker.Start(ctx)
		close(done)
	}()

	// Let it run a few ticks
	time.Sleep(30 * time.Millisecond)

	// Cancel and verify it stops
	cancel()
	select {
	case <-done:
		// OK
	case <-time.After(time.Second):
		t.Fatal("worker did not stop within timeout")
	}
}

func TestWorkerConfig_Defaults(t *testing.T) {
	coordinator := NewEnrichmentCoordinator(nil, nil, nil)
	worker := NewEnrichmentWorker(coordinator, &mockMediaItemStore{}, &mockTagStore{}, nil, EnrichmentWorkerConfig{})

	if worker.batchSize != DefaultBatchSize {
		t.Errorf("expected batch size %d, got %d", DefaultBatchSize, worker.batchSize)
	}
	if worker.maxRetries != DefaultMaxRetries {
		t.Errorf("expected max retries %d, got %d", DefaultMaxRetries, worker.maxRetries)
	}
	if worker.interval != DefaultPollInterval {
		t.Errorf("expected poll interval %v, got %v", DefaultPollInterval, worker.interval)
	}
}

// workerTestMediaStore wraps mockMediaItemStore but overrides claim/update
// methods needed by the worker.
type workerTestMediaStore struct {
	*mockMediaItemStore
	claimFn               func(ctx context.Context, limit int32, maxRetries int32) ([]EnrichmentItem, error)
	updateStatusFn        func(ctx context.Context, itemID uuid.UUID, status string) error
	updateStatusRetriesFn func(ctx context.Context, itemID uuid.UUID, status string, retries int32) error
}

func (m *workerTestMediaStore) ClaimPendingItems(ctx context.Context, limit int32, maxRetries int32) ([]EnrichmentItem, error) {
	if m.claimFn != nil {
		return m.claimFn(ctx, limit, maxRetries)
	}
	return nil, nil
}

func (m *workerTestMediaStore) UpdateEnrichmentStatus(ctx context.Context, itemID uuid.UUID, status string) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, itemID, status)
	}
	return nil
}

func (m *workerTestMediaStore) UpdateEnrichmentStatusWithRetries(ctx context.Context, itemID uuid.UUID, status string, retries int32) error {
	if m.updateStatusRetriesFn != nil {
		return m.updateStatusRetriesFn(ctx, itemID, status, retries)
	}
	return nil
}
