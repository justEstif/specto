---
# project-media-consumption-analysis-gp7h
title: Implement AniList enrichment provider
status: completed
type: feature
priority: normal
created_at: 2026-03-14T17:52:36Z
updated_at: 2026-03-14T17:57:30Z
---

Add AniList GraphQL-based enrichment provider for anime/manga items. Implements EnrichmentProvider interface with genre/tag mapping, rate limiting, and test coverage.

## Tasks\n\n- [x] Read existing codebase (core interfaces, tags, app wiring)\n- [x] Create AniList provider package (internal/plugins/anilist/provider.go)\n- [x] Implement GraphQL client with rate limiting\n- [x] Implement genre/tag mapping logic\n- [x] Implement Enrich method with error handling\n- [x] Write comprehensive tests (provider_test.go)\n- [x] Wire provider in app.go\n- [x] Verify build and tests pass

## Summary of Changes

### Files Created
- **internal/plugins/anilist/provider.go** — AniList enrichment provider (430 lines)
- **internal/plugins/anilist/provider_test.go** — Comprehensive test suite (22 tests)

### Files Modified
- **internal/app/app.go** — Added anilist import and unconditional registration

### Provider API
- `Name()` → `"anilist"`
- `Supports(mediaType, platform)` → true for `"video"` items
- `Enrich(ctx, items)` → searches AniList by title, adds genre/format/mood/topic tags

### Genre/Tag Mapping
- **Direct genres**: Action, Adventure, Comedy, Drama, Fantasy, Horror, Mystery, Romance, Sci-Fi, Thriller
- **Mapped genres**: Supernatural→fantasy, Slice of Life→drama, Sports→sports, Music→musical
- **AniList tags** (rank ≥ 60, non-spoiler): Psychological→intense, Gore→dark, Mecha→technology, Isekai→fantasy, Time Travel→sci-fi
- **Format tags**: TV→series, MOVIE→film, OVA/ONA/SPECIAL→episode, MANGA→graphic-novel

### Key Design Decisions
- Rate limiter: channel-based token bucket, 667ms interval (90 req/min)
- Per-item failures logged and skipped, never abort the batch
- Media type detection: RawMetadata hints > platform name > default ANIME
- All tags validated against core.IsValidTag() before assignment
- No API key required — always registered unconditionally

### Test Results
All 22 tests passing, covering: successful lookup, no match, genre mapping, tag filtering (spoilers + low rank), format detection, media type detection, HTTP errors (429/500), invalid JSON, GraphQL errors, batch partial failure, context cancellation, immutability, tag preservation, and valid-tag-only enforcement.
