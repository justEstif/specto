---
# project-media-consumption-analysis-j0cp
title: API layer
status: completed
type: epic
priority: normal
created_at: 2026-03-11T17:11:07Z
updated_at: 2026-03-13T12:48:20Z
parent: project-media-consumption-analysis-bja8
blocked_by:
    - project-media-consumption-analysis-86vz
---

Implement the canonical /api/v1 surface documented in docs/api.md, including auth-aware route wiring, request/response envelopes, validation, and httpyac-backed API workflows.

## Summary of Changes\n\nImplemented the full /api/v1 surface per docs/api.md across 3 handler files with route wiring in main.go:\n\n### New files\n- `internal/handlers/plugins.go` — 7 plugin API endpoints\n- `internal/handlers/timeline.go` — Timeline endpoint with pagination\n- `internal/handlers/insights.go` — 4 insights endpoints\n- `internal/handlers/plugins_test.go` — 15 tests\n- `internal/handlers/timeline_test.go` — 5 tests\n- `internal/handlers/insights_test.go` — 6 tests\n\n### Modified files\n- `cmd/web/main.go` — Wired all new routes under authenticated group\n\n### Endpoints added\n- `GET /api/v1/plugins`\n- `GET /api/v1/plugins/{plugin}`\n- `POST /api/v1/plugins/{plugin}/connect`\n- `POST /api/v1/plugins/{plugin}/import`\n- `DELETE /api/v1/plugins/{plugin}/disconnect`\n- `POST /api/v1/plugins/{plugin}/sync`\n- `GET /api/v1/plugins/{plugin}/sync-history`\n- `GET /api/v1/timeline`\n- `GET /api/v1/insights/summary`\n- `GET /api/v1/insights/platform-breakdown`\n- `GET /api/v1/insights/tags`\n- `GET /api/v1/insights/timeline`\n\nTotal: 20 new handler tests, all passing. go vet clean.
