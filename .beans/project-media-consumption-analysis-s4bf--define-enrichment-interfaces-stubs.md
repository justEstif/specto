---
# project-media-consumption-analysis-s4bf
title: Define enrichment interfaces (stubs)
status: todo
type: task
created_at: 2026-03-12T20:21:11Z
updated_at: 2026-03-12T20:21:11Z
parent: project-media-consumption-analysis-86vz
blocked_by:
    - project-media-consumption-analysis-3vdi
---

Create internal/core/enrichment.go with the Enricher interface and supporting types, but NOT the LLM implementation (that's in the Enrichment/LLM epic). Define: Enricher interface with Enrich(ctx, MediaItem) (TagResult, error), TagResult struct (genre/topic/mood/format slices with confidence scores), buildTagPrompt() function signature. Include a NoOpEnricher for testing/bootstrapping that returns empty results. This establishes the contract that plugins and sync orchestration depend on.
