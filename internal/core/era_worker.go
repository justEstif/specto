package core

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// EraDetectionInterval is how often the era worker runs.
const EraDetectionInterval = 1 * time.Hour

// EraWorker is a background goroutine that periodically recomputes eras
// for all users. It follows the same Start/tick pattern as EnrichmentWorker.
type EraWorker struct {
	eras     EraStore
	users    UserStore
	namer    EraNamer // optional LLM namer (nil = no suggested titles)
	interval time.Duration
	logger   *slog.Logger
}

// EraWorkerConfig holds optional configuration for the era worker.
type EraWorkerConfig struct {
	PollInterval time.Duration
}

// NewEraWorker creates a new background era detection worker.
func NewEraWorker(
	eras EraStore,
	users UserStore,
	namer EraNamer,
	logger *slog.Logger,
	cfg EraWorkerConfig,
) *EraWorker {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = EraDetectionInterval
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &EraWorker{
		eras:     eras,
		users:    users,
		namer:    namer,
		interval: cfg.PollInterval,
		logger:   logger,
	}
}

// Start begins the polling loop. It blocks until ctx is cancelled.
func (w *EraWorker) Start(ctx context.Context) {
	w.logger.Info("era worker started", "poll_interval", w.interval)

	// Run once immediately on startup.
	w.tick(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("era worker stopping")
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

// tick recomputes eras for all users across all media types.
func (w *EraWorker) tick(ctx context.Context) {
	start := time.Now()

	// For now, detect eras for all known media types.
	mediaTypes := []string{
		string(MediaMusic),
		string(MediaVideo),
	}

	// Look back 2 years from now.
	to := time.Now()
	from := to.AddDate(-2, 0, 0)

	we := map[string]any{
		"eras_created":  0,
		"users_scanned": 0,
		"errors":        0,
	}

	userIDs, err := w.users.ListUserIDsWithEnrichedItems(ctx)
	if err != nil {
		we["outcome"] = "error"
		we["error"] = err.Error()
		we["duration_ms"] = time.Since(start).Milliseconds()
		we["media_types"] = mediaTypes
		we["range_from"] = from
		we["range_to"] = to
		w.logger.Error("era_detection_tick", wideAttrs(we)...)
		return
	}

	we["users_scanned"] = len(userIDs)

	for _, uid := range userIDs {
		for _, mt := range mediaTypes {
			if err := w.detectForType(ctx, uid, mt, from, to); err != nil {
				w.logger.Error("era_detection",
					"user_id", uid,
					"media_type", mt,
					"error", err,
				)
				we["errors"] = we["errors"].(int) + 1
			}
		}
	}

	we["outcome"] = "success"
	we["duration_ms"] = time.Since(start).Milliseconds()
	we["media_types"] = mediaTypes
	we["range_from"] = from
	we["range_to"] = to
	w.logger.Info("era_detection_tick", wideAttrs(we)...)
}

// DetectUserEras runs era detection for a single user across all media types.
// This is the main entry point, called by the worker tick or on-demand.
func (w *EraWorker) DetectUserEras(ctx context.Context, userID uuid.UUID) error {
	mediaTypes := []string{
		string(MediaMusic),
		string(MediaVideo),
	}

	to := time.Now()
	from := to.AddDate(-2, 0, 0)

	for _, mt := range mediaTypes {
		if err := w.detectForType(ctx, userID, mt, from, to); err != nil {
			w.logger.Error("era_detection",
				"user_id", userID,
				"media_type", mt,
				"error", err,
			)
			// Continue with other media types.
		}
	}
	return nil
}

// detectForType runs era detection for a single user + media type.
func (w *EraWorker) detectForType(ctx context.Context, userID uuid.UUID, mediaType string, from, to time.Time) error {
	// Step 1: Fetch tag vectors by window.
	entries, err := w.eras.TagVectorByWindow(ctx, userID, from, to, mediaType)
	if err != nil {
		return fmt.Errorf("fetching tag vectors: %w", err)
	}

	if len(entries) == 0 {
		return nil // no data for this media type
	}

	// Step 2: Build windows.
	windows := BuildWindows(entries)
	if len(windows) == 0 {
		return nil // not enough data in any window
	}

	// Step 3: Detect eras.
	boundaries := DetectEras(windows)
	if len(boundaries) == 0 {
		return nil
	}

	// Step 4: Clear previous suggested eras (preserve confirmed/dismissed).
	if err := w.eras.DeleteSuggested(ctx, userID, mediaType); err != nil {
		return fmt.Errorf("clearing suggested eras: %w", err)
	}

	// Step 5: Persist new eras.
	for _, b := range boundaries {
		// Get actual item count from the database.
		endTime := to
		if b.EndedAt != nil {
			endTime = *b.EndedAt
		}
		itemCount, err := w.eras.CountItemsInRange(ctx, userID, mediaType, b.StartedAt, endTime)
		if err != nil {
			return fmt.Errorf("counting items: %w", err)
		}

		era, err := w.eras.Create(ctx, Era{
			UserID:          userID,
			MediaType:       &mediaType,
			StartedAt:       b.StartedAt,
			EndedAt:         b.EndedAt,
			ItemCount:       int32(itemCount),
			Distinctiveness: b.Distinctiveness,
			Status:          EraStatusSuggested,
		})
		if err != nil {
			return fmt.Errorf("creating era: %w", err)
		}

		// Persist top tags.
		for _, tag := range b.TopTags {
			// We need the tag ID. Look it up from the entries.
			tagID := findTagID(entries, tag.TagName)
			if tagID == uuid.Nil {
				continue
			}
			if err := w.eras.UpsertTag(ctx, era.ID, tagID, tag.Weight); err != nil {
				w.logger.Error("era_detection",
					"outcome", "tag_upsert_failed",
					"era_id", era.ID,
					"tag", tag.TagName,
					"error", err,
				)
			}
		}

		// Generate suggested title via LLM (non-fatal).
		if w.namer != nil {
			title, err := w.namer.NameEra(ctx, mediaType, b.TopTags)
			if err != nil {
				w.logger.Error("era_detection",
					"outcome", "naming_failed",
					"era_id", era.ID,
					"error", err,
				)
			} else if title != "" {
				if _, err := w.eras.UpdateSuggestedTitle(ctx, era.ID, title); err != nil {
					w.logger.Error("era_detection",
						"outcome", "suggested_title_failed",
						"era_id", era.ID,
						"error", err,
					)
				}
			}
		}
	}

	w.logger.Info("era_detection",
		"user_id", userID,
		"media_type", mediaType,
		"eras_detected", len(boundaries),
	)

	return nil
}

// findTagID looks up a tag's UUID from the window entries by name.
func findTagID(entries []WindowTagEntry, tagName string) uuid.UUID {
	for _, e := range entries {
		if e.TagName == tagName {
			return e.TagID
		}
	}
	return uuid.Nil
}
