---
# project-media-consumption-analysis-2vk5
title: Implement Genkit LLM enricher
status: completed
type: feature
priority: high
created_at: 2026-03-14T03:52:54Z
updated_at: 2026-03-14T18:26:29Z
parent: project-media-consumption-analysis-eo0f
blocked_by:
    - project-media-consumption-analysis-2rhd
---

Replace NoOpEnricher with a real Genkit-based LLM enricher implementation.

## Tasks

- [x] Add Genkit Go dependency (github.com/firebase/genkit/go + googlegenai plugin)
- [x] Create prompts/classify.prompt (Dotprompt with Handlebars + Picoschema)
- [x] Implement GenkitEnricher struct (Enricher interface) using LookupDataPrompt
- [x] Define ClassifyInput/ClassifyOutput Go structs matching prompt schema
- [x] Add LLM config to app.Config (Provider, Model, APIKey, OllamaBaseURL)
- [x] Wire GenkitEnricher into EnrichmentCoordinator in app.go (replace nil)
- [x] Read LLM env vars in main.go and pass to app.Config
- [x] Validate returned tags against fixed tag set (already done in coordinator)
- [x] Default confidence 0.8 for LLM tags
- [x] Wire into SyncService replacing NoOpEnricher
- [x] Tests with mock Enricher (no real LLM calls)

## Reference

See docs/enrichment.md — Core LLM Enricher section for prompt design, structured output shape, and provider config.

## Summary of Changes (Infrastructure Phase)\n\nBuilt the enrichment infrastructure foundation:\n- Added `EnrichmentProvider` interface to `internal/core/plugin.go`\n- Created `EnrichmentCoordinator` with two-phase execution (API providers concurrent, then LLM) in `internal/core/enrichment.go`\n- Created `EnrichmentWorker` with `SELECT ... FOR UPDATE SKIP LOCKED` polling in `internal/core/worker.go`\n- Added DB migration for `enrichment_retries` column\n- Added `ClaimPendingItems` and `UpdateEnrichmentStatusWithRetries` SQL queries\n- Removed inline enrichment from `SyncService` — enrichment is now fully async\n- Wired coordinator + worker into `app.go` and `main.go` with graceful shutdown\n- 88+ core tests passing\n\nRemaining tasks (Genkit LLM-specific) still need implementation.

## Summary of Changes (Genkit LLM Enricher)

Implemented the Genkit-based LLM enricher that completes the two-phase enrichment pipeline:

### New files:
- `internal/enrichment/genkit.go` — GenkitEnricher struct implementing core.Enricher, uses Dotprompt for structured classification
- `internal/enrichment/prompts/classify.prompt` — Dotprompt file with Handlebars template, Picoschema output, multi-role (system+user)
- `internal/enrichment/genkit_test.go` — 11 tests (convertTagScores edge cases, config validation, output conversion)

### Modified files:
- `internal/app/app.go` — Added `LLMEnricher core.Enricher` to Config, wired into EnrichmentCoordinator (replaces nil)
- `cmd/web/main.go` — Reads LLM_PROVIDER/LLM_MODEL/LLM_API_KEY env vars, initializes GenkitEnricher when configured
- `go.mod`/`go.sum` — Added github.com/firebase/genkit/go + googlegenai plugin dependencies

### Architecture:
- Genkit initialized with embedded prompt FS (`//go:embed prompts/*`)
- Uses `genkit.LookupDataPrompt[ClassifyInput, *ClassifyOutput]` for type-safe structured output
- Prompt passes full allowed tag lists as Handlebars context — LLM sees exact valid tags
- Confidence defaults to 0.8, clamped to [0,1], empty tags dropped
- Coordinator's existing ValidateTagResult() provides second-pass validation against fixed tag set
- Provider is optional — if LLM_PROVIDER not set, Phase 2 is skipped (nil enricher)
