---
# project-media-consumption-analysis-gkr1
title: Implement YouTube plugin — API enrichment for imported videos
status: todo
type: task
priority: normal
created_at: 2026-03-13T12:52:53Z
updated_at: 2026-03-13T12:53:08Z
parent: project-media-consumption-analysis-6ksu
blocked_by:
    - project-media-consumption-analysis-jlln
---

Add YouTube Data API v3 enrichment to fill in metadata that Takeout doesn't provide (duration, tags, category, thumbnail, description, view count).

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
