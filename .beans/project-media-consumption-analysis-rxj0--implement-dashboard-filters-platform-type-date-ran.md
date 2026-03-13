---
# project-media-consumption-analysis-rxj0
title: Implement dashboard filters (platform, type, date range)
status: completed
type: feature
priority: normal
created_at: 2026-03-13T21:02:09Z
updated_at: 2026-03-13T21:11:53Z
blocked_by:
    - project-media-consumption-analysis-yp0q
---

Add filtering to the dashboard page so all sections (stats, activity chart, recent items, top tags, platform breakdown) respond to platform, media type, and date range filters.

## Tasks
- [x] Add filter bar component to dashboard.templ (platform select, type select, date range tabs)
- [x] Create a dashboard partial endpoint that returns filtered dashboard content
- [x] Update handler to accept filter query params and pass them to data queries
- [x] Update InsightsService/store queries to accept platform and type filters
- [x] Move the activity chart range tabs into the global filter bar
- [x] Wire HTMX: filter changes swap the dashboard content area
- [x] Ensure filter state is reflected in URL (hx-push-url)
- [x] Test with multiple filter combinations

## Summary of Changes

Added dashboard filtering across the full stack:

### SQL (queries.sql)
- Added `PlatformBreakdownFiltered` and `TagDistributionFiltered` queries with optional `platform` and `media_type` params via `sqlc.narg()`

### Domain (core/)
- Added `InsightsFilter` struct with optional Platform and MediaType fields
- Added filtered variants to `InsightsStore` interface: `PlatformBreakdownFiltered`, `TagDistributionFiltered`, `ListMediaItemsFiltered`
- Added `GetSummaryFiltered`, `GetTimelineFiltered`, `GetPlatformBreakdownFiltered`, `GetTagDistributionFiltered` to `InsightsService`
- Original unfiltered methods now delegate to filtered variants with empty filter

### Store (core/store/)
- Implemented all filtered methods in `PgInsightsStore`
- Updated Querier interface with new query methods

### Handler (handlers/home.go)
- `Home` now parses `?platform=`, `?type=`, `?range=` query params
- Added `DashboardPartial` handler at `GET /partials/dashboard`
- `RecentItemsPartial` now carries filter state for Show more
- Added `parseDashboardFilters()` helper

### Template (components/dashboard.templ)
- Added `DashboardFilters` struct and `dashboardFilterBar` component
- Extracted `DashboardContent` as a separate templ for partial swap target
- Filter bar: platform select, type select, date range tabs (moved from activity section)
- All controls use `hx-get` with `hx-include` to preserve sibling filter values
- `hx-push-url` on all controls for URL state
- Show more button carries active filters
- Empty state messages are filter-aware

### Route
- Added `GET /partials/dashboard` to authenticated partials group
