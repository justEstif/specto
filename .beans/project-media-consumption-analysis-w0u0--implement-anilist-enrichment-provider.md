---
# project-media-consumption-analysis-w0u0
title: Implement AniList enrichment provider
status: completed
type: feature
priority: normal
created_at: 2026-03-14T17:50:01Z
updated_at: 2026-03-14T17:59:32Z
parent: project-media-consumption-analysis-eo0f
---

Add AniList GraphQL-based enrichment provider for anime/manga items. No API key required (public API). Rate limit: 90 req/min.

## Summary of Changes\n\nImplemented AniList enrichment provider:\n- Created `internal/plugins/anilist/provider.go` with GraphQL-based anime/manga lookup\n- Genre mapping: AniList genres → fixed tags (direct + mapped)\n- Tag filtering: rank >= 60, non-spoiler, normalized to fixed set\n- Format tags: series, film, episode, graphic-novel\n- No API key required — always registered\n- 22 tests covering lookup, genre/tag mapping, spoiler filtering, errors\n- Wired into app.go unconditionally
