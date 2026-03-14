package enrichment

import (
	"testing"

	"github.com/justestif/specto/internal/core"
)

func TestConvertTagScores(t *testing.T) {
	tests := []struct {
		name   string
		input  []ClassifyTagScore
		want   int
		checks func(t *testing.T, got []core.TagScore)
	}{
		{
			name:  "nil input",
			input: nil,
			want:  0,
		},
		{
			name:  "empty input",
			input: []ClassifyTagScore{},
			want:  0,
		},
		{
			name: "normal scores",
			input: []ClassifyTagScore{
				{Tag: "rock", Confidence: 0.9},
				{Tag: "pop", Confidence: 0.7},
			},
			want: 2,
			checks: func(t *testing.T, got []core.TagScore) {
				if got[0].Tag != "rock" || got[0].Confidence != 0.9 {
					t.Errorf("got %+v, want rock/0.9", got[0])
				}
				if got[1].Tag != "pop" || got[1].Confidence != 0.7 {
					t.Errorf("got %+v, want pop/0.7", got[1])
				}
			},
		},
		{
			name: "zero confidence gets default",
			input: []ClassifyTagScore{
				{Tag: "jazz", Confidence: 0},
			},
			want: 1,
			checks: func(t *testing.T, got []core.TagScore) {
				if got[0].Confidence != DefaultConfidence {
					t.Errorf("got confidence %f, want %f", got[0].Confidence, DefaultConfidence)
				}
			},
		},
		{
			name: "negative confidence gets default",
			input: []ClassifyTagScore{
				{Tag: "blues", Confidence: -0.5},
			},
			want: 1,
			checks: func(t *testing.T, got []core.TagScore) {
				if got[0].Confidence != DefaultConfidence {
					t.Errorf("got confidence %f, want %f", got[0].Confidence, DefaultConfidence)
				}
			},
		},
		{
			name: "confidence clamped to 1.0",
			input: []ClassifyTagScore{
				{Tag: "metal", Confidence: 1.5},
			},
			want: 1,
			checks: func(t *testing.T, got []core.TagScore) {
				if got[0].Confidence != 1.0 {
					t.Errorf("got confidence %f, want 1.0", got[0].Confidence)
				}
			},
		},
		{
			name: "empty tag skipped",
			input: []ClassifyTagScore{
				{Tag: "", Confidence: 0.9},
				{Tag: "rock", Confidence: 0.8},
			},
			want: 1,
			checks: func(t *testing.T, got []core.TagScore) {
				if got[0].Tag != "rock" {
					t.Errorf("got tag %q, want rock", got[0].Tag)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertTagScores(tt.input)
			if len(got) != tt.want {
				t.Fatalf("len = %d, want %d", len(got), tt.want)
			}
			if tt.checks != nil {
				tt.checks(t, got)
			}
		})
	}
}

func TestClassifyOutputToTagResult(t *testing.T) {
	output := &ClassifyOutput{
		Genre: []ClassifyTagScore{
			{Tag: "rock", Confidence: 0.95},
			{Tag: "alternative", Confidence: 0.8},
		},
		Topic: []ClassifyTagScore{
			{Tag: "nature", Confidence: 0.7},
		},
		Mood: []ClassifyTagScore{
			{Tag: "energetic", Confidence: 0.85},
			{Tag: "uplifting", Confidence: 0.6},
		},
		Format: []ClassifyTagScore{
			{Tag: "album-track", Confidence: 0.9},
		},
	}

	result := &core.TagResult{
		Genre:  convertTagScores(output.Genre),
		Topic:  convertTagScores(output.Topic),
		Mood:   convertTagScores(output.Mood),
		Format: convertTagScores(output.Format),
	}

	if len(result.Genre) != 2 {
		t.Fatalf("genre count = %d, want 2", len(result.Genre))
	}
	if len(result.Topic) != 1 {
		t.Fatalf("topic count = %d, want 1", len(result.Topic))
	}
	if len(result.Mood) != 2 {
		t.Fatalf("mood count = %d, want 2", len(result.Mood))
	}
	if len(result.Format) != 1 {
		t.Fatalf("format count = %d, want 1", len(result.Format))
	}

	// Verify TagResult methods work
	if result.IsEmpty() {
		t.Error("result should not be empty")
	}
	all := result.AllTags()
	if len(all) != 6 {
		t.Errorf("AllTags() count = %d, want 6", len(all))
	}
}

func TestClassifyOutputEmpty(t *testing.T) {
	output := &ClassifyOutput{
		Genre:  nil,
		Topic:  []ClassifyTagScore{},
		Mood:   nil,
		Format: nil,
	}

	result := &core.TagResult{
		Genre:  convertTagScores(output.Genre),
		Topic:  convertTagScores(output.Topic),
		Mood:   convertTagScores(output.Mood),
		Format: convertTagScores(output.Format),
	}

	if !result.IsEmpty() {
		t.Error("result should be empty")
	}
}

func TestDefaultConfidenceValue(t *testing.T) {
	if DefaultConfidence != 0.8 {
		t.Errorf("DefaultConfidence = %f, want 0.8", DefaultConfidence)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name:    "empty provider",
			cfg:     Config{Model: "gemini-2.5-flash"},
			wantErr: "LLM_PROVIDER is required",
		},
		{
			name:    "empty model",
			cfg:     Config{Provider: "googlegenai"},
			wantErr: "LLM_MODEL is required",
		},
		{
			name:    "unsupported provider",
			cfg:     Config{Provider: "anthropic", Model: "claude-3"},
			wantErr: "unsupported LLM_PROVIDER",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// New() will fail for these configs without needing a real API key
			// since validation happens before Genkit init
			_, err := New(t.Context(), tt.cfg, nil)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
