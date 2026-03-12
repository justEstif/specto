---
# project-media-consumption-analysis-binz
title: Implement tag management and fixed tag set
status: completed
type: task
priority: normal
created_at: 2026-03-12T20:21:04Z
updated_at: 2026-03-12T20:54:59Z
parent: project-media-consumption-analysis-86vz
blocked_by:
    - project-media-consumption-analysis-3vdi
---

Define the canonical fixed tag set from docs/enrichment.md as Go data in internal/core/tags.go: genre tags (action, comedy, drama, etc.), topic tags, mood tags, format tags. Implement tag validation (reject unknown tags), tag alias resolution (look up tag_aliases before persisting), and get-or-create logic for the tags table. This is used by both the enrichment pipeline and sync orchestration. Include unit tests.

## Todo

- [x] Define canonical fixed tag sets as Go constants/data (genre, topic, mood, format)
- [x] Implement tag validation (reject unknown tags)
- [x] Define TagStore interface in store layer with get-or-create and alias resolution
- [x] Implement PgTagStore wrapping sqlc queries
- [x] Write unit tests for tag validation and tag sets
- [x] Write unit tests for PgTagStore with mock querier
- [x] Verify code compiles

## Summary of Changes

Implemented tag management across two packages:

**internal/core/tags.go** — Fixed tag taxonomy (128 tags across 4 categories: 45 genre, 41 topic, 20 mood, 24 format), `IsValidTag()`, `TagCategoryOf()`, `ValidateTagResult()` (filters LLM output to fixed set), `AllFixedTags()`, `TagsByCategory()`.

**internal/core/store/tag_store.go** — `TagStore` interface + `PgTagStore`: `ResolveTag()` (direct match + alias fallback with case normalization), `GetOrCreate()` (validates against fixed set, auto-categorizes), `AddMediaItemTag()` (plugin/authoritative=nil confidence, llm=float32), `ListMediaItemTags()`.

Extended `Querier` interface with 5 tag query methods. 20 new tests (tags_test.go + tag_store_test.go).
