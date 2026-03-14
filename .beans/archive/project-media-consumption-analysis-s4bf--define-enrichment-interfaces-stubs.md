---
# project-media-consumption-analysis-s4bf
title: Define enrichment interfaces (stubs)
status: completed
type: task
priority: normal
created_at: 2026-03-12T20:21:11Z
updated_at: 2026-03-12T20:51:51Z
parent: project-media-consumption-analysis-86vz
blocked_by:
    - project-media-consumption-analysis-3vdi
---

Create internal/core/enrichment.go with the Enricher interface and supporting types, but NOT the LLM implementation (that's in the Enrichment/LLM epic). Define: Enricher interface with Enrich(ctx, MediaItem) (TagResult, error), TagResult struct (genre/topic/mood/format slices with confidence scores), buildTagPrompt() function signature. Include a NoOpEnricher for testing/bootstrapping that returns empty results. This establishes the contract that plugins and sync orchestration depend on.

## Todo

- [x] Define Enricher interface with Enrich(ctx, MediaItem) (TagResult, error)
- [x] Define TagResult struct with genre/topic/mood/format slices and confidence scores
- [x] Implement NoOpEnricher that returns empty results
- [x] Write unit tests for NoOpEnricher
- [x] Verify code compiles

## Summary of Changes

Created `internal/core/enrichment.go` with:
- `Enricher` interface — `Enrich(ctx, MediaItem, existingTags) (*TagResult, error)`
- `TagResult` struct with `Genre`, `Topic`, `Mood`, `Format` slices of `TagScore` (tag + confidence)
- `TagScore` struct pairing tag name with float32 confidence (0.0-1.0)
- `IsEmpty()` and `AllTags()` helper methods on TagResult
- `NoOpEnricher` stub returning empty results for testing/bootstrapping
- Compile-time interface assertion

9 tests in `enrichment_test.go` covering NoOpEnricher and all TagResult methods.
