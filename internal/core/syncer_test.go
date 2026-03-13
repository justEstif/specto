package core

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
)

// --- Mock implementations ---

// mockPlugin implements SourcePlugin for testing.
// Use newMockPlugin() to get a properly initialized instance (AuthNone).
type mockPlugin struct {
	name       string
	authType   AuthType
	authConfig *OAuthConfig
	syncFn     func(ctx context.Context, creds Credentials, cursor string) SyncResult
	enrichFn   func(ctx context.Context, creds Credentials, items []MediaItem) ([]MediaItem, error)
}

func newMockPlugin(name string) *mockPlugin {
	return &mockPlugin{name: name, authType: AuthNone}
}

func (m *mockPlugin) Name() string             { return m.name }
func (m *mockPlugin) AuthType() AuthType       { return m.authType }
func (m *mockPlugin) AuthConfig() *OAuthConfig { return m.authConfig }
func (m *mockPlugin) Sync(ctx context.Context, creds Credentials, cursor string) SyncResult {
	if m.syncFn != nil {
		return m.syncFn(ctx, creds, cursor)
	}
	return SyncResult{}
}
func (m *mockPlugin) Enrich(ctx context.Context, creds Credentials, items []MediaItem) ([]MediaItem, error) {
	if m.enrichFn != nil {
		return m.enrichFn(ctx, creds, items)
	}
	return items, nil
}

// mockMediaItemStore implements MediaItemStore for testing.
type mockMediaItemStore struct {
	createFn                 func(ctx context.Context, userID uuid.UUID, item MediaItem) (uuid.UUID, error)
	getFn                    func(ctx context.Context, userID, itemID uuid.UUID) (*MediaItem, error)
	getByExternalIDFn        func(ctx context.Context, userID uuid.UUID, platform, externalID string) (*MediaItem, uuid.UUID, error)
	listFn                   func(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int32) ([]MediaItem, error)
	updateEnrichmentStatusFn func(ctx context.Context, itemID uuid.UUID, status string) error
	listPendingEnrichmentFn  func(ctx context.Context, limit int32) ([]MediaItem, error)
}

func (m *mockMediaItemStore) Create(ctx context.Context, userID uuid.UUID, item MediaItem) (uuid.UUID, error) {
	if m.createFn != nil {
		return m.createFn(ctx, userID, item)
	}
	return uuid.New(), nil
}
func (m *mockMediaItemStore) GetByExternalID(ctx context.Context, userID uuid.UUID, platform, externalID string) (*MediaItem, uuid.UUID, error) {
	if m.getByExternalIDFn != nil {
		return m.getByExternalIDFn(ctx, userID, platform, externalID)
	}
	return &MediaItem{}, uuid.New(), nil
}
func (m *mockMediaItemStore) Get(ctx context.Context, userID, itemID uuid.UUID) (*MediaItem, error) {
	if m.getFn != nil {
		return m.getFn(ctx, userID, itemID)
	}
	return nil, errors.New("not found")
}
func (m *mockMediaItemStore) List(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int32) ([]MediaItem, error) {
	if m.listFn != nil {
		return m.listFn(ctx, userID, from, to, limit, offset)
	}
	return nil, nil
}
func (m *mockMediaItemStore) UpdateEnrichmentStatus(ctx context.Context, itemID uuid.UUID, status string) error {
	if m.updateEnrichmentStatusFn != nil {
		return m.updateEnrichmentStatusFn(ctx, itemID, status)
	}
	return nil
}
func (m *mockMediaItemStore) ListFiltered(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int32, platform, mediaType, search *string) ([]MediaItem, error) {
	return m.List(ctx, userID, from, to, limit, offset)
}
func (m *mockMediaItemStore) ListPendingEnrichment(ctx context.Context, limit int32) ([]MediaItem, error) {
	if m.listPendingEnrichmentFn != nil {
		return m.listPendingEnrichmentFn(ctx, limit)
	}
	return nil, nil
}

// mockPluginStateStore implements PluginStateStore for testing.
type mockPluginStateStore struct {
	getStateFn          func(ctx context.Context, userID uuid.UUID, plugin string) (*PluginStateInfo, error)
	upsertStateFn       func(ctx context.Context, userID uuid.UUID, plugin, status string, enabled bool) (*PluginStateInfo, error)
	updateStatusFn      func(ctx context.Context, userID uuid.UUID, plugin, status string, errMsg *string) (*PluginStateInfo, error)
	updateSyncedFn      func(ctx context.Context, userID uuid.UUID, plugin string, cursor *string) (*PluginStateInfo, error)
	listStatesFn        func(ctx context.Context, userID uuid.UUID) ([]PluginStateInfo, error)
	getCredentialsFn    func(ctx context.Context, userID uuid.UUID, plugin string) (*Credentials, error)
	upsertCredentialsFn func(ctx context.Context, userID uuid.UUID, plugin string, authType AuthType, creds Credentials, expiresAt *time.Time) error
	deleteCredentialsFn func(ctx context.Context, userID uuid.UUID, plugin string) error
}

func (m *mockPluginStateStore) GetState(ctx context.Context, userID uuid.UUID, plugin string) (*PluginStateInfo, error) {
	if m.getStateFn != nil {
		return m.getStateFn(ctx, userID, plugin)
	}
	return &PluginStateInfo{Plugin: plugin, Status: "connected"}, nil
}
func (m *mockPluginStateStore) UpsertState(ctx context.Context, userID uuid.UUID, plugin, status string, enabled bool) (*PluginStateInfo, error) {
	if m.upsertStateFn != nil {
		return m.upsertStateFn(ctx, userID, plugin, status, enabled)
	}
	return &PluginStateInfo{Plugin: plugin, Status: status}, nil
}
func (m *mockPluginStateStore) UpdateStatus(ctx context.Context, userID uuid.UUID, plugin, status string, errMsg *string) (*PluginStateInfo, error) {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, userID, plugin, status, errMsg)
	}
	return &PluginStateInfo{Plugin: plugin, Status: status}, nil
}
func (m *mockPluginStateStore) UpdateSynced(ctx context.Context, userID uuid.UUID, plugin string, cursor *string) (*PluginStateInfo, error) {
	if m.updateSyncedFn != nil {
		return m.updateSyncedFn(ctx, userID, plugin, cursor)
	}
	return &PluginStateInfo{Plugin: plugin}, nil
}
func (m *mockPluginStateStore) ListStates(ctx context.Context, userID uuid.UUID) ([]PluginStateInfo, error) {
	if m.listStatesFn != nil {
		return m.listStatesFn(ctx, userID)
	}
	return nil, nil
}
func (m *mockPluginStateStore) GetCredentials(ctx context.Context, userID uuid.UUID, plugin string) (*Credentials, error) {
	if m.getCredentialsFn != nil {
		return m.getCredentialsFn(ctx, userID, plugin)
	}
	return &Credentials{AccessToken: "test-token"}, nil
}
func (m *mockPluginStateStore) UpsertCredentials(ctx context.Context, userID uuid.UUID, plugin string, authType AuthType, creds Credentials, expiresAt *time.Time) error {
	if m.upsertCredentialsFn != nil {
		return m.upsertCredentialsFn(ctx, userID, plugin, authType, creds, expiresAt)
	}
	return nil
}
func (m *mockPluginStateStore) DeleteCredentials(ctx context.Context, userID uuid.UUID, plugin string) error {
	if m.deleteCredentialsFn != nil {
		return m.deleteCredentialsFn(ctx, userID, plugin)
	}
	return nil
}

// mockSyncLogStore implements SyncLogStore for testing.
type mockSyncLogStore struct {
	beginFn    func(ctx context.Context, userID uuid.UUID, plugin string) (uuid.UUID, error)
	completeFn func(ctx context.Context, logID uuid.UUID, result SyncLogResult) error
	failFn     func(ctx context.Context, logID uuid.UUID, result SyncLogResult) error
	listFn     func(ctx context.Context, userID uuid.UUID, plugin string, limit int32) ([]SyncLogEntry, error)
}

func (m *mockSyncLogStore) Begin(ctx context.Context, userID uuid.UUID, plugin string) (uuid.UUID, error) {
	if m.beginFn != nil {
		return m.beginFn(ctx, userID, plugin)
	}
	return uuid.New(), nil
}
func (m *mockSyncLogStore) Complete(ctx context.Context, logID uuid.UUID, result SyncLogResult) error {
	if m.completeFn != nil {
		return m.completeFn(ctx, logID, result)
	}
	return nil
}
func (m *mockSyncLogStore) Fail(ctx context.Context, logID uuid.UUID, result SyncLogResult) error {
	if m.failFn != nil {
		return m.failFn(ctx, logID, result)
	}
	return nil
}
func (m *mockSyncLogStore) List(ctx context.Context, userID uuid.UUID, plugin string, limit int32) ([]SyncLogEntry, error) {
	if m.listFn != nil {
		return m.listFn(ctx, userID, plugin, limit)
	}
	return nil, nil // no previous syncs
}

// mockTagStore implements TagStore for testing.
type mockTagStore struct {
	resolveTagFn       func(ctx context.Context, tag string) (uuid.UUID, string, error)
	getOrCreateFn      func(ctx context.Context, tag string) (uuid.UUID, error)
	addMediaItemTagFn  func(ctx context.Context, itemID, tagID uuid.UUID, source string, confidence *float32) error
	listMediaItemTagFn func(ctx context.Context, itemID uuid.UUID) ([]MediaItemTagInfo, error)
}

func (m *mockTagStore) ResolveTag(ctx context.Context, tag string) (uuid.UUID, string, error) {
	if m.resolveTagFn != nil {
		return m.resolveTagFn(ctx, tag)
	}
	return uuid.New(), tag, nil
}
func (m *mockTagStore) GetOrCreate(ctx context.Context, tag string) (uuid.UUID, error) {
	if m.getOrCreateFn != nil {
		return m.getOrCreateFn(ctx, tag)
	}
	return uuid.New(), nil
}
func (m *mockTagStore) AddMediaItemTag(ctx context.Context, itemID, tagID uuid.UUID, source string, confidence *float32) error {
	if m.addMediaItemTagFn != nil {
		return m.addMediaItemTagFn(ctx, itemID, tagID, source, confidence)
	}
	return nil
}
func (m *mockTagStore) ListMediaItemTags(ctx context.Context, itemID uuid.UUID) ([]MediaItemTagInfo, error) {
	if m.listMediaItemTagFn != nil {
		return m.listMediaItemTagFn(ctx, itemID)
	}
	return nil, nil
}

// mockEnricher implements Enricher for testing.
type mockEnricher struct {
	enrichFn func(ctx context.Context, item MediaItem, existingTags []string) (*TagResult, error)
}

func (m *mockEnricher) Enrich(ctx context.Context, item MediaItem, existingTags []string) (*TagResult, error) {
	if m.enrichFn != nil {
		return m.enrichFn(ctx, item, existingTags)
	}
	return &TagResult{}, nil
}

// --- Helpers ---

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func newTestSyncService(opts ...func(*SyncService)) *SyncService {
	reg := NewPluginRegistry()
	svc := &SyncService{
		Registry:    reg,
		Plugins:     &mockPluginStateStore{},
		Media:       &mockMediaItemStore{},
		SyncLogs:    &mockSyncLogStore{},
		Tags:        &mockTagStore{},
		Enricher:    &NoOpEnricher{},
		MinInterval: 1 * time.Millisecond, // very short for tests
		Logger:      discardLogger(),
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

// --- Tests ---

func TestSyncPlugin_HappyPath(t *testing.T) {
	userID := uuid.New()
	items := []MediaItem{
		{Platform: "spotify", Title: "Song A", ExternalID: "a", Type: MediaMusic, ConsumedAt: time.Now()},
		{Platform: "spotify", Title: "Song B", ExternalID: "b", Type: MediaMusic, ConsumedAt: time.Now()},
	}

	var createdCount int
	var completeCalled bool
	var cursorSaved *string

	svc := newTestSyncService(func(s *SyncService) {
		p := newMockPlugin("spotify")
		p.syncFn = func(_ context.Context, _ Credentials, _ string) SyncResult {
			return SyncResult{Items: items, NextCursor: "cursor-1"}
		}
		s.Registry.Register(p)

		s.Media = &mockMediaItemStore{
			createFn: func(_ context.Context, _ uuid.UUID, _ MediaItem) (uuid.UUID, error) {
				createdCount++
				return uuid.New(), nil
			},
		}

		s.SyncLogs = &mockSyncLogStore{
			completeFn: func(_ context.Context, _ uuid.UUID, result SyncLogResult) error {
				completeCalled = true
				if result.ItemsAdded != 2 {
					t.Errorf("expected 2 items added, got %d", result.ItemsAdded)
				}
				return nil
			},
		}

		s.Plugins = &mockPluginStateStore{
			updateSyncedFn: func(_ context.Context, _ uuid.UUID, _ string, cursor *string) (*PluginStateInfo, error) {
				cursorSaved = cursor
				return &PluginStateInfo{}, nil
			},
		}
	})

	summary, err := svc.SyncPlugin(context.Background(), userID, "spotify")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.ItemsAdded != 2 {
		t.Errorf("expected 2 items added, got %d", summary.ItemsAdded)
	}
	if createdCount != 2 {
		t.Errorf("expected 2 creates, got %d", createdCount)
	}
	if !completeCalled {
		t.Error("expected sync log Complete to be called")
	}
	if cursorSaved == nil || *cursorSaved != "cursor-1" {
		t.Errorf("expected cursor 'cursor-1', got %v", cursorSaved)
	}
}

func TestSyncPlugin_EmptyResult(t *testing.T) {
	userID := uuid.New()

	svc := newTestSyncService(func(s *SyncService) {
		p := newMockPlugin("spotify")
		p.syncFn = func(_ context.Context, _ Credentials, _ string) SyncResult {
			return SyncResult{Items: nil, NextCursor: ""}
		}
		s.Registry.Register(p)
	})

	summary, err := svc.SyncPlugin(context.Background(), userID, "spotify")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.ItemsAdded != 0 {
		t.Errorf("expected 0 items added, got %d", summary.ItemsAdded)
	}
}

func TestSyncPlugin_PluginNotFound(t *testing.T) {
	userID := uuid.New()
	svc := newTestSyncService()

	_, err := svc.SyncPlugin(context.Background(), userID, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent plugin")
	}
}

func TestSyncPlugin_RateLimited_TooSoon(t *testing.T) {
	userID := uuid.New()

	svc := newTestSyncService(func(s *SyncService) {
		s.MinInterval = 1 * time.Hour
		p := newMockPlugin("spotify")
		s.Registry.Register(p)

		s.SyncLogs = &mockSyncLogStore{
			listFn: func(_ context.Context, _ uuid.UUID, _ string, _ int32) ([]SyncLogEntry, error) {
				return []SyncLogEntry{
					{StartedAt: time.Now().Add(-5 * time.Minute), Status: "completed"},
				}, nil
			},
		}
	})

	_, err := svc.SyncPlugin(context.Background(), userID, "spotify")
	if err == nil {
		t.Fatal("expected rate limit error")
	}

	var rateLimitErr *RateLimitError
	if !errors.As(err, &rateLimitErr) {
		t.Fatalf("expected RateLimitError, got %T: %v", err, err)
	}
	if rateLimitErr.RetryAfter <= 0 {
		t.Errorf("expected positive RetryAfter, got %v", rateLimitErr.RetryAfter)
	}
}

func TestSyncPlugin_RateLimited_SyncInProgress(t *testing.T) {
	userID := uuid.New()

	svc := newTestSyncService(func(s *SyncService) {
		p := newMockPlugin("spotify")
		s.Registry.Register(p)

		s.SyncLogs = &mockSyncLogStore{
			listFn: func(_ context.Context, _ uuid.UUID, _ string, _ int32) ([]SyncLogEntry, error) {
				return []SyncLogEntry{
					{StartedAt: time.Now(), Status: "running"},
				}, nil
			},
		}
	})

	_, err := svc.SyncPlugin(context.Background(), userID, "spotify")
	if err == nil {
		t.Fatal("expected rate limit error for running sync")
	}

	var rateLimitErr *RateLimitError
	if !errors.As(err, &rateLimitErr) {
		t.Fatalf("expected RateLimitError, got %T: %v", err, err)
	}
	if rateLimitErr.Reason != "a sync is already in progress" {
		t.Errorf("unexpected reason: %s", rateLimitErr.Reason)
	}
}

func TestSyncPlugin_FirstSyncEver(t *testing.T) {
	userID := uuid.New()

	svc := newTestSyncService(func(s *SyncService) {
		p := newMockPlugin("spotify")
		p.syncFn = func(_ context.Context, _ Credentials, cursor string) SyncResult {
			if cursor != "" {
				t.Errorf("expected empty cursor for first sync, got %q", cursor)
			}
			return SyncResult{Items: []MediaItem{{Platform: "spotify", Title: "X", ExternalID: "x", Type: MediaMusic, ConsumedAt: time.Now()}}}
		}
		s.Registry.Register(p)

		s.Plugins = &mockPluginStateStore{
			getStateFn: func(_ context.Context, _ uuid.UUID, _ string) (*PluginStateInfo, error) {
				return &PluginStateInfo{Cursor: nil}, nil
			},
		}
	})

	summary, err := svc.SyncPlugin(context.Background(), userID, "spotify")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.ItemsAdded != 1 {
		t.Errorf("expected 1 item, got %d", summary.ItemsAdded)
	}
}

func TestSyncPlugin_UsesCursor(t *testing.T) {
	userID := uuid.New()
	var receivedCursor string

	svc := newTestSyncService(func(s *SyncService) {
		p := newMockPlugin("spotify")
		p.syncFn = func(_ context.Context, _ Credentials, cursor string) SyncResult {
			receivedCursor = cursor
			return SyncResult{Items: nil}
		}
		s.Registry.Register(p)

		existingCursor := "prev-cursor-abc"
		s.Plugins = &mockPluginStateStore{
			getStateFn: func(_ context.Context, _ uuid.UUID, _ string) (*PluginStateInfo, error) {
				return &PluginStateInfo{Cursor: &existingCursor}, nil
			},
		}
	})

	_, err := svc.SyncPlugin(context.Background(), userID, "spotify")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedCursor != "prev-cursor-abc" {
		t.Errorf("expected cursor 'prev-cursor-abc', got %q", receivedCursor)
	}
}

func TestSyncPlugin_AuthExpired(t *testing.T) {
	userID := uuid.New()
	var statusUpdated string
	var failCalled bool

	svc := newTestSyncService(func(s *SyncService) {
		p := newMockPlugin("spotify")
		p.syncFn = func(_ context.Context, _ Credentials, _ string) SyncResult {
			return SyncResult{
				Err: &PluginError{Code: ErrAuthExpired, Message: "token expired"},
			}
		}
		s.Registry.Register(p)

		s.Plugins = &mockPluginStateStore{
			updateStatusFn: func(_ context.Context, _ uuid.UUID, _ string, status string, _ *string) (*PluginStateInfo, error) {
				statusUpdated = status
				return &PluginStateInfo{Status: status}, nil
			},
		}

		s.SyncLogs = &mockSyncLogStore{
			failFn: func(_ context.Context, _ uuid.UUID, _ SyncLogResult) error {
				failCalled = true
				return nil
			},
		}
	})

	_, err := svc.SyncPlugin(context.Background(), userID, "spotify")
	if err == nil {
		t.Fatal("expected error for auth expired")
	}
	if statusUpdated != "disconnected" {
		t.Errorf("expected plugin status 'disconnected', got %q", statusUpdated)
	}
	if !failCalled {
		t.Error("expected sync log Fail to be called")
	}
}

func TestSyncPlugin_PlatformRateLimit(t *testing.T) {
	userID := uuid.New()

	svc := newTestSyncService(func(s *SyncService) {
		p := newMockPlugin("spotify")
		p.syncFn = func(_ context.Context, _ Credentials, _ string) SyncResult {
			return SyncResult{
				Err: &PluginError{
					Code:    ErrRateLimit,
					Message: "too many requests",
					After:   30 * time.Second,
				},
			}
		}
		s.Registry.Register(p)
	})

	_, err := svc.SyncPlugin(context.Background(), userID, "spotify")
	if err == nil {
		t.Fatal("expected error for rate limit")
	}

	var rateLimitErr *RateLimitError
	if !errors.As(err, &rateLimitErr) {
		t.Fatalf("expected RateLimitError, got %T: %v", err, err)
	}
	if rateLimitErr.RetryAfter != 30*time.Second {
		t.Errorf("expected RetryAfter 30s, got %v", rateLimitErr.RetryAfter)
	}
}

func TestSyncPlugin_PartialSync(t *testing.T) {
	userID := uuid.New()
	var cursorSaved *string

	svc := newTestSyncService(func(s *SyncService) {
		p := newMockPlugin("spotify")
		p.syncFn = func(_ context.Context, _ Credentials, _ string) SyncResult {
			return SyncResult{
				Items:      []MediaItem{{Platform: "spotify", Title: "A", ExternalID: "a", Type: MediaMusic, ConsumedAt: time.Now()}},
				NextCursor: "partial-cursor",
				Err:        &PluginError{Code: ErrPartialSync, Message: "connection dropped"},
			}
		}
		s.Registry.Register(p)

		s.Plugins = &mockPluginStateStore{
			updateSyncedFn: func(_ context.Context, _ uuid.UUID, _ string, cursor *string) (*PluginStateInfo, error) {
				cursorSaved = cursor
				return &PluginStateInfo{}, nil
			},
		}
	})

	summary, err := svc.SyncPlugin(context.Background(), userID, "spotify")

	// Should return both summary AND error
	if err == nil {
		t.Fatal("expected error for partial sync")
	}
	var pluginErr *PluginError
	if !errors.As(err, &pluginErr) {
		t.Fatalf("expected PluginError, got %T: %v", err, err)
	}
	if pluginErr.Code != ErrPartialSync {
		t.Errorf("expected ErrPartialSync, got %s", pluginErr.Code)
	}

	if summary == nil {
		t.Fatal("expected summary for partial sync")
	}
	if summary.ItemsAdded != 1 {
		t.Errorf("expected 1 item added, got %d", summary.ItemsAdded)
	}

	if cursorSaved == nil || *cursorSaved != "partial-cursor" {
		t.Errorf("expected cursor 'partial-cursor', got %v", cursorSaved)
	}
}

func TestSyncPlugin_UpstreamError(t *testing.T) {
	userID := uuid.New()
	var failCalled bool

	svc := newTestSyncService(func(s *SyncService) {
		p := newMockPlugin("spotify")
		p.syncFn = func(_ context.Context, _ Credentials, _ string) SyncResult {
			return SyncResult{
				Err: &PluginError{Code: ErrUpstream, Message: "server 500"},
			}
		}
		s.Registry.Register(p)

		s.SyncLogs = &mockSyncLogStore{
			failFn: func(_ context.Context, _ uuid.UUID, _ SyncLogResult) error {
				failCalled = true
				return nil
			},
		}
	})

	_, err := svc.SyncPlugin(context.Background(), userID, "spotify")
	if err == nil {
		t.Fatal("expected error for upstream failure")
	}
	if !failCalled {
		t.Error("expected sync log Fail to be called")
	}
}

func TestSyncPlugin_PluginEnrichmentFailure_NonFatal(t *testing.T) {
	userID := uuid.New()

	svc := newTestSyncService(func(s *SyncService) {
		p := newMockPlugin("spotify")
		p.syncFn = func(_ context.Context, _ Credentials, _ string) SyncResult {
			return SyncResult{
				Items: []MediaItem{{Platform: "spotify", Title: "A", ExternalID: "a", Type: MediaMusic, ConsumedAt: time.Now()}},
			}
		}
		p.enrichFn = func(_ context.Context, _ Credentials, _ []MediaItem) ([]MediaItem, error) {
			return nil, errors.New("enrichment service down")
		}
		s.Registry.Register(p)
	})

	// Should succeed despite plugin enrichment failure
	summary, err := svc.SyncPlugin(context.Background(), userID, "spotify")
	if err != nil {
		t.Fatalf("unexpected error (enrichment failure should be non-fatal): %v", err)
	}
	if summary.ItemsAdded != 1 {
		t.Errorf("expected 1 item, got %d", summary.ItemsAdded)
	}
}

func TestSyncPlugin_CoreEnrichment_TagsPersisted(t *testing.T) {
	userID := uuid.New()
	var taggedItems []string

	svc := newTestSyncService(func(s *SyncService) {
		p := newMockPlugin("spotify")
		p.syncFn = func(_ context.Context, _ Credentials, _ string) SyncResult {
			return SyncResult{
				Items: []MediaItem{{Platform: "spotify", Title: "Song", ExternalID: "s", Type: MediaMusic, ConsumedAt: time.Now()}},
			}
		}
		s.Registry.Register(p)

		s.Enricher = &mockEnricher{
			enrichFn: func(_ context.Context, _ MediaItem, _ []string) (*TagResult, error) {
				return &TagResult{
					Genre: []TagScore{{Tag: "rock", Confidence: 0.9}},
					Mood:  []TagScore{{Tag: "energetic", Confidence: 0.8}},
				}, nil
			},
		}

		s.Tags = &mockTagStore{
			addMediaItemTagFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, source string, _ *float32) error {
				taggedItems = append(taggedItems, source)
				return nil
			},
		}
	})

	summary, err := svc.SyncPlugin(context.Background(), userID, "spotify")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.ItemsAdded != 1 {
		t.Errorf("expected 1 item, got %d", summary.ItemsAdded)
	}
	if len(taggedItems) != 2 {
		t.Errorf("expected 2 tags persisted, got %d", len(taggedItems))
	}
	for _, source := range taggedItems {
		if source != "llm" {
			t.Errorf("expected source 'llm', got %q", source)
		}
	}
}

func TestSyncPlugin_HasMore(t *testing.T) {
	userID := uuid.New()

	svc := newTestSyncService(func(s *SyncService) {
		p := newMockPlugin("spotify")
		p.syncFn = func(_ context.Context, _ Credentials, _ string) SyncResult {
			return SyncResult{
				Items:   []MediaItem{{Platform: "spotify", Title: "A", ExternalID: "a", Type: MediaMusic, ConsumedAt: time.Now()}},
				HasMore: true,
			}
		}
		s.Registry.Register(p)
	})

	summary, err := svc.SyncPlugin(context.Background(), userID, "spotify")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !summary.HasMore {
		t.Error("expected HasMore to be true")
	}
}

func TestSyncPlugin_NoCursorSavedWhenEmpty(t *testing.T) {
	userID := uuid.New()
	var cursorSaved *string
	syncedCalled := false

	svc := newTestSyncService(func(s *SyncService) {
		p := newMockPlugin("spotify")
		p.syncFn = func(_ context.Context, _ Credentials, _ string) SyncResult {
			return SyncResult{Items: nil, NextCursor: ""}
		}
		s.Registry.Register(p)

		s.Plugins = &mockPluginStateStore{
			updateSyncedFn: func(_ context.Context, _ uuid.UUID, _ string, cursor *string) (*PluginStateInfo, error) {
				syncedCalled = true
				cursorSaved = cursor
				return &PluginStateInfo{}, nil
			},
		}
	})

	_, err := svc.SyncPlugin(context.Background(), userID, "spotify")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !syncedCalled {
		t.Error("expected UpdateSynced to be called")
	}
	if cursorSaved != nil {
		t.Errorf("expected nil cursor for empty NextCursor, got %v", cursorSaved)
	}
}

func TestSyncPlugin_StoreItemFailure_CountsAsSkipped(t *testing.T) {
	userID := uuid.New()

	svc := newTestSyncService(func(s *SyncService) {
		p := newMockPlugin("spotify")
		p.syncFn = func(_ context.Context, _ Credentials, _ string) SyncResult {
			return SyncResult{
				Items: []MediaItem{
					{Platform: "spotify", Title: "Good", ExternalID: "g", Type: MediaMusic, ConsumedAt: time.Now()},
					{Platform: "spotify", Title: "Bad", ExternalID: "b", Type: MediaMusic, ConsumedAt: time.Now()},
					{Platform: "spotify", Title: "Also Good", ExternalID: "ag", Type: MediaMusic, ConsumedAt: time.Now()},
				},
			}
		}
		s.Registry.Register(p)

		s.Media = &mockMediaItemStore{
			createFn: func(_ context.Context, _ uuid.UUID, item MediaItem) (uuid.UUID, error) {
				if item.Title == "Bad" {
					return uuid.Nil, errors.New("db constraint violation")
				}
				return uuid.New(), nil
			},
		}
	})

	summary, err := svc.SyncPlugin(context.Background(), userID, "spotify")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.ItemsAdded != 2 {
		t.Errorf("expected 2 items added, got %d", summary.ItemsAdded)
	}
	if summary.ItemsSkipped != 1 {
		t.Errorf("expected 1 item skipped, got %d", summary.ItemsSkipped)
	}
}

func TestSyncPlugin_CredentialLoadFailure(t *testing.T) {
	userID := uuid.New()

	svc := newTestSyncService(func(s *SyncService) {
		p := newMockPlugin("spotify")
		s.Registry.Register(p)

		s.Plugins = &mockPluginStateStore{
			getCredentialsFn: func(_ context.Context, _ uuid.UUID, _ string) (*Credentials, error) {
				return nil, errors.New("decryption failed")
			},
		}
	})

	_, err := svc.SyncPlugin(context.Background(), userID, "spotify")
	if err == nil {
		t.Fatal("expected error for credential load failure")
	}
}

func TestSyncPlugin_InvalidDataError(t *testing.T) {
	userID := uuid.New()

	svc := newTestSyncService(func(s *SyncService) {
		p := newMockPlugin("spotify")
		p.syncFn = func(_ context.Context, _ Credentials, _ string) SyncResult {
			return SyncResult{
				Err: &PluginError{Code: ErrInvalidData, Message: "unexpected JSON"},
			}
		}
		s.Registry.Register(p)
	})

	_, err := svc.SyncPlugin(context.Background(), userID, "spotify")
	if err == nil {
		t.Fatal("expected error for invalid data")
	}
}

func TestSyncPlugin_PermissionDeniedError(t *testing.T) {
	userID := uuid.New()

	svc := newTestSyncService(func(s *SyncService) {
		p := newMockPlugin("spotify")
		p.syncFn = func(_ context.Context, _ Credentials, _ string) SyncResult {
			return SyncResult{
				Err: &PluginError{Code: ErrPermissionDenied, Message: "insufficient scopes"},
			}
		}
		s.Registry.Register(p)
	})

	_, err := svc.SyncPlugin(context.Background(), userID, "spotify")
	if err == nil {
		t.Fatal("expected error for permission denied")
	}
}

func TestSyncPlugin_FileParseError(t *testing.T) {
	userID := uuid.New()

	svc := newTestSyncService(func(s *SyncService) {
		p := newMockPlugin("netflix")
		p.syncFn = func(_ context.Context, _ Credentials, _ string) SyncResult {
			return SyncResult{
				Err: &PluginError{Code: ErrFileParseError, Message: "invalid CSV at line 3"},
			}
		}
		s.Registry.Register(p)
	})

	_, err := svc.SyncPlugin(context.Background(), userID, "netflix")
	if err == nil {
		t.Fatal("expected error for file parse error")
	}
}

func TestNewSyncService_Defaults(t *testing.T) {
	reg := NewPluginRegistry()
	svc := NewSyncService(reg, &mockPluginStateStore{}, &mockMediaItemStore{}, &mockSyncLogStore{}, &mockTagStore{}, &NoOpEnricher{}, 0, nil)

	if svc.MinInterval != DefaultMinSyncInterval {
		t.Errorf("expected default interval %v, got %v", DefaultMinSyncInterval, svc.MinInterval)
	}
	if svc.Logger == nil {
		t.Error("expected non-nil logger")
	}
}

func TestRateLimitError_Error(t *testing.T) {
	err := &RateLimitError{
		Plugin:     "spotify",
		RetryAfter: 10 * time.Minute,
		Reason:     "too soon",
	}
	msg := err.Error()
	if msg == "" {
		t.Error("expected non-empty error message")
	}
}

func TestSyncPlugin_CoreEnrichment_InvalidTagsFiltered(t *testing.T) {
	userID := uuid.New()
	var addedTags []string

	svc := newTestSyncService(func(s *SyncService) {
		p := newMockPlugin("spotify")
		p.syncFn = func(_ context.Context, _ Credentials, _ string) SyncResult {
			return SyncResult{
				Items: []MediaItem{{Platform: "spotify", Title: "Song", ExternalID: "s", Type: MediaMusic, ConsumedAt: time.Now()}},
			}
		}
		s.Registry.Register(p)

		// Return some valid and some invalid tags
		s.Enricher = &mockEnricher{
			enrichFn: func(_ context.Context, _ MediaItem, _ []string) (*TagResult, error) {
				return &TagResult{
					Genre: []TagScore{
						{Tag: "rock", Confidence: 0.9},       // valid
						{Tag: "fake-genre", Confidence: 0.5}, // invalid — not in fixed set
					},
				}, nil
			},
		}

		s.Tags = &mockTagStore{
			getOrCreateFn: func(_ context.Context, tag string) (uuid.UUID, error) {
				addedTags = append(addedTags, tag)
				return uuid.New(), nil
			},
		}
	})

	_, err := svc.SyncPlugin(context.Background(), userID, "spotify")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only "rock" should pass validation; "fake-genre" should be filtered
	if len(addedTags) != 1 {
		t.Errorf("expected 1 tag to be created, got %d: %v", len(addedTags), addedTags)
	}
	if len(addedTags) > 0 && addedTags[0] != "rock" {
		t.Errorf("expected tag 'rock', got %q", addedTags[0])
	}
}
