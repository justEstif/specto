---
# project-media-consumption-analysis-khe4
title: 'Era detection Phase 2: handlers, UI, and LLM naming'
status: completed
type: task
priority: high
created_at: 2026-03-15T00:25:48Z
updated_at: 2026-03-15T00:31:50Z
parent: project-media-consumption-analysis-oyb5
---

## Tasks
- [x] Add Eras tab to timeline.templ with era timeline visualization
- [x] Add era tab logic to timeline_page.go handler + action handlers (eras_page.go)
- [x] Add EraNamer interface to core + LLM era naming prompt
- [x] Wire era naming into enrichment + era worker
- [x] Add routes to main.go (navbar unchanged - eras is a timeline tab)
- [x] Run templ generate + build + test

## Summary of Changes

### Eras as Timeline Tab
- Added `TimelineTabEras` to `components/timeline.templ` as third tab alongside Overview and Activity
- Era timeline renders per-media-type "lanes" (Music eras, Video eras) with connected segment visualization
- Each era segment shows: date range, title (confirmed/suggested/untitled), item count, top weighted tags, status badge
- Suggested eras have hover-reveal actions: Confirm, Rename (inline form), Dismiss
- HTMX-powered interactions: confirm/rename/dismiss swap the individual era segment without full page reload

### Handler Layer
- `timeline_page.go`: Added `TimelineTabEras` case + `fetchErasData()` method that loads eras per media type with tags
- `eras_page.go`: New file with `ConfirmEra`, `UpdateEraTitle`, `DismissEra` action handlers
- Routes wired under `/api/v1/eras/{id}/confirm`, `/api/v1/eras/{id}/title`, `/api/v1/eras/{id}`

### LLM Era Naming
- `EraNamer` interface added to `core/era.go` (single method: `NameEra(ctx, mediaType, tags)`)
- `prompts/era_name.prompt`: Dotprompt template that generates 2-5 word evocative era titles from tags
- Prompt rules: no generic labels, no "era" in title, lowercase, mood/scene-oriented
- `GenkitEnricher` implements `EraNamer` — registered schemas, lookup prompt, `NameEra()` method
- Era worker calls `NameEra()` after persisting tags, stores result as `suggested_title`
- New `UpdateEraSuggestedTitle` SQL query + store method + interface method

### SQL
- Added `UpdateEraSuggestedTitle` query to `queries.sql`
- Regenerated sqlc

### Test Fixes
- Added era query stubs to `mock_querier_test.go` (11 methods)
