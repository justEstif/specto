---
# project-media-consumption-analysis-tizz
title: Media diet scorecard on timeline page
status: completed
type: feature
priority: normal
created_at: 2026-03-14T23:35:34Z
updated_at: 2026-03-14T23:44:52Z
parent: project-media-consumption-analysis-doqg
---

Add a summary scorecard card at the top of the timeline page showing consumption takeaways for the current 90-day window. Uses existing InsightsStore methods (TagDistribution, PlatformBreakdown, AttentionByType, TopicSpikes).

## Scope

A collapsible summary card at the top of the timeline page that gives users takeaways from their last 90 days of media consumption. No new DB queries needed — the InsightsStore already provides all the aggregation methods.

## What the scorecard shows

### Row 1: Top-line stats (from AttentionByType + PlatformBreakdown)
- Total items consumed (count)
- Total time spent (formatted as hours)
- Number of unique platforms active
- Number of unique media types

### Row 2: Top tags (from TagDistributionByCategory)
- Top 3 genres (badge pills)
- Top 3 moods (badge pills)
- Top 3 topics (badge pills)

### Row 3: What's trending (from TopicSpikes)
- 1-3 tags with recent spikes vs historical average
- Show as "Trending: sci-fi ↑42%" style callouts
- Only shown if spikes exist

## Implementation plan

- [x] Extract generic TabBar component from insights tabs (components/tabs.templ)
- [x] Refactor insights page to use generic TabBar
- [x] Add tab structure to timeline page (Overview | Timeline tabs)
- [x] Add scorecard data struct to components package (TimelineScorecardData)
- [x] Add handler method to fetch scorecard data (calls InsightsStore methods in parallel)
- [x] Create templ component for the scorecard card (Overview tab content)
- [x] Wire scorecard into timeline handler with tab routing + partials
- [x] Handle empty states (no items, no tags yet, enrichment pending)
- [x] Regenerate templ and verify build

## Design notes

- Use DaisyUI stat components for row 1
- Badge pills for tags, color-coded by category
- Collapsible via DaisyUI collapse or details/summary
- Skeleton loading state while data loads
- Responsive: stack vertically on mobile
- Keep it compact — this is a summary, not a dashboard

## Summary of Changes

### Files created
- `components/tabs.templ` — Generic `TabBar` component with `Tab` struct, reusable across pages

### Files modified
- `components/insights.templ` — Refactored to use generic `TabBar` via `insightsTabDefs()` helper
- `components/timeline.templ` — Added tab support (Overview/Activity), `ScorecardData` struct, scorecard components (stats grid, top tags, trending spikes), empty state
- `internal/handlers/timeline_page.go` — Added tab routing (`validTimelineTab`, `TimelineTabPartial`, `buildTimelinePageData`), scorecard data fetching (`fetchScorecardData` using InsightsService)
- `cmd/web/main.go` — Added routes: `GET /timeline/{tab}`, `GET /partials/timeline/{tab}`

### Architecture
- Timeline page now defaults to Overview tab showing a consumption scorecard (top-line stats, top tags by category, trending spikes)
- Activity tab preserves the existing chronological feed with filters
- Both insights and timeline pages share the generic `TabBar` component
