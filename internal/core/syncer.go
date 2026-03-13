package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// DefaultMinSyncInterval is the minimum time between syncs per user per plugin.
const DefaultMinSyncInterval = 15 * time.Minute

// TokenRefreshResult holds the result of a token refresh.
type TokenRefreshResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

// TokenRefresher refreshes OAuth tokens for a plugin. Implemented by auth.OAuthService.
type TokenRefresher interface {
	RefreshPluginToken(pluginName string, cfg *OAuthConfig, refreshToken string) (*TokenRefreshResult, error)
}

// SyncService orchestrates the full sync flow for a plugin.
// It coordinates between the plugin registry, credential store,
// media item store, enricher, tag store, and sync log.
type SyncService struct {
	Registry       *PluginRegistry
	Plugins        PluginStateStore
	Media          MediaItemStore
	SyncLogs       SyncLogStore
	Tags           TagStore
	Enricher       Enricher
	TokenRefresher TokenRefresher // optional — if set, auto-refreshes expired tokens
	MinInterval    time.Duration  // minimum interval between syncs; defaults to DefaultMinSyncInterval
	Logger         *slog.Logger
}

// NewSyncService creates a SyncService with the given dependencies.
// If minInterval is zero, DefaultMinSyncInterval is used.
// If logger is nil, the default slog logger is used.
func NewSyncService(
	registry *PluginRegistry,
	plugins PluginStateStore,
	media MediaItemStore,
	syncLogs SyncLogStore,
	tags TagStore,
	enricher Enricher,
	minInterval time.Duration,
	logger *slog.Logger,
) *SyncService {
	if minInterval <= 0 {
		minInterval = DefaultMinSyncInterval
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &SyncService{
		Registry:    registry,
		Plugins:     plugins,
		Media:       media,
		SyncLogs:    syncLogs,
		Tags:        tags,
		Enricher:    enricher,
		MinInterval: minInterval,
		Logger:      logger,
	}
}

// SyncSummary is the result returned to callers after a sync completes.
type SyncSummary struct {
	ItemsAdded   int32
	ItemsSkipped int32 // items that were duplicates (already existed)
	ItemsUpdated int32 // items that were updated (upserted with new data)
	HasMore      bool  // true if there are more items to fetch (pagination not exhausted)
	Error        error // nil on full success, non-nil on failure or partial success
}

// SyncPlugin runs the full sync flow for a single plugin for a user.
//
// The flow is:
//  1. Rate-limit check (minimum interval between syncs per user/plugin)
//  2. Look up plugin from registry
//  3. Load and decrypt credentials from store
//  4. Get cursor from plugin state
//  5. Call plugin.Sync(credentials, cursor)
//  6. Store items (upsert handles deduplication)
//  7. Run plugin-specific enrichment (plugin.Enrich)
//  8. Run core LLM enrichment on new items
//  9. Persist tags from enrichment
//  10. Update sync_log (success/failure/item counts)
//  11. Update plugin_state cursor
//
// PluginError codes are handled:
//   - auth_expired: marks plugin as disconnected
//   - rate_limited: returns rate-limit error with retry hint
//   - partial_sync: stores partial items + cursor, logs error
//   - upstream: logs error, no retry in this call
//   - invalid_data, permission_denied, file_parse_error: logs and returns
func (s *SyncService) SyncPlugin(ctx context.Context, userID uuid.UUID, pluginName string) (*SyncSummary, error) {
	log := s.Logger.With("user_id", userID, "plugin", pluginName)

	// Step 1: Rate-limit check
	if err := s.checkRateLimit(ctx, userID, pluginName); err != nil {
		return nil, err
	}

	// Step 2: Look up plugin from registry
	plugin := s.Registry.Get(pluginName)
	if plugin == nil {
		return nil, fmt.Errorf("plugin %q not found in registry", pluginName)
	}

	// Step 3: Load and decrypt credentials
	creds, err := s.Plugins.GetCredentials(ctx, userID, pluginName)
	if err != nil {
		return nil, fmt.Errorf("loading credentials for %q: %w", pluginName, err)
	}

	// Step 3b: Auto-refresh token if plugin uses OAuth and a refresher is configured
	if plugin.AuthType() == AuthOAuth && s.TokenRefresher != nil && creds.RefreshToken != "" {
		refreshedCreds, refreshErr := s.tryRefreshToken(ctx, userID, pluginName, plugin, creds, log)
		if refreshErr != nil {
			log.Warn("token refresh failed, proceeding with existing token", "error", refreshErr)
		} else if refreshedCreds != nil {
			creds = refreshedCreds
		}
	}

	// Step 4: Get cursor from plugin state
	state, err := s.Plugins.GetState(ctx, userID, pluginName)
	if err != nil {
		return nil, fmt.Errorf("loading plugin state for %q: %w", pluginName, err)
	}

	cursor := ""
	if state.Cursor != nil {
		cursor = *state.Cursor
	}

	// Begin sync log
	logID, err := s.SyncLogs.Begin(ctx, userID, pluginName)
	if err != nil {
		return nil, fmt.Errorf("creating sync log: %w", err)
	}

	startTime := time.Now()

	// Step 5: Call plugin.Sync
	log.Info("starting sync", "cursor", cursor)
	result := plugin.Sync(ctx, *creds, cursor)

	// Handle plugin errors
	if result.Err != nil {
		return s.handleSyncError(ctx, userID, pluginName, logID, startTime, &result, log)
	}

	// Step 6: Store items (upsert handles dedup)
	summary, err := s.storeItems(ctx, userID, result.Items, log)
	if err != nil {
		s.failSyncLog(ctx, logID, startTime, summary, err, log)
		return summary, fmt.Errorf("storing items: %w", err)
	}
	summary.HasMore = result.HasMore

	// Step 7: Plugin-specific enrichment
	enrichedItems, err := plugin.Enrich(ctx, *creds, result.Items)
	if err != nil {
		// Plugin enrichment failure is non-fatal — log and continue
		log.Warn("plugin enrichment failed, continuing with unenriched items", "error", err)
		enrichedItems = result.Items
	}

	// Step 8 & 9: Core enrichment + tag persistence
	s.enrichAndTagItems(ctx, userID, enrichedItems, log)

	// Step 10: Complete sync log
	durationMs := int32(time.Since(startTime).Milliseconds())
	err = s.SyncLogs.Complete(ctx, logID, SyncLogResult{
		ItemsAdded:   summary.ItemsAdded,
		ItemsSkipped: summary.ItemsSkipped,
		ItemsUpdated: summary.ItemsUpdated,
		DurationMs:   durationMs,
	})
	if err != nil {
		log.Error("failed to complete sync log", "error", err)
	}

	// Step 11: Update plugin state cursor
	newCursor := &result.NextCursor
	if result.NextCursor == "" {
		newCursor = nil
	}
	_, err = s.Plugins.UpdateSynced(ctx, userID, pluginName, newCursor)
	if err != nil {
		log.Error("failed to update plugin state cursor", "error", err)
	}

	log.Info("sync completed",
		"items_added", summary.ItemsAdded,
		"items_skipped", summary.ItemsSkipped,
		"has_more", summary.HasMore,
	)

	return summary, nil
}

// SyncPluginWithFile runs the full sync flow for a file-import plugin,
// injecting the provided file reader into the credentials. This is needed
// because io.Reader cannot be serialized to the credential store — the
// handler must pass the file reader directly.
func (s *SyncService) SyncPluginWithFile(ctx context.Context, userID uuid.UUID, pluginName string, file io.Reader) (*SyncSummary, error) {
	log := s.Logger.With("user_id", userID, "plugin", pluginName)

	// Step 1: Rate-limit check
	if err := s.checkRateLimit(ctx, userID, pluginName); err != nil {
		return nil, err
	}

	// Step 2: Look up plugin from registry
	plugin := s.Registry.Get(pluginName)
	if plugin == nil {
		return nil, fmt.Errorf("plugin %q not found in registry", pluginName)
	}

	// Step 3: Build credentials with the file reader directly
	creds := Credentials{File: file}

	// Step 4: Get cursor from plugin state (always empty for file imports)
	state, err := s.Plugins.GetState(ctx, userID, pluginName)
	if err != nil {
		return nil, fmt.Errorf("loading plugin state for %q: %w", pluginName, err)
	}

	cursor := ""
	if state.Cursor != nil {
		cursor = *state.Cursor
	}

	// Begin sync log
	logID, err := s.SyncLogs.Begin(ctx, userID, pluginName)
	if err != nil {
		return nil, fmt.Errorf("creating sync log: %w", err)
	}

	startTime := time.Now()

	// Step 5: Call plugin.Sync with file credentials
	log.Info("starting file import sync", "cursor", cursor)
	result := plugin.Sync(ctx, creds, cursor)

	// Handle plugin errors
	if result.Err != nil {
		return s.handleSyncError(ctx, userID, pluginName, logID, startTime, &result, log)
	}

	// Step 6: Store items (upsert handles dedup)
	summary, err := s.storeItems(ctx, userID, result.Items, log)
	if err != nil {
		s.failSyncLog(ctx, logID, startTime, summary, err, log)
		return summary, fmt.Errorf("storing items: %w", err)
	}
	summary.HasMore = result.HasMore

	// Step 7: Plugin-specific enrichment (skip file for enrichment creds)
	enrichedItems, err := plugin.Enrich(ctx, creds, result.Items)
	if err != nil {
		log.Warn("plugin enrichment failed, continuing with unenriched items", "error", err)
		enrichedItems = result.Items
	}

	// Step 8 & 9: Core enrichment + tag persistence
	s.enrichAndTagItems(ctx, userID, enrichedItems, log)

	// Step 10: Complete sync log
	durationMs := int32(time.Since(startTime).Milliseconds())
	err = s.SyncLogs.Complete(ctx, logID, SyncLogResult{
		ItemsAdded:   summary.ItemsAdded,
		ItemsSkipped: summary.ItemsSkipped,
		ItemsUpdated: summary.ItemsUpdated,
		DurationMs:   durationMs,
	})
	if err != nil {
		log.Error("failed to complete sync log", "error", err)
	}

	// Step 11: Update plugin state cursor
	newCursor := &result.NextCursor
	if result.NextCursor == "" {
		newCursor = nil
	}
	_, err = s.Plugins.UpdateSynced(ctx, userID, pluginName, newCursor)
	if err != nil {
		log.Error("failed to update plugin state cursor", "error", err)
	}

	log.Info("file import sync completed",
		"items_added", summary.ItemsAdded,
		"items_skipped", summary.ItemsSkipped,
	)

	return summary, nil
}

// tryRefreshToken attempts to refresh an OAuth token and update stored credentials.
// Returns updated credentials if refresh succeeded, nil if skipped, or an error.
func (s *SyncService) tryRefreshToken(
	ctx context.Context,
	userID uuid.UUID,
	pluginName string,
	plugin SourcePlugin,
	creds *Credentials,
	log *slog.Logger,
) (*Credentials, error) {
	cfg := plugin.AuthConfig()
	if cfg == nil {
		return nil, nil
	}

	result, err := s.TokenRefresher.RefreshPluginToken(pluginName, cfg, creds.RefreshToken)
	if err != nil {
		return nil, err
	}

	log.Info("token refreshed successfully", "plugin", pluginName)

	newCreds := &Credentials{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	}
	// Keep old refresh token if provider didn't issue a new one
	if newCreds.RefreshToken == "" {
		newCreds.RefreshToken = creds.RefreshToken
	}

	// Compute new expiry
	var expiresAt *time.Time
	if result.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
		expiresAt = &t
	}

	// Persist refreshed credentials
	if err := s.Plugins.UpsertCredentials(ctx, userID, pluginName, AuthOAuth, *newCreds, expiresAt); err != nil {
		log.Error("failed to persist refreshed credentials", "error", err)
		// Still use the new token for this sync even if persist fails
	}

	return newCreds, nil
}

// checkRateLimit verifies enough time has passed since the last sync.
func (s *SyncService) checkRateLimit(ctx context.Context, userID uuid.UUID, plugin string) error {
	logs, err := s.SyncLogs.List(ctx, userID, plugin, 1)
	if err != nil {
		return fmt.Errorf("checking rate limit: %w", err)
	}
	if len(logs) == 0 {
		return nil // first sync ever
	}

	last := logs[0]
	// If the last sync is still running, don't allow a new one
	if last.Status == "running" {
		return &RateLimitError{
			Plugin:     plugin,
			RetryAfter: s.MinInterval,
			Reason:     "a sync is already in progress",
		}
	}

	elapsed := time.Since(last.StartedAt)
	if elapsed < s.MinInterval {
		return &RateLimitError{
			Plugin:     plugin,
			RetryAfter: s.MinInterval - elapsed,
			Reason:     fmt.Sprintf("last sync was %s ago, minimum interval is %s", elapsed.Round(time.Second), s.MinInterval),
		}
	}
	return nil
}

// handleSyncError processes a PluginError from plugin.Sync() and takes
// appropriate action based on the error code.
func (s *SyncService) handleSyncError(
	ctx context.Context,
	userID uuid.UUID,
	pluginName string,
	logID uuid.UUID,
	startTime time.Time,
	result *SyncResult,
	log *slog.Logger,
) (*SyncSummary, error) {
	pluginErr := result.Err
	log.Warn("plugin sync returned error",
		"code", pluginErr.Code,
		"message", pluginErr.Message,
	)

	switch pluginErr.Code {
	case ErrAuthExpired:
		// Mark plugin as disconnected — user needs to re-authorize
		errMsg := pluginErr.Message
		_, stateErr := s.Plugins.UpdateStatus(ctx, userID, pluginName, "disconnected", &errMsg)
		if stateErr != nil {
			log.Error("failed to mark plugin as disconnected", "error", stateErr)
		}
		s.failSyncLog(ctx, logID, startTime, &SyncSummary{}, pluginErr, log)
		return nil, fmt.Errorf("auth expired for %q: %w", pluginName, pluginErr)

	case ErrRateLimit:
		// Platform rate limit — return with retry hint
		s.failSyncLog(ctx, logID, startTime, &SyncSummary{}, pluginErr, log)
		return nil, &RateLimitError{
			Plugin:     pluginName,
			RetryAfter: pluginErr.After,
			Reason:     fmt.Sprintf("platform rate limit: %s", pluginErr.Message),
		}

	case ErrPartialSync:
		// Store partial items and update cursor
		summary, storeErr := s.storeItems(ctx, userID, result.Items, log)
		if storeErr != nil {
			log.Error("failed to store partial sync items", "error", storeErr)
		}
		summary.HasMore = result.HasMore
		summary.Error = pluginErr

		// Still update cursor for partial sync so next sync resumes from here
		if result.NextCursor != "" {
			newCursor := result.NextCursor
			_, cursorErr := s.Plugins.UpdateSynced(ctx, userID, pluginName, &newCursor)
			if cursorErr != nil {
				log.Error("failed to update cursor after partial sync", "error", cursorErr)
			}
		}

		// Log as partial success
		durationMs := int32(time.Since(startTime).Milliseconds())
		errCode := string(pluginErr.Code)
		errMsg := pluginErr.Message
		logErr := s.SyncLogs.Fail(ctx, logID, SyncLogResult{
			ItemsAdded:   summary.ItemsAdded,
			ItemsSkipped: summary.ItemsSkipped,
			ItemsUpdated: summary.ItemsUpdated,
			ErrorCode:    &errCode,
			ErrorMessage: &errMsg,
			DurationMs:   durationMs,
		})
		if logErr != nil {
			log.Error("failed to record partial sync log", "error", logErr)
		}

		return summary, pluginErr

	default:
		// upstream, invalid_data, permission_denied, file_parse_error
		s.failSyncLog(ctx, logID, startTime, &SyncSummary{}, pluginErr, log)
		return nil, fmt.Errorf("sync failed for %q [%s]: %w", pluginName, pluginErr.Code, pluginErr)
	}
}

// storeItems upserts fetched media items and returns counts.
func (s *SyncService) storeItems(
	ctx context.Context,
	userID uuid.UUID,
	items []MediaItem,
	log *slog.Logger,
) (*SyncSummary, error) {
	summary := &SyncSummary{}

	for i, item := range items {
		_, err := s.Media.Create(ctx, userID, item)
		if err != nil {
			// If we fail to store an item, log and count as skipped
			log.Warn("failed to store item, skipping",
				"index", i,
				"title", item.Title,
				"error", err,
			)
			summary.ItemsSkipped++
			continue
		}
		// The store does an upsert on (user_id, platform, external_id).
		// We count all successful upserts as "added" since the store
		// handles dedup transparently.
		summary.ItemsAdded++
	}

	return summary, nil
}

// enrichAndTagItems runs core enrichment on items and persists resulting tags.
func (s *SyncService) enrichAndTagItems(
	ctx context.Context,
	userID uuid.UUID,
	items []MediaItem,
	log *slog.Logger,
) {
	for _, item := range items {
		// Run core enrichment (LLM or NoOp)
		tagResult, err := s.Enricher.Enrich(ctx, item, item.Tags)
		if err != nil {
			log.Warn("core enrichment failed for item, skipping",
				"title", item.Title,
				"error", err,
			)
			continue
		}

		if tagResult == nil || tagResult.IsEmpty() {
			continue
		}

		// Validate against fixed tag set
		validated := ValidateTagResult(tagResult)
		if validated.IsEmpty() {
			continue
		}

		// Look up the item's UUID by its external ID.
		_, itemID, lookupErr := s.Media.GetByExternalID(ctx, userID, item.Platform, item.ExternalID)
		if lookupErr != nil {
			log.Warn("failed to look up item for tagging",
				"title", item.Title,
				"error", lookupErr,
			)
			continue
		}

		for _, ts := range validated.AllTags() {
			tagID, err := s.Tags.GetOrCreate(ctx, ts.Tag)
			if err != nil {
				log.Warn("failed to get/create tag",
					"tag", ts.Tag,
					"error", err,
				)
				continue
			}

			conf := ts.Confidence
			err = s.Tags.AddMediaItemTag(ctx, itemID, tagID, "llm", &conf)
			if err != nil {
				log.Warn("failed to add tag to item",
					"tag", ts.Tag,
					"item_title", item.Title,
					"error", err,
				)
			}
		}
	}
}

// failSyncLog records a failed sync in the sync log.
func (s *SyncService) failSyncLog(
	ctx context.Context,
	logID uuid.UUID,
	startTime time.Time,
	summary *SyncSummary,
	syncErr error,
	log *slog.Logger,
) {
	durationMs := int32(time.Since(startTime).Milliseconds())
	errMsg := syncErr.Error()

	var errCode *string
	var pluginErr *PluginError
	if errors.As(syncErr, &pluginErr) {
		code := string(pluginErr.Code)
		errCode = &code
	}

	failErr := s.SyncLogs.Fail(ctx, logID, SyncLogResult{
		ItemsAdded:   summary.ItemsAdded,
		ItemsSkipped: summary.ItemsSkipped,
		ItemsUpdated: summary.ItemsUpdated,
		ErrorCode:    errCode,
		ErrorMessage: &errMsg,
		DurationMs:   durationMs,
	})
	if failErr != nil {
		log.Error("failed to record sync failure in log", "error", failErr)
	}
}

// RateLimitError is returned when a sync is rejected due to rate limiting.
type RateLimitError struct {
	Plugin     string
	RetryAfter time.Duration
	Reason     string
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limited for plugin %q: %s (retry after %s)", e.Plugin, e.Reason, e.RetryAfter.Round(time.Second))
}
