---
# project-media-consumption-analysis-w1nk
title: Scaffold project from go-web-template
status: completed
type: task
priority: high
created_at: 2026-03-12T02:37:25Z
updated_at: 2026-03-12T02:43:28Z
parent: project-media-consumption-analysis-hj33
---

Use `gonew` to scaffold from github.com/justEstif/go-web-template. Update module path, project name, and clean out example handlers/components/migrations. Keep the tooling (mise, air, templ, sqlc, chi, CSRF, docker-compose).

## Summary of Changes

- Scaffolded from `github.com/justEstif/go-web-template` using `gonew`
- Module path: `github.com/justestif/specto`
- Removed example handlers (about, contact) and components
- Kept: Chi router, templ, sqlc, CSRF middleware, air, docker-compose
- Added `/api/v1/health` endpoint
- Added `httpyac` to mise tools + `http/` directory with health.http and local env
- Updated docker-compose DB to `specto_dev`
- Updated `docs/development-workflow.md` with mise tasks and httpyac setup
- Updated README for specto project
