package core

import (
	"context"
	"log/slog"
	"time"
)

// DefaultPollInterval is how often the worker checks for pending items.
const DefaultPollInterval = 5 * time.Second

// DefaultBatchSize is the number of items claimed per tick.
const DefaultBatchSize int32 = 50

// DefaultMaxRetries is the per-item retry limit before marking as 'failed'.
const DefaultMaxRetries int32 = 3

// EnrichmentWorker is a background goroutine that polls for pending items
// and runs them through the EnrichmentCoordinator. It uses
// SELECT ... FOR UPDATE SKIP LOCKED for safe concurrent processing.
type EnrichmentWorker struct {
	coordinator *EnrichmentCoordinator
	media       MediaItemStore
	tags        TagStore
	batchSize   int32
	maxRetries  int32
	interval    time.Duration
	logger      *slog.Logger
}

// EnrichmentWorkerConfig holds optional configuration for the worker.
type EnrichmentWorkerConfig struct {
	BatchSize    int32
	MaxRetries   int32
	PollInterval time.Duration
}

// NewEnrichmentWorker creates a new background enrichment worker.
func NewEnrichmentWorker(
	coordinator *EnrichmentCoordinator,
	media MediaItemStore,
	tags TagStore,
	logger *slog.Logger,
	cfg EnrichmentWorkerConfig,
) *EnrichmentWorker {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = DefaultBatchSize
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = DefaultMaxRetries
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = DefaultPollInterval
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &EnrichmentWorker{
		coordinator: coordinator,
		media:       media,
		tags:        tags,
		batchSize:   cfg.BatchSize,
		maxRetries:  cfg.MaxRetries,
		interval:    cfg.PollInterval,
		logger:      logger,
	}
}

// Start begins the polling loop. It blocks until ctx is cancelled,
// then returns after the current tick completes. Call this in a goroutine.
func (w *EnrichmentWorker) Start(ctx context.Context) {
	w.logger.Info("enrichment worker started",
		"batch_size", w.batchSize,
		"poll_interval", w.interval,
		"max_retries", w.maxRetries,
	)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("enrichment worker stopping")
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

// tick runs one poll cycle: claim pending items, enrich them, update status.
// Emits a single wide event per batch with all context.
func (w *EnrichmentWorker) tick(ctx context.Context) {
	start := time.Now()

	// Claim pending items with row-level locking
	claimed, err := w.media.ClaimPendingItems(ctx, w.batchSize, w.maxRetries)
	if err != nil {
		w.logger.Error("enrichment_tick",
			"outcome", "claim_failed",
			"error", err,
		)
		return
	}

	if len(claimed) == 0 {
		return
	}

	// Wide event: one log per batch tick
	we := map[string]any{
		"batch_size":     len(claimed),
		"items_enriched": 0,
		"items_failed":   0,
		"items_retried":  0,
		"tags_persisted": 0,
		"tag_errors":     0,
	}

	// Mark items as 'enriching'
	for _, ei := range claimed {
		if err := w.media.UpdateEnrichmentStatus(ctx, ei.ID, "enriching"); err != nil {
			w.logger.Error("enrichment_tick",
				"outcome", "status_update_failed",
				"item_id", ei.ID,
				"error", err,
			)
		}
	}

	// Extract MediaItems for the coordinator
	items := make([]MediaItem, len(claimed))
	for i, ei := range claimed {
		items[i] = ei.Item
	}

	// Run enrichment
	results, stats, err := w.coordinator.Run(ctx, items)
	if err != nil {
		we["outcome"] = "coordinator_failed"
		we["error"] = err.Error()
		we["duration_ms"] = time.Since(start).Milliseconds()
		w.logger.Error("enrichment_tick", wideAttrs(we)...)
		for _, ei := range claimed {
			w.markRetry(ctx, ei, we)
		}
		return
	}

	// Record provider stats
	if stats != nil {
		providerSummary := make(map[string]any, len(stats.Providers))
		for _, ps := range stats.Providers {
			entry := map[string]any{
				"items_received": ps.ItemsReceived,
				"items_enriched": ps.ItemsEnriched,
				"tags_assigned":  ps.TagsAssigned,
			}
			if ps.Failed {
				entry["error"] = ps.Error
			}
			providerSummary[ps.Name] = entry
		}
		we["providers"] = providerSummary

		if stats.LLMItems > 0 {
			we["llm_items"] = stats.LLMItems
			we["llm_tags"] = stats.LLMTags
			we["llm_errors"] = stats.LLMErrors
		}
	}

	// Persist results
	for i, ei := range claimed {
		result := results[i]
		tagCount, tagErrors := w.persistTags(ctx, ei, result)
		we["tags_persisted"] = we["tags_persisted"].(int) + tagCount
		we["tag_errors"] = we["tag_errors"].(int) + tagErrors

		if err := w.media.UpdateEnrichmentStatus(ctx, ei.ID, "enriched"); err != nil {
			w.logger.Error("enrichment_tick",
				"outcome", "status_update_failed",
				"item_id", ei.ID,
				"error", err,
			)
			we["items_failed"] = we["items_failed"].(int) + 1
			continue
		}
		we["items_enriched"] = we["items_enriched"].(int) + 1
	}

	we["outcome"] = "success"
	we["duration_ms"] = time.Since(start).Milliseconds()
	w.logger.Info("enrichment_tick", wideAttrs(we)...)
}

// persistTags saves tags from the enrichment result to the database.
// Returns the count of tags persisted and tag errors.
func (w *EnrichmentWorker) persistTags(ctx context.Context, ei EnrichmentItem, result EnrichmentResult) (persisted int, errors int) {
	// Build a lookup of LLM tags for source/confidence attribution.
	llmTags := make(map[string]float32)
	if result.LLMTagResult != nil {
		for _, ts := range result.LLMTagResult.AllTags() {
			llmTags[ts.Tag] = ts.Confidence
		}
	}

	for _, tag := range result.Item.Tags {
		if !IsValidTag(tag) {
			continue
		}

		tagID, err := w.tags.GetOrCreate(ctx, tag)
		if err != nil {
			errors++
			continue
		}

		source := "api"
		var confidence *float32
		if conf, ok := llmTags[tag]; ok {
			source = "llm"
			confidence = &conf
		}

		if err := w.tags.AddMediaItemTag(ctx, ei.ID, tagID, source, confidence); err != nil {
			errors++
			continue
		}
		persisted++
	}

	return persisted, errors
}

// markRetry increments the retry count or marks as failed if max retries reached.
func (w *EnrichmentWorker) markRetry(ctx context.Context, ei EnrichmentItem, we map[string]any) {
	newRetries := ei.Retries + 1
	status := "pending" // will be retried next tick
	if newRetries >= w.maxRetries {
		status = "failed"
		we["items_failed"] = we["items_failed"].(int) + 1
	} else {
		we["items_retried"] = we["items_retried"].(int) + 1
	}

	if err := w.media.UpdateEnrichmentStatusWithRetries(ctx, ei.ID, status, newRetries); err != nil {
		w.logger.Error("enrichment_tick",
			"outcome", "retry_update_failed",
			"item_id", ei.ID,
			"error", err,
		)
	}
}

// wideAttrs flattens a map to slog key-value pairs.
func wideAttrs(m map[string]any) []any {
	attrs := make([]any, 0, len(m)*2)
	for k, v := range m {
		attrs = append(attrs, k, v)
	}
	return attrs
}
