---
# project-media-consumption-analysis-0jo5
title: Software design improvements (review score 7→8-9)
status: completed
type: task
priority: normal
created_at: 2026-03-14T19:15:06Z
updated_at: 2026-03-14T19:19:53Z
---

## Tasks

- [x] Remove database package globals — InitDB returns values, main.go owns lifecycle
- [x] Extract repeated auth+plugin lookup in handlers into shared helper
- [x] Replace _llm_tag_result magic key with typed EnrichmentResult from coordinator

## Notes
From software design review — these three changes address the weakest areas.

## Summary of Changes

1. **database globals removed** — `InitDB` now returns `(*Queries, *pgxpool.Pool, error)`, main.go owns the pool lifecycle via `defer pool.Close()`
2. **handler plugin helper** — new `requirePlugin()` method on Handler extracts auth+plugin lookup; refactored 8 handlers in plugins.go (~80 lines removed)
3. **typed enrichment result** — new `EnrichmentResult` struct replaces `_llm_tag_result` magic key in RawMetadata; worker uses typed `LLMTagResult` field with a simple map lookup
