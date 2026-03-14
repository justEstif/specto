// Package enrichment implements the LLM-based enrichment pipeline using
// Firebase Genkit. It provides a GenkitEnricher that implements core.Enricher
// and uses Dotprompt for structured tag classification.
package enrichment

import (
	"context"
	"embed"
	"fmt"
	"log/slog"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	oaioption "github.com/openai/openai-go/option"

	"github.com/justestif/specto/internal/core"
)

//go:embed prompts/*
var promptsFS embed.FS

// Config holds the LLM enricher configuration.
type Config struct {
	Provider string // "googlegenai" or "openai"
	Model    string // e.g. "gemini-2.5-flash", "gpt-4o-mini"
	APIKey   string // required for all providers
	BaseURL  string // optional: custom base URL for openai provider (e.g. Ollama)
}

// ClassifyInput is the input schema for the classification prompt.
type ClassifyInput struct {
	Title        string   `json:"title"`
	Creator      string   `json:"creator,omitempty"`
	MediaType    string   `json:"mediaType"`
	Platform     string   `json:"platform"`
	ExistingTags []string `json:"existingTags,omitempty"`
	GenreTags    []string `json:"genreTags"`
	TopicTags    []string `json:"topicTags"`
	MoodTags     []string `json:"moodTags"`
	FormatTags   []string `json:"formatTags"`
}

// ClassifyOutput is the output schema for the classification prompt.
// Mirrors core.TagResult but uses its own type for Genkit schema generation.
type ClassifyOutput struct {
	Genre  []ClassifyTagScore `json:"genre"`
	Topic  []ClassifyTagScore `json:"topic"`
	Mood   []ClassifyTagScore `json:"mood"`
	Format []ClassifyTagScore `json:"format"`
}

// ClassifyTagScore pairs a tag with a confidence score.
type ClassifyTagScore struct {
	Tag        string  `json:"tag"`
	Confidence float64 `json:"confidence"`
}

// GenkitEnricher implements core.Enricher using Firebase Genkit with Dotprompt.
type GenkitEnricher struct {
	g         *genkit.Genkit
	prompt    *ai.DataPrompt[ClassifyInput, *ClassifyOutput]
	modelName string // fully qualified model name (e.g. "googleai/gemini-2.5-flash", "openai/gpt-4o-mini")
	logger    *slog.Logger
}

// Compile-time interface check.
var _ core.Enricher = (*GenkitEnricher)(nil)

// New creates and initializes a GenkitEnricher. It initializes the Genkit
// runtime with the configured provider plugin and loads the classify prompt.
func New(ctx context.Context, cfg Config, logger *slog.Logger) (*GenkitEnricher, error) {
	if logger == nil {
		logger = slog.Default()
	}

	if cfg.Provider == "" {
		return nil, fmt.Errorf("enrichment: LLM_PROVIDER is required")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("enrichment: LLM_MODEL is required")
	}

	var opts []genkit.GenkitOption
	var modelName string

	switch cfg.Provider {
	case "googlegenai":
		opts = append(opts, genkit.WithPlugins(&googlegenai.GoogleAI{}))
		modelName = "googleai/" + cfg.Model
	case "openai":
		plugin := &openai.OpenAI{
			APIKey: cfg.APIKey,
		}
		if cfg.BaseURL != "" {
			plugin.Opts = append(plugin.Opts, oaioption.WithBaseURL(cfg.BaseURL))
		}
		opts = append(opts, genkit.WithPlugins(plugin))
		modelName = "openai/" + cfg.Model
	default:
		return nil, fmt.Errorf("enrichment: unsupported LLM_PROVIDER %q (supported: googlegenai, openai)", cfg.Provider)
	}

	opts = append(opts,
		genkit.WithPromptFS(promptsFS),
	)

	g := genkit.Init(ctx, opts...)

	// Register schemas so the prompt can reference them
	genkit.DefineSchemaFor[ClassifyInput](g)
	genkit.DefineSchemaFor[ClassifyOutput](g)

	prompt := genkit.LookupDataPrompt[ClassifyInput, *ClassifyOutput](g, "classify")
	if prompt == nil {
		return nil, fmt.Errorf("enrichment: classify prompt not found (check prompts/classify.prompt)")
	}

	logger.Info("genkit enricher initialized",
		"provider", cfg.Provider,
		"model", modelName,
	)

	return &GenkitEnricher{
		g:         g,
		prompt:    prompt,
		modelName: modelName,
		logger:    logger,
	}, nil
}

// DefaultConfidence is the confidence score assigned to LLM-generated tags.
const DefaultConfidence = 0.8

// Enrich classifies a media item using the LLM and returns tag assignments.
// existingTags are passed as context to help the LLM make better decisions.
func (e *GenkitEnricher) Enrich(ctx context.Context, item core.MediaItem, existingTags []string) (*core.TagResult, error) {
	input := ClassifyInput{
		Title:        item.Title,
		Creator:      item.Creator,
		MediaType:    string(item.Type),
		Platform:     item.Platform,
		ExistingTags: existingTags,
		GenreTags:    core.GenreTags,
		TopicTags:    core.TopicTags,
		MoodTags:     core.MoodTags,
		FormatTags:   core.FormatTags,
	}

	output, _, err := e.prompt.Execute(ctx, input, ai.WithModelName(e.modelName))
	if err != nil {
		return nil, fmt.Errorf("enrichment: LLM classification failed: %w", err)
	}

	if output == nil {
		return &core.TagResult{}, nil
	}

	// Convert to core.TagResult with default confidence
	result := &core.TagResult{
		Genre:  convertTagScores(output.Genre),
		Topic:  convertTagScores(output.Topic),
		Mood:   convertTagScores(output.Mood),
		Format: convertTagScores(output.Format),
	}

	return result, nil
}

// convertTagScores converts ClassifyTagScore slices to core.TagScore slices.
// Tags with zero or negative confidence are dropped. Confidence is clamped
// to [0.0, 1.0] and defaults to DefaultConfidence if not provided.
func convertTagScores(scores []ClassifyTagScore) []core.TagScore {
	if len(scores) == 0 {
		return nil
	}

	result := make([]core.TagScore, 0, len(scores))
	for _, s := range scores {
		if s.Tag == "" {
			continue
		}
		conf := float32(s.Confidence)
		if conf <= 0 {
			conf = DefaultConfidence
		}
		if conf > 1.0 {
			conf = 1.0
		}
		result = append(result, core.TagScore{
			Tag:        s.Tag,
			Confidence: conf,
		})
	}
	return result
}
