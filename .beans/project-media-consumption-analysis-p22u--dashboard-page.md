---
# project-media-consumption-analysis-p22u
title: Dashboard page
status: completed
type: feature
priority: high
created_at: 2026-03-13T15:23:12Z
updated_at: 2026-03-13T15:36:16Z
parent: project-media-consumption-analysis-nn0q
blocked_by:
    - project-media-consumption-analysis-d41y
---

Authenticated dashboard at / (when logged in). Summary stats (4-col grid), activity bar chart with 7d/30d/90d tabs, recent items list with 'Show more' pagination, top tags list, platform breakdown bars. Refs: docs/ui-design.md §Dashboard, docs/api.md (insights endpoints).

## Tasks
- [ ] Dashboard page templ template (components/dashboard.templ)
- [ ] Summary stats row: items count, total time, top source, top type (DaisyUI stat, 4-col grid)
- [ ] Activity chart section with time range tabs (7d/30d/90d)
- [ ] Partial: /partials/activity-chart?range=X — hx-get on tab click
- [ ] Recent items list (DaisyUI list, media type icons, platform badges)
- [ ] 'Show more' button: hx-get /partials/timeline?offset=5&limit=5, hx-swap=beforeend
- [ ] Top Tags list with counts
- [ ] Platform Breakdown horizontal bars with percentages
- [ ] Responsive: 4-col -> 2-col -> 1-col stat grid, stacked bottom sections on mobile
- [ ] Handler + route wiring (authenticated, redirect to /login if not)

## Summary of Changes

Built the dashboard page as the authenticated home view at /. Includes:
- Summary stats: 4-col responsive grid with total items, total time, top source, top type
- Activity chart: CSS bar chart with 7d/30d/90d tab switching via HTMX partial swap
- Recent items: list with media type icons, platform badges, relative timestamps, and Show more pagination
- Top Tags: ranked list with counts
- Platform Breakdown: horizontal percentage bars
- All data fetched server-side from InsightsService and MediaItemStore
- HTMX partials: /partials/activity-chart and /partials/timeline for dynamic updates
- Empty states for all sections
- Home handler now renders dashboard when authenticated, landing page when not

### Files created/modified
- components/dashboard.templ - Dashboard + all sub-components + helper functions
- internal/handlers/home.go - Rewrote with dashboard rendering + partials
- cmd/web/main.go - Added authenticated partials route group
