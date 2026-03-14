---
# project-media-consumption-analysis-l2a9
title: Cross-platform taste DNA
status: completed
type: feature
priority: normal
created_at: 2026-03-14T21:04:53Z
updated_at: 2026-03-14T23:10:41Z
parent: project-media-consumption-analysis-doqg
---

Reveal the unified taste profile across platforms — which tags/genres/moods cut across Spotify, YouTube, Netflix etc. The killer insight no single-platform tool can offer. Needs cross-platform tag aggregation with HAVING COUNT(DISTINCT platform) >= 2. Persona: Dario. Ref: docs/research/ux-use-cases.md #2

## Plan\n- [x] Add SQL queries (cross-platform tags with HAVING COUNT(DISTINCT platform) >= 2)\n- [x] Run sqlc generate\n- [x] Add store + service methods\n- [x] Add API endpoint + handler\n- [x] Add templ component\n- [x] Add HTML route/partial

## Summary of Changes

Implemented cross-platform taste DNA feature:
- SQL query with `HAVING COUNT(DISTINCT platform) >= 2` to find tags shared across platforms
- Full stack: store, service, API handler (`GET /api/v1/insights/taste-dna`), templ page (`/taste-dna`)
- UI shows unified taste profile as badges, with per-category breakdowns showing which platforms share each tag
- HTMX filter bar with platform/type/range controls
