package core

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Enricher is the interface for the core LLM-based enrichment pipeline.
// It classifies media items by assigning tags from the fixed tag set.
//
// Plugin enrichment (platform-specific) is handled by SourcePlugin.Enrich().
// This interface is for the universal core enricher that runs after plugin
// enrichment to fill gaps (mood, topic, etc.).
//
// The actual LLM implementation lives in the Enrichment/LLM epic.
// For bootstrapping and testing, use NoOpEnricher.
type Enricher interface {
	// Enrich classifies a media item, returning tag assignments with
	// confidence scores. The existingTags parameter contains tags already
	// assigned by plugin enrichment (used as context for the LLM).
	Enrich(ctx context.Context, item MediaItem, existingTags []string) (*TagResult, error)
}

// TagResult holds the tags assigned to a media item by the enricher.
// Each category contains zero or more tags from the fixed tag set,
// paired with a confidence score (0.0-1.0).
type TagResult struct {
	Genre  []TagScore `json:"genre"`
	Topic  []TagScore `json:"topic"`
	Mood   []TagScore `json:"mood"`
	Format []TagScore `json:"format"`
}

// TagScore pairs a tag name with a confidence score.
type TagScore struct {
	Tag        string  `json:"tag"`
	Confidence float32 `json:"confidence"`
}

// IsEmpty returns true if the TagResult has no tags in any category.
func (tr *TagResult) IsEmpty() bool {
	return len(tr.Genre) == 0 && len(tr.Topic) == 0 && len(tr.Mood) == 0 && len(tr.Format) == 0
}

// AllTags returns a flat slice of all tag scores across all categories.
func (tr *TagResult) AllTags() []TagScore {
	all := make([]TagScore, 0, len(tr.Genre)+len(tr.Topic)+len(tr.Mood)+len(tr.Format))
	all = append(all, tr.Genre...)
	all = append(all, tr.Topic...)
	all = append(all, tr.Mood...)
	all = append(all, tr.Format...)
	return all
}

// NoOpEnricher is an Enricher that returns empty results.
// Used for testing and bootstrapping before the LLM enricher is implemented.
type NoOpEnricher struct{}

// Enrich returns an empty TagResult with no error.
func (n *NoOpEnricher) Enrich(_ context.Context, _ MediaItem, _ []string) (*TagResult, error) {
	return &TagResult{}, nil
}

// Compile-time assertion that NoOpEnricher implements Enricher.
var _ Enricher = (*NoOpEnricher)(nil)

// EnrichmentCoordinator orchestrates two-phase enrichment:
//  1. Phase 1: Run all API providers concurrently on items they support
//  2. Phase 2: Run the LLM enricher last, using Phase 1 tags as context
//
// Each provider only receives items where Supports() returns true.
// Per-item errors are logged and do not abort the batch.
type EnrichmentCoordinator struct {
	providers []EnrichmentProvider
	enricher  Enricher // LLM enricher (runs last); nil to skip Phase 2
	logger    *slog.Logger
}

// NewEnrichmentCoordinator creates a coordinator with the given API
// providers and an optional LLM enricher. If enricher is nil, Phase 2
// is skipped.
func NewEnrichmentCoordinator(
	providers []EnrichmentProvider,
	enricher Enricher,
	logger *slog.Logger,
) *EnrichmentCoordinator {
	if logger == nil {
		logger = slog.Default()
	}
	return &EnrichmentCoordinator{
		providers: providers,
		enricher:  enricher,
		logger:    logger,
	}
}

// Run executes two-phase enrichment on a batch of items.
//
// Phase 1: All API providers run concurrently. Each provider receives only
// the items it supports. Results (tags) are merged back onto the items.
//
// Phase 2: The LLM enricher runs on all items, using accumulated tags
// from Phase 1 as context.
//
// Returns the enriched items with merged tags. Per-provider and per-item
// errors are logged but do not fail the batch.
func (c *EnrichmentCoordinator) Run(ctx context.Context, items []MediaItem) ([]MediaItem, error) {
	if len(items) == 0 {
		return items, nil
	}

	// Phase 1: API providers (concurrent)
	items = c.runAPIProviders(ctx, items)

	// Phase 2: LLM enricher (sequential, uses Phase 1 tags as context)
	items = c.runLLMEnricher(ctx, items)

	return items, nil
}

// runAPIProviders runs all API providers concurrently and merges their
// tag results back onto items.
func (c *EnrichmentCoordinator) runAPIProviders(ctx context.Context, items []MediaItem) []MediaItem {
	if len(c.providers) == 0 {
		return items
	}

	// providerResult holds the output of a single provider run.
	type providerResult struct {
		name  string
		items []MediaItem
		err   error
	}

	var wg sync.WaitGroup
	results := make([]providerResult, len(c.providers))

	for i, p := range c.providers {
		// Filter items this provider supports
		var supported []MediaItem
		for _, item := range items {
			if p.Supports(string(item.Type), item.Platform) {
				supported = append(supported, item)
			}
		}

		if len(supported) == 0 {
			results[i] = providerResult{name: p.Name()}
			continue
		}

		wg.Add(1)
		go func(idx int, provider EnrichmentProvider, batch []MediaItem) {
			defer wg.Done()
			enriched, err := provider.Enrich(ctx, batch)
			results[idx] = providerResult{
				name:  provider.Name(),
				items: enriched,
				err:   err,
			}
		}(i, p, supported)
	}

	wg.Wait()

	// Merge tags from all provider results back onto the original items.
	// Build a lookup by ExternalID for efficient merging.
	for _, pr := range results {
		if pr.err != nil {
			c.logger.Warn("enrichment provider failed",
				"provider", pr.name,
				"error", pr.err,
			)
			continue
		}

		if len(pr.items) == 0 {
			continue
		}

		// Build a map of enriched tags by ExternalID
		enrichedTags := make(map[string][]string, len(pr.items))
		for _, enrichedItem := range pr.items {
			enrichedTags[enrichedItem.ExternalID] = enrichedItem.Tags
		}

		// Merge tags into original items
		for i := range items {
			if tags, ok := enrichedTags[items[i].ExternalID]; ok {
				items[i].Tags = mergeUniqueTags(items[i].Tags, tags)
			}
		}
	}

	return items
}

// runLLMEnricher runs the LLM enricher (Phase 2) on all items.
func (c *EnrichmentCoordinator) runLLMEnricher(ctx context.Context, items []MediaItem) []MediaItem {
	if c.enricher == nil {
		return items
	}

	for i, item := range items {
		tagResult, err := c.enricher.Enrich(ctx, item, item.Tags)
		if err != nil {
			c.logger.Warn("LLM enrichment failed for item",
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

		// Merge LLM tags into the item's tag list
		var llmTags []string
		for _, ts := range validated.AllTags() {
			llmTags = append(llmTags, ts.Tag)
		}
		items[i].Tags = mergeUniqueTags(items[i].Tags, llmTags)

		// Store the validated tag result in raw metadata for later persistence
		if items[i].RawMetadata == nil {
			items[i].RawMetadata = make(map[string]any)
		}
		items[i].RawMetadata["_llm_tag_result"] = validated
	}

	return items
}

// mergeUniqueTags merges two tag slices, deduplicating entries.
func mergeUniqueTags(existing, new []string) []string {
	seen := make(map[string]struct{}, len(existing)+len(new))
	for _, t := range existing {
		seen[t] = struct{}{}
	}

	merged := make([]string, len(existing))
	copy(merged, existing)

	for _, t := range new {
		if _, ok := seen[t]; !ok {
			merged = append(merged, t)
			seen[t] = struct{}{}
		}
	}
	return merged
}

// EnrichmentError wraps a per-item enrichment failure for logging.
type EnrichmentError struct {
	ItemTitle string
	Provider  string
	Err       error
}

func (e *EnrichmentError) Error() string {
	return fmt.Sprintf("enrichment failed for %q (provider: %s): %v", e.ItemTitle, e.Provider, e.Err)
}

func (e *EnrichmentError) Unwrap() error {
	return e.Err
}
