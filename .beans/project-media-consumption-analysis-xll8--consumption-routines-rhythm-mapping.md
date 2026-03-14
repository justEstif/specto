---
# project-media-consumption-analysis-xll8
title: Consumption routines & rhythm mapping
status: completed
type: feature
priority: normal
created_at: 2026-03-14T21:04:53Z
updated_at: 2026-03-14T22:00:01Z
parent: project-media-consumption-analysis-doqg
---

Heatmap grid (24h x 7d) showing media patterns — what content type/genre/platform maps to what time of day and day of week. Makes unconscious habits visible. Persona: Kenji/Maya. Ref: docs/research/ux-use-cases.md #6

## Implementation Plan\n\n- [x] Add SQL query for hourly/daily consumption patterns\n- [x] Run sqlc to generate Go code\n- [x] Add store interface method and implementation\n- [x] Add service method\n- [x] Add handler (page, partial, JSON API)\n- [x] Create heatmap templ component\n- [x] Wire routes in main.go\n- [x] Test (all tests pass)

## Summary of Changes

- Added `ConsumptionHeatmap` SQL query grouping by day-of-week (DOW) and hour-of-day
- Added `HeatmapCell` domain type, store interface method, and PgInsightsStore implementation
- Added `InsightsService.GetConsumptionHeatmap` service method
- Created `routines.templ` with pure CSS heatmap (oklch color interpolation via inline styles, no JS)
- Created `routines.go` handler with page, partial, and JSON API endpoints
- Added insight cards: peak hour, most active day, quiet window detection
- Routes: `/routines`, `/partials/routines`, `/api/v1/insights/heatmap`
- Added Routines link to navbar
