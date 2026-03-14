---
# project-media-consumption-analysis-2vk5
title: Implement Genkit LLM enricher
status: in-progress
type: feature
priority: high
created_at: 2026-03-14T03:52:54Z
updated_at: 2026-03-14T15:17:33Z
parent: project-media-consumption-analysis-eo0f
blocked_by:
    - project-media-consumption-analysis-2rhd
---

Replace NoOpEnricher with a real Genkit-based LLM enricher implementation.

## Tasks

- [ ] Add Genkit dependency and provider plugin (googlegenai + ollama)
- [ ] Implement Enricher interface with Genkit GenerateData[TagResult]
- [ ] Build classification prompt (item metadata + existing tags → genre/topic/mood/format)
- [ ] Validate returned tags against fixed tag set, drop hallucinated tags
- [ ] Resolve tag aliases before persisting
- [ ] Store tags with confidence scores (default 0.8 for LLM tags)
- [ ] Support batch classification (multiple items per prompt)
- [ ] Add provider configuration (provider, model, API key, batch_size, max_concurrent)
- [x] Wire into SyncService replacing NoOpEnricher
- [ ] Tests with mock LLM responses

## Reference

See docs/enrichment.md — Core LLM Enricher section for prompt design, structured output shape, and provider config.

## Summary of Changes (Infrastructure Phase)\n\nBuilt the enrichment infrastructure foundation:\n- Added `EnrichmentProvider` interface to `internal/core/plugin.go`\n- Created `EnrichmentCoordinator` with two-phase execution (API providers concurrent, then LLM) in `internal/core/enrichment.go`\n- Created `EnrichmentWorker` with `SELECT ... FOR UPDATE SKIP LOCKED` polling in `internal/core/worker.go`\n- Added DB migration for `enrichment_retries` column\n- Added `ClaimPendingItems` and `UpdateEnrichmentStatusWithRetries` SQL queries\n- Removed inline enrichment from `SyncService` — enrichment is now fully async\n- Wired coordinator + worker into `app.go` and `main.go` with graceful shutdown\n- 88+ core tests passing\n\nRemaining tasks (Genkit LLM-specific) still need implementation.
