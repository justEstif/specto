---
# project-media-consumption-analysis-2b2i
title: Figure out hosting
status: completed
type: task
priority: normal
created_at: 2026-03-15T01:43:37Z
updated_at: 2026-03-15T02:35:08Z
parent: project-media-consumption-analysis-doqg
---

Research and decide on hosting solution for Specto v1 deployment

## Tasks
- [x] Create Dockerfile (multi-stage, Go build)
- [x] Create compose.prod.yaml (app + postgres)
- [x] Expand self-hosting guide (systemd, backup/restore, monitoring, troubleshooting)

## Summary of Changes
- Created `Dockerfile` with multi-stage build (golang builder → alpine runtime), includes templ/sqlc code generation
- Created `compose.prod.yaml` with app + postgres services, healthcheck, env_file support
- Expanded `docs/self-hosting.md` with Docker Compose deployment, binary deployment with systemd service, reverse proxy (Caddy + Nginx), backup/restore, monitoring, troubleshooting, and security sections — modeled after go-spotify-era-organizer's self-hosting guide
