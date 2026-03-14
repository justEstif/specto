---
# project-media-consumption-analysis-7sq9
title: Topic obsession tracker
status: completed
type: feature
priority: normal
created_at: 2026-03-14T21:04:53Z
updated_at: 2026-03-14T23:10:41Z
parent: project-media-consumption-analysis-doqg
---

Visualize obsession arcs: onset, peak, duration, fadeout for specific tags/topics. Show how interest spreads across platforms over time. Time-series tag aggregation + simple spike detection. Persona: Dario/Kenji. Ref: docs/research/ux-use-cases.md #8

## Plan\n- [x] Add SQL queries (time-bucketed tag counts for obsession arcs)\n- [x] Run sqlc generate\n- [x] Add store + service methods\n- [x] Add API endpoint + handler\n- [x] Add templ component\n- [x] Add HTML route/partial

## Summary of Changes

Implemented topic obsession tracker feature:
- Two SQL queries: `TopicTimeSeries` (weekly tag counts) and `TopicSpikes` (recent activity spikes vs historical average)
- Full stack: store, service, API handlers (`GET /api/v1/insights/topic-timeline`, `GET /api/v1/insights/topic-spikes`), templ page (`/obsessions`)
- UI shows "Currently Obsessed With" spike cards and sparkline-style obsession arc bars for top 5 tags
- HTMX filter bar with platform/type/range controls (defaults to 90d for wider obsession detection)
