---
# project-media-consumption-analysis-goow
title: Implement Insights API endpoints
status: completed
type: feature
priority: normal
created_at: 2026-03-13T12:43:17Z
updated_at: 2026-03-13T12:48:08Z
parent: project-media-consumption-analysis-j0cp
---

GET /insights/summary, GET /insights/platform-breakdown, GET /insights/tags, GET /insights/timeline. Per docs/api.md.

## Summary of Changes\n\nImplemented in `internal/handlers/insights.go`:\n\n- `GET /api/v1/insights/summary` — Total items, duration, top platform/type\n- `GET /api/v1/insights/platform-breakdown` — Consumption stats by platform+type\n- `GET /api/v1/insights/tags` — Tag distribution with configurable limit\n- `GET /api/v1/insights/timeline` — Time-bucketed consumption (day/week/month)\n\n6 unit tests in `insights_test.go` covering auth, all endpoints, invalid bucket.
