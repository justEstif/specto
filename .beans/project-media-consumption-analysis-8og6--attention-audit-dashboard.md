---
# project-media-consumption-analysis-8og6
title: Attention audit dashboard
status: completed
type: feature
priority: high
created_at: 2026-03-14T21:04:53Z
updated_at: 2026-03-14T21:20:36Z
parent: project-media-consumption-analysis-doqg
---

Show users where their attention actually goes — platform/type/topic breakdowns with time-period selection. The core value proposition. All data already available through existing queries. Persona: Maya (Intentional Consumer). Ref: docs/research/ux-use-cases.md #1

## Implementation Plan

- [x] Add tag-category-specific distribution queries (genre/topic/mood separately)
- [x] Add attention time breakdown query (time_spent by platform/type)
- [x] Run sqlc to generate Go code
- [x] Extend InsightsService with attention audit methods
- [x] Add new API endpoints for attention audit data
- [x] Create attention audit templ page/components
- [x] Add route and handler
- [x] Test (all tests pass)

## Summary of Changes

- Added `TagDistributionByCategory` SQL query for category-specific tag breakdowns
- Added `AttentionByType` SQL query for time-spent analysis by media type
- Extended `InsightsStore` interface with both new methods
- Added `InsightsService.GetTagDistributionByCategory` and `GetAttentionByType`
- Created `attention.templ` with full attention audit page (type cards, genre/topic/mood breakdowns, platform attention)
- Created `attention.go` handler with page, partial, and JSON API endpoints
- Added `/attention` page route, `/partials/attention` HTMX partial, `/api/v1/insights/attention-by-type` and `/api/v1/insights/tags-by-category` API endpoints
- Added Attention link to navbar
