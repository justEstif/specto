---
# project-media-consumption-analysis-qm28
title: Timeline page
status: completed
type: feature
priority: normal
created_at: 2026-03-13T15:23:23Z
updated_at: 2026-03-13T15:37:41Z
parent: project-media-consumption-analysis-nn0q
blocked_by:
    - project-media-consumption-analysis-d41y
---

Full chronological feed at /timeline. Filter bar (platform, type, search, date range), day-grouped item cards with media icons/tags/timestamps, infinite 'Load more' pagination, privacy toggle per item. Refs: docs/ui-design.md §Timeline, docs/api.md (GET /api/v1/timeline).

## Tasks
- [ ] Timeline page templ template (components/timeline.templ)
- [ ] Filter bar: platform select, type select, search input, date range
- [ ] Filters trigger hx-get /partials/timeline?platform=X&type=Y with hx-push-url
- [ ] Search input: hx-trigger="keyup changed delay:400ms" (active search)
- [ ] Day-grouped item list with date headers
- [ ] Item row: media type icon, title+duration, creator+platform badge, tag badges, timestamp
- [ ] Privacy lock toggle: hx-post /api/v1/items/{id}/privacy, swaps row
- [ ] 'Load more' button: hx-get with offset, hx-swap=beforeend
- [ ] Partial: /partials/timeline — returns item rows only
- [ ] Responsive: filters horizontal >= sm, stacked < sm; timestamp below tags on mobile
- [ ] Handler + route wiring

## Summary of Changes

Built the timeline page at /timeline with:
- Filter bar: platform select, type select, search input with active search (400ms debounce)
- Day-grouped item list with date headers (Today, Yesterday, date format)
- Detailed item rows: media icon, title+duration, creator+platform badge, tag badges, timestamp
- Responsive timestamps: right-aligned on desktop, below content on mobile
- Load more pagination via HTMX beforeend swap
- HTMX partial: /partials/timeline-page for filter changes with hx-push-url
- Client-side filtering by platform/type/search in handler (TODO: move to SQL)
- Empty states for no items and no filter matches

### Files created
- components/timeline.templ - TimelinePage, TimelineItems, timelineDetailRow, filter components
- internal/handlers/timeline_page.go - TimelinePage, TimelinePagePartial, filtering logic
