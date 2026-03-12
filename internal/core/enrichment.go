package core

import "context"

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
