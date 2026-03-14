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
func (w *EnrichmentWorker) tick(ctx context.Context) {
	// Claim pending items with row-level locking
	claimed, err := w.media.ClaimPendingItems(ctx, w.batchSize, w.maxRetries)
	if err != nil {
		w.logger.Error("failed to claim pending items", "error", err)
		return
	}

	if len(claimed) == 0 {
		return
	}

	w.logger.Info("claimed items for enrichment", "count", len(claimed))

	// Mark items as 'enriching'
	for _, ei := range claimed {
		if err := w.media.UpdateEnrichmentStatus(ctx, ei.ID, "enriching"); err != nil {
			w.logger.Error("failed to mark item as enriching",
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
	results, err := w.coordinator.Run(ctx, items)
	if err != nil {
		w.logger.Error("coordinator run failed", "error", err)
		// Mark all items for retry
		for _, ei := range claimed {
			w.markRetry(ctx, ei)
		}
		return
	}

	// Update status for each item
	for i, ei := range claimed {
		result := results[i]

		// Persist tags from enrichment
		w.persistTags(ctx, ei, result)

		// Mark as enriched
		if err := w.media.UpdateEnrichmentStatus(ctx, ei.ID, "enriched"); err != nil {
			w.logger.Error("failed to mark item as enriched",
				"item_id", ei.ID,
				"error", err,
			)
		}
	}

	w.logger.Info("enrichment batch completed", "count", len(claimed))
}

// persistTags saves tags from the enrichment result to the database.
// LLM-sourced tags are identified via the typed LLMTagResult field;
// all other tags are treated as authoritative API-provider tags.
func (w *EnrichmentWorker) persistTags(ctx context.Context, ei EnrichmentItem, result EnrichmentResult) {
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
			w.logger.Warn("failed to get/create tag",
				"tag", tag,
				"error", err,
			)
			continue
		}

		source := "api"
		var confidence *float32
		if conf, ok := llmTags[tag]; ok {
			source = "llm"
			confidence = &conf
		}

		if err := w.tags.AddMediaItemTag(ctx, ei.ID, tagID, source, confidence); err != nil {
			w.logger.Warn("failed to add tag to item",
				"tag", tag,
				"item_id", ei.ID,
				"error", err,
			)
		}
	}
}

// markRetry increments the retry count or marks as failed if max retries reached.
func (w *EnrichmentWorker) markRetry(ctx context.Context, ei EnrichmentItem) {
	newRetries := ei.Retries + 1
	status := "pending" // will be retried next tick
	if newRetries >= w.maxRetries {
		status = "failed"
		w.logger.Warn("item exceeded max retries, marking as failed",
			"item_id", ei.ID,
			"retries", newRetries,
		)
	}

	if err := w.media.UpdateEnrichmentStatusWithRetries(ctx, ei.ID, status, newRetries); err != nil {
		w.logger.Error("failed to update retry status",
			"item_id", ei.ID,
			"error", err,
		)
	}
}
