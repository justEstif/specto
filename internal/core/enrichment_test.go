package core

import (
	"context"
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
