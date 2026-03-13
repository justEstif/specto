---
# project-media-consumption-analysis-gkr1
title: Implement YouTube plugin — API enrichment for imported videos
status: completed
type: task
priority: normal
created_at: 2026-03-13T12:52:53Z
updated_at: 2026-03-13T13:51:26Z
parent: project-media-consumption-analysis-6ksu
blocked_by:
    - project-media-consumption-analysis-jlln
---

Add YouTube Data API v3 enrichment to fill in metadata that Takeout doesn't provide (duration, tags, category, thumbnail, description, view count).

## Tasks

- [x] Implement YouTube API video metadata fetcher with batch support (50 IDs/request)
- [x] Parse ISO 8601 duration into time.Duration
- [x] Map YouTube categoryId to category names (static map of standard IDs)
- [x] Map snippet.tags to fixed tag taxonomy via core.IsValidTag
- [x] Replace Enrich() stub on Plugin with real implementation
- [x] Add NewWithBaseURL() constructor for testing
- [x] Handle deleted/private videos gracefully (missing from API response)
- [x] HTTP error mapping (401, 403, 429, 5xx)
- [x] Write comprehensive tests (batch enrichment, partial failures, deleted videos, ISO duration parsing)
- [x] Verify all tests pass

Implementation:
- Use Enrich() method on the YouTube plugin to call GET /videos?part=snippet,contentDetails,statistics&id={ids}
- Batch up to 50 video IDs per request (1 quota unit each)
- Map: snippet.title → title (canonical, overrides Takeout), contentDetails.duration (ISO 8601) → duration, snippet.tags → tags, snippet.categoryId → raw_metadata.category_id, snippet.thumbnails → raw_metadata.thumbnail_url
- Handle deleted/private videos gracefully (API returns empty for those IDs)
- Track quota usage (10,000 units/day free tier — log usage)
- Requires OAuth token or API key for authenticated requests
- Provide NewWithBaseURL() constructor for testing
- Unit tests: batch enrichment, partial failures, deleted videos

Requires OAuth infrastructure (gqno) OR can work with just an API key. Blocked by YouTube Takeout import task.


## Summary of Changes

Implemented YouTube Data API v3 enrichment in `internal/plugins/youtube/enrich.go` with 22 tests in `enrich_test.go`.

### Key implementation details:
- `EnrichPlugin` wraps the existing `Plugin` (file import) and adds API enrichment via `Enrich()` method
- Batch fetches video metadata in groups of 50 IDs per request (1 quota unit each)
- Parses ISO 8601 durations (PT12M34S) into `time.Duration`
- Maps YouTube category IDs to fixed tag taxonomy (gaming, education, comedy, etc.)
- Normalizes snippet.tags to match fixed taxonomy (lowercase, hyphenated)
- Deduplicates tags when merging with existing item tags
- Stores enrichment metadata in RawMetadata: view_count, like_count, published_at, category_id, category_name, description, thumbnail_url
- Full HTTP error mapping: 401→auth_expired, 403→permission_denied, 429→rate_limit, 5xx→upstream
- Graceful handling of deleted/private videos (missing from API response = unchanged)
- `NewWithBaseURL()` constructor for testing with httptest.Server
- Does not mutate original items slice
