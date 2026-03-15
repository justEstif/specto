---
# project-media-consumption-analysis-tymy
title: AniList SourcePlugin (GraphQL API)
status: completed
type: feature
priority: normal
created_at: 2026-03-15T01:25:05Z
updated_at: 2026-03-15T01:35:14Z
parent: project-media-consumption-analysis-geho
---

Promote AniList from enrichment-only provider to a full SourcePlugin that fetches user anime/manga watch/read lists via the public GraphQL API. Supports OAuth for private lists, no auth for public lists. Incremental sync via updatedAt cursor.

## Tasks
- [ ] Create SourcePlugin implementation in internal/plugins/anilist/
- [ ] Implement OAuth flow for private list access
- [ ] Query user's anime list (MediaListCollection) via GraphQL
- [ ] Query user's manga list via GraphQL
- [ ] Map AniList media entries to MediaItem (status, score, progress, dates)
- [ ] Use AniList media ID as ExternalID
- [ ] Implement cursor-based incremental sync (updatedAt timestamp)
- [ ] Handle rate limiting (90 req/min)
- [ ] Leverage existing AniList EnrichmentProvider for Enrich()
- [ ] Write comprehensive tests with mock GraphQL server
- [ ] Register plugin in cmd/web/main.go

## Summary of Changes\nImplemented AniList OAuth SourcePlugin fetching anime+manga lists via GraphQL, cursor-based incremental sync, rate limiting. 28 tests passing.
