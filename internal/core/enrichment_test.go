package core

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNoOpEnricher_ReturnsEmptyResult(t *testing.T) {
	enricher := &NoOpEnricher{}
	item := MediaItem{
		Platform:   "spotify",
		Type:       MediaMusic,
		Title:      "Test Song",
		Creator:    "Test Artist",
		ConsumedAt: time.Now(),
		ExternalID: "test-1",
	}

	result, err := enricher.Enrich(context.Background(), item, []string{"rock", "alternative"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.IsEmpty() {
		t.Errorf("expected empty result, got %+v", result)
	}
}

func TestNoOpEnricher_NilExistingTags(t *testing.T) {
	enricher := &NoOpEnricher{}
	result, err := enricher.Enrich(context.Background(), MediaItem{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsEmpty() {
		t.Errorf("expected empty result")
	}
}

func TestTagResult_IsEmpty(t *testing.T) {
	tests := []struct {
		name   string
		result TagResult
		want   bool
	}{
		{
			name:   "empty result",
			result: TagResult{},
			want:   true,
		},
		{
			name: "has genre",
			result: TagResult{
				Genre: []TagScore{{Tag: "rock", Confidence: 0.9}},
			},
			want: false,
		},
		{
			name: "has topic",
			result: TagResult{
				Topic: []TagScore{{Tag: "technology", Confidence: 0.8}},
			},
			want: false,
		},
		{
			name: "has mood",
			result: TagResult{
				Mood: []TagScore{{Tag: "chill", Confidence: 0.7}},
			},
			want: false,
		},
		{
			name: "has format",
			result: TagResult{
				Format: []TagScore{{Tag: "album-track", Confidence: 0.95}},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.IsEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTagResult_AllTags(t *testing.T) {
	result := TagResult{
		Genre:  []TagScore{{Tag: "rock", Confidence: 0.9}, {Tag: "indie", Confidence: 0.7}},
		Topic:  []TagScore{{Tag: "music-theory", Confidence: 0.6}},
		Mood:   []TagScore{{Tag: "melancholic", Confidence: 0.8}},
		Format: []TagScore{{Tag: "album-track", Confidence: 0.95}},
	}

	all := result.AllTags()
	if len(all) != 5 {
		t.Fatalf("expected 5 tags, got %d", len(all))
	}

	// Verify order: genre, topic, mood, format
	expected := []string{"rock", "indie", "music-theory", "melancholic", "album-track"}
	for i, ts := range all {
		if ts.Tag != expected[i] {
			t.Errorf("tag[%d]: want %q, got %q", i, expected[i], ts.Tag)
		}
	}
}

func TestTagResult_AllTags_Empty(t *testing.T) {
	result := TagResult{}
	all := result.AllTags()
	if len(all) != 0 {
		t.Fatalf("expected 0 tags, got %d", len(all))
	}
}

// --- mockEnrichmentProvider ---

type mockEnrichmentProvider struct {
	name       string
	supportsFn func(mediaType, platform string) bool
	enrichFn   func(ctx context.Context, items []MediaItem) ([]MediaItem, error)
}

func (p *mockEnrichmentProvider) Name() string { return p.name }
func (p *mockEnrichmentProvider) Supports(mediaType, platform string) bool {
	if p.supportsFn != nil {
		return p.supportsFn(mediaType, platform)
	}
	return true
}
func (p *mockEnrichmentProvider) Enrich(ctx context.Context, items []MediaItem) ([]MediaItem, error) {
	if p.enrichFn != nil {
		return p.enrichFn(ctx, items)
	}
	return items, nil
}

var _ EnrichmentProvider = (*mockEnrichmentProvider)(nil)

// --- EnrichmentCoordinator tests ---

func TestCoordinator_EmptyItems(t *testing.T) {
	c := NewEnrichmentCoordinator(nil, nil, discardLogger())
	items, err := c.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if items != nil {
		t.Errorf("expected nil items, got %v", items)
	}
}

func TestCoordinator_NoProviders(t *testing.T) {
	c := NewEnrichmentCoordinator(nil, nil, discardLogger())
	items := []MediaItem{
		{Platform: "spotify", Type: MediaMusic, Title: "Song", ExternalID: "1"},
	}
	result, err := c.Run(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 item, got %d", len(result))
	}
}

func TestCoordinator_SingleProvider_AddsTags(t *testing.T) {
	provider := &mockEnrichmentProvider{
		name: "lastfm",
		enrichFn: func(_ context.Context, items []MediaItem) ([]MediaItem, error) {
			for i := range items {
				items[i].Tags = append(items[i].Tags, "rock", "indie")
			}
			return items, nil
		},
	}

	c := NewEnrichmentCoordinator([]EnrichmentProvider{provider}, nil, discardLogger())
	items := []MediaItem{
		{Platform: "spotify", Type: MediaMusic, Title: "Song", ExternalID: "1"},
	}

	result, err := c.Run(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result[0].Tags) != 2 {
		t.Errorf("expected 2 tags, got %d: %v", len(result[0].Tags), result[0].Tags)
	}
}

func TestCoordinator_ProviderSupportsFilter(t *testing.T) {
	var calledWith []string
	provider := &mockEnrichmentProvider{
		name: "tmdb",
		supportsFn: func(mediaType, platform string) bool {
			return mediaType == "video" // only supports video
		},
		enrichFn: func(_ context.Context, items []MediaItem) ([]MediaItem, error) {
			for _, item := range items {
				calledWith = append(calledWith, item.Title)
			}
			return items, nil
		},
	}

	c := NewEnrichmentCoordinator([]EnrichmentProvider{provider}, nil, discardLogger())
	items := []MediaItem{
		{Platform: "spotify", Type: MediaMusic, Title: "Song", ExternalID: "1"},
		{Platform: "youtube", Type: MediaVideo, Title: "Video", ExternalID: "2"},
	}

	_, err := c.Run(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(calledWith) != 1 {
		t.Fatalf("expected provider called with 1 item, got %d", len(calledWith))
	}
	if calledWith[0] != "Video" {
		t.Errorf("expected 'Video', got %q", calledWith[0])
	}
}

func TestCoordinator_MultipleProviders_MergeTags(t *testing.T) {
	p1 := &mockEnrichmentProvider{
		name: "lastfm",
		enrichFn: func(_ context.Context, items []MediaItem) ([]MediaItem, error) {
			for i := range items {
				items[i].Tags = []string{"rock"}
			}
			return items, nil
		},
	}
	p2 := &mockEnrichmentProvider{
		name: "musicbrainz",
		enrichFn: func(_ context.Context, items []MediaItem) ([]MediaItem, error) {
			for i := range items {
				items[i].Tags = []string{"rock", "alternative"} // rock is a duplicate
			}
			return items, nil
		},
	}

	c := NewEnrichmentCoordinator([]EnrichmentProvider{p1, p2}, nil, discardLogger())
	items := []MediaItem{
		{Platform: "spotify", Type: MediaMusic, Title: "Song", ExternalID: "1"},
	}

	result, err := c.Run(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have "rock" and "alternative" (deduplicated)
	tags := result[0].Tags
	if len(tags) != 2 {
		t.Errorf("expected 2 tags (deduplicated), got %d: %v", len(tags), tags)
	}
}

func TestCoordinator_ProviderError_NonFatal(t *testing.T) {
	p1 := &mockEnrichmentProvider{
		name: "failing",
		enrichFn: func(_ context.Context, _ []MediaItem) ([]MediaItem, error) {
			return nil, errors.New("API down")
		},
	}
	p2 := &mockEnrichmentProvider{
		name: "working",
		enrichFn: func(_ context.Context, items []MediaItem) ([]MediaItem, error) {
			for i := range items {
				items[i].Tags = []string{"rock"}
			}
			return items, nil
		},
	}

	c := NewEnrichmentCoordinator([]EnrichmentProvider{p1, p2}, nil, discardLogger())
	items := []MediaItem{
		{Platform: "spotify", Type: MediaMusic, Title: "Song", ExternalID: "1"},
	}

	result, err := c.Run(context.Background(), items)
	if err != nil {
		t.Fatalf("expected no error (failures are non-fatal), got: %v", err)
	}

	// Should still have tags from the working provider
	if len(result[0].Tags) != 1 || result[0].Tags[0] != "rock" {
		t.Errorf("expected ['rock'], got %v", result[0].Tags)
	}
}

func TestCoordinator_LLMPhase2_RunsAfterAPI(t *testing.T) {
	var llmReceivedTags []string

	provider := &mockEnrichmentProvider{
		name: "lastfm",
		enrichFn: func(_ context.Context, items []MediaItem) ([]MediaItem, error) {
			for i := range items {
				items[i].Tags = []string{"rock"}
			}
			return items, nil
		},
	}

	llm := &mockEnricher{
		enrichFn: func(_ context.Context, item MediaItem, existingTags []string) (*TagResult, error) {
			llmReceivedTags = existingTags
			return &TagResult{
				Mood: []TagScore{{Tag: "melancholic", Confidence: 0.8}},
			}, nil
		},
	}

	c := NewEnrichmentCoordinator([]EnrichmentProvider{provider}, llm, discardLogger())
	items := []MediaItem{
		{Platform: "spotify", Type: MediaMusic, Title: "Song", ExternalID: "1"},
	}

	result, err := c.Run(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// LLM should have received the "rock" tag from Phase 1
	if len(llmReceivedTags) != 1 || llmReceivedTags[0] != "rock" {
		t.Errorf("LLM should receive Phase 1 tags, got: %v", llmReceivedTags)
	}

	// Result should have both API and LLM tags
	tags := result[0].Tags
	if len(tags) != 2 {
		t.Errorf("expected 2 tags (rock + melancholic), got %d: %v", len(tags), tags)
	}
}

func TestCoordinator_LLMPhase2_InvalidTagsFiltered(t *testing.T) {
	llm := &mockEnricher{
		enrichFn: func(_ context.Context, _ MediaItem, _ []string) (*TagResult, error) {
			return &TagResult{
				Genre: []TagScore{
					{Tag: "rock", Confidence: 0.9},     // valid
					{Tag: "fake-tag", Confidence: 0.5}, // invalid
				},
			}, nil
		},
	}

	c := NewEnrichmentCoordinator(nil, llm, discardLogger())
	items := []MediaItem{
		{Platform: "spotify", Type: MediaMusic, Title: "Song", ExternalID: "1"},
	}

	result, err := c.Run(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only valid tags should pass
	tags := result[0].Tags
	if len(tags) != 1 || tags[0] != "rock" {
		t.Errorf("expected ['rock'], got %v", tags)
	}
}

func TestCoordinator_NilLLM_SkipsPhase2(t *testing.T) {
	c := NewEnrichmentCoordinator(nil, nil, discardLogger())
	items := []MediaItem{
		{Platform: "spotify", Type: MediaMusic, Title: "Song", ExternalID: "1", Tags: []string{"rock"}},
	}

	result, err := c.Run(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Tags unchanged
	if len(result[0].Tags) != 1 || result[0].Tags[0] != "rock" {
		t.Errorf("expected ['rock'], got %v", result[0].Tags)
	}
}

// --- mergeUniqueTags tests ---

func TestMergeUniqueTags(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		new      []string
		want     int
	}{
		{"both empty", nil, nil, 0},
		{"new only", nil, []string{"a", "b"}, 2},
		{"existing only", []string{"a"}, nil, 1},
		{"no overlap", []string{"a"}, []string{"b"}, 2},
		{"full overlap", []string{"a", "b"}, []string{"a", "b"}, 2},
		{"partial overlap", []string{"a", "b"}, []string{"b", "c"}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeUniqueTags(tt.existing, tt.new)
			if len(got) != tt.want {
				t.Errorf("mergeUniqueTags() = %v (len %d), want len %d", got, len(got), tt.want)
			}
		})
	}
}

// --- EnrichmentError tests ---

func TestEnrichmentError(t *testing.T) {
	err := &EnrichmentError{
		ItemTitle: "Song A",
		Provider:  "lastfm",
		Err:       errors.New("api timeout"),
	}

	msg := err.Error()
	if msg == "" {
		t.Error("expected non-empty error message")
	}
	if err.Unwrap() == nil {
		t.Error("expected non-nil unwrap")
	}
}
