package core

import "testing"

func TestIsValidTag(t *testing.T) {
	tests := []struct {
		tag  string
		want bool
	}{
		{"rock", true},
		{"hip-hop", true},
		{"electronic", true},
		{"melancholic", true},
		{"album-track", true},
		{"technology", true},
		{"podcast-episode", true},
		{"vaporwave", false},
		{"", false},
		{"ROCK", false}, // case-sensitive
		{"Rock", false},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			if got := IsValidTag(tt.tag); got != tt.want {
				t.Errorf("IsValidTag(%q) = %v, want %v", tt.tag, got, tt.want)
			}
		})
	}
}

func TestTagCategoryOf(t *testing.T) {
	tests := []struct {
		tag  string
		want TagCategory
	}{
		{"rock", TagCategoryGenre},
		{"comedy", TagCategoryGenre},
		{"technology", TagCategoryTopic},
		{"ai", TagCategoryTopic},
		{"melancholic", TagCategoryMood},
		{"chill", TagCategoryMood},
		{"album-track", TagCategoryFormat},
		{"podcast-episode", TagCategoryFormat},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			if got := TagCategoryOf(tt.tag); got != tt.want {
				t.Errorf("TagCategoryOf(%q) = %q, want %q", tt.tag, got, tt.want)
			}
		})
	}
}

func TestValidateTagResult(t *testing.T) {
	input := &TagResult{
		Genre: []TagScore{
			{Tag: "rock", Confidence: 0.9},
			{Tag: "vaporwave", Confidence: 0.7},  // invalid — should be dropped
			{Tag: "technology", Confidence: 0.6}, // wrong category — should be dropped
		},
		Topic: []TagScore{
			{Tag: "technology", Confidence: 0.8},
			{Tag: "rock", Confidence: 0.5}, // wrong category
		},
		Mood: []TagScore{
			{Tag: "chill", Confidence: 0.85},
			{Tag: "fake-mood", Confidence: 0.3}, // invalid
		},
		Format: []TagScore{
			{Tag: "album-track", Confidence: 0.95},
		},
	}

	result := ValidateTagResult(input)

	if len(result.Genre) != 1 || result.Genre[0].Tag != "rock" {
		t.Errorf("Genre: want [rock], got %v", result.Genre)
	}
	if len(result.Topic) != 1 || result.Topic[0].Tag != "technology" {
		t.Errorf("Topic: want [technology], got %v", result.Topic)
	}
	if len(result.Mood) != 1 || result.Mood[0].Tag != "chill" {
		t.Errorf("Mood: want [chill], got %v", result.Mood)
	}
	if len(result.Format) != 1 || result.Format[0].Tag != "album-track" {
		t.Errorf("Format: want [album-track], got %v", result.Format)
	}
}

func TestValidateTagResult_Empty(t *testing.T) {
	result := ValidateTagResult(&TagResult{})
	if !result.IsEmpty() {
		t.Errorf("expected empty result after validating empty input")
	}
}

func TestValidateTagResult_AllInvalid(t *testing.T) {
	input := &TagResult{
		Genre: []TagScore{
			{Tag: "made-up-genre", Confidence: 0.9},
		},
		Topic: []TagScore{
			{Tag: "bogus-topic", Confidence: 0.8},
		},
	}
	result := ValidateTagResult(input)
	if !result.IsEmpty() {
		t.Errorf("expected empty result when all tags are invalid, got %+v", result)
	}
}

func TestAllFixedTags_Immutability(t *testing.T) {
	tags1 := AllFixedTags()
	tags1["injected"] = "hack"

	tags2 := AllFixedTags()
	if _, ok := tags2["injected"]; ok {
		t.Error("AllFixedTags should return an independent copy")
	}
}

func TestTagsByCategory(t *testing.T) {
	genres := TagsByCategory(TagCategoryGenre)
	if len(genres) == 0 {
		t.Error("expected non-empty genre tags")
	}
	if genres[0] != "rock" {
		t.Errorf("first genre tag: want 'rock', got %q", genres[0])
	}

	topics := TagsByCategory(TagCategoryTopic)
	if len(topics) == 0 {
		t.Error("expected non-empty topic tags")
	}

	moods := TagsByCategory(TagCategoryMood)
	if len(moods) == 0 {
		t.Error("expected non-empty mood tags")
	}

	formats := TagsByCategory(TagCategoryFormat)
	if len(formats) == 0 {
		t.Error("expected non-empty format tags")
	}

	unknown := TagsByCategory("nonexistent")
	if unknown != nil {
		t.Errorf("expected nil for unknown category, got %v", unknown)
	}
}

func TestAllFixedTags_Completeness(t *testing.T) {
	all := AllFixedTags()
	expectedCount := len(GenreTags) + len(TopicTags) + len(MoodTags) + len(FormatTags)
	if len(all) != expectedCount {
		t.Errorf("AllFixedTags count: want %d, got %d", expectedCount, len(all))
	}
}

func TestTagCounts(t *testing.T) {
	// Verify the counts match the enrichment doc.
	if len(GenreTags) != 45 {
		t.Errorf("GenreTags count: want 45, got %d", len(GenreTags))
	}
	if len(TopicTags) != 41 {
		t.Errorf("TopicTags count: want 41, got %d", len(TopicTags))
	}
	if len(MoodTags) != 20 {
		t.Errorf("MoodTags count: want 20, got %d", len(MoodTags))
	}
	if len(FormatTags) != 24 {
		t.Errorf("FormatTags count: want 24, got %d", len(FormatTags))
	}
}

func TestNoDuplicateTags(t *testing.T) {
	seen := make(map[string]bool)
	for _, tag := range GenreTags {
		if seen[tag] {
			t.Errorf("duplicate genre tag: %q", tag)
		}
		seen[tag] = true
	}
	for _, tag := range TopicTags {
		if seen[tag] {
			t.Errorf("duplicate topic tag (or cross-category collision): %q", tag)
		}
		seen[tag] = true
	}
	for _, tag := range MoodTags {
		if seen[tag] {
			t.Errorf("duplicate mood tag (or cross-category collision): %q", tag)
		}
		seen[tag] = true
	}
	for _, tag := range FormatTags {
		if seen[tag] {
			t.Errorf("duplicate format tag (or cross-category collision): %q", tag)
		}
		seen[tag] = true
	}
}
