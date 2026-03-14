---
# project-media-consumption-analysis-vxea
title: Enrichment observability & retry
status: completed
type: task
priority: normal
created_at: 2026-03-14T03:53:07Z
updated_at: 2026-03-14T19:53:31Z
parent: project-media-consumption-analysis-eo0f
blocked_by:
    - project-media-consumption-analysis-2rhd
---

Add observability and retry mechanisms for the enrichment pipeline.

## Tasks

- [x] Track enrichment status per item (pending → plugin-enriched → enriched / failed) — already implemented
- [x] Retry failed enrichments (background job or on-demand) — already implemented (markRetry with max 3)
- [x] Per-source error logging with rate-limit backoff tracking — per-provider stats in BatchStats + wide event
- [x] Enrichment metrics: items enriched, tags assigned, LLM calls, failures, latency — wide event per tick
- [x] Re-enrichment support (when tag set expands or prompts change) — ResetEnrichment + EnrichmentStats queries

## Reference

See docs/enrichment.md — Error Handling section.

## Summary of Changes

### Wide event logging for enrichment worker
- Consolidated scattered log calls in worker.go into a single wide event per batch tick
- Each tick emits one `enrichment_tick` log with: batch_size, items_enriched, items_failed, items_retried, tags_persisted, tag_errors, duration_ms, outcome, and per-provider stats
- Provider stats include items_received, items_enriched, tags_assigned, and errors
- LLM metrics: llm_items, llm_tags, llm_errors

### Per-provider metrics via BatchStats
- Added `BatchStats` and `ProviderStats` types to enrichment coordinator
- `Run()` now returns `([]EnrichmentResult, *BatchStats, error)` — stats are populated during both phases
- Provider failures tracked with name + error message

### Re-enrichment support
- Added `ResetEnrichmentByUser` SQL query — resets all enriched/failed items to pending
- Added `ResetEnrichmentByID` SQL query — reset a single item
- Added `EnrichmentStats` SQL query — counts by status (pending/enriching/enriched/failed)
- Added corresponding store interface methods and implementations
- Regenerated sqlc code

### persistTags returns counts
- `persistTags()` now returns (persisted, errors) counts instead of logging individually
- Tag errors are aggregated into the wide event
