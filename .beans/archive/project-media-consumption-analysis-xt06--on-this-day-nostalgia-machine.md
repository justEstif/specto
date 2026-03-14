---
# project-media-consumption-analysis-xt06
title: On This Day / nostalgia machine
status: completed
type: feature
priority: high
created_at: 2026-03-14T21:04:53Z
updated_at: 2026-03-14T21:16:12Z
parent: project-media-consumption-analysis-doqg
---

Show users what they were consuming exactly 1/2/5 years ago today. Simple date-comparison query, high emotional payoff, drives daily return visits. Persona: Dario/Kenji. Ref: docs/research/ux-use-cases.md #5

## Implementation Plan

- [x] Add SQL query for items consumed on this date in past years
- [x] Run sqlc to generate Go code
- [x] Add store interface method and implementation
- [x] Add core service method (not needed, store handles directly)
- [x] Add JSON API endpoint
- [x] Add HTMX partial endpoint
- [x] Add templ component (dashboard card + dedicated section)
- [x] Wire routes in main.go
- [x] Test (all tests pass)

## Summary of Changes

- Added `OnThisDay` SQL query that matches items by month+day from previous years
- Added `OnThisDayItem` domain type and `OnThisDay` method to `MediaItemStore` interface
- Implemented store method in `PgMediaItemStore`
- Created `on_this_day.templ` component with year-grouped nostalgia card
- Added JSON API at `GET /api/v1/insights/on-this-day`
- Added HTMX partial at `GET /partials/on-this-day`
- Integrated into dashboard (shows when no filters are active)
- Updated all mock stores to satisfy the new interface method
