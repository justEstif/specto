---
# project-media-consumption-analysis-fqwt
title: Implement Plugin API endpoints
status: completed
type: feature
priority: normal
created_at: 2026-03-13T12:43:15Z
updated_at: 2026-03-13T12:48:02Z
parent: project-media-consumption-analysis-j0cp
---

GET /plugins, GET /plugins/{plugin}, POST /plugins/{plugin}/connect, POST /plugins/{plugin}/import, DELETE /plugins/{plugin}/disconnect, POST /plugins/{plugin}/sync, GET /plugins/{plugin}/sync-history. Per docs/api.md.

## Summary of Changes\n\nImplemented all 7 plugin API endpoints in `internal/handlers/plugins.go`:\n\n- `GET /api/v1/plugins` — List all registered plugins with per-user state\n- `GET /api/v1/plugins/{plugin}` — Get single plugin with capabilities\n- `POST /api/v1/plugins/{plugin}/connect` — Start OAuth connection flow\n- `POST /api/v1/plugins/{plugin}/import` — Upload file for file-import plugins\n- `DELETE /api/v1/plugins/{plugin}/disconnect` — Disconnect and delete credentials\n- `POST /api/v1/plugins/{plugin}/sync` — Trigger sync with rate limit handling\n- `GET /api/v1/plugins/{plugin}/sync-history` — List recent sync runs\n\n15 unit tests in `plugins_test.go` covering auth, not-found, capabilities, connect/disconnect, sync history.
