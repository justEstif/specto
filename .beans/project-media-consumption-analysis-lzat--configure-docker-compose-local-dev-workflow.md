---
# project-media-consumption-analysis-lzat
title: Configure Docker Compose & local dev workflow
status: completed
type: task
priority: normal
created_at: 2026-03-12T02:37:25Z
updated_at: 2026-03-12T03:12:11Z
parent: project-media-consumption-analysis-hj33
blocked_by:
    - project-media-consumption-analysis-w1nk
---

Update docker-compose.yml for specto (DB name, any additional services). Verify mise run setup/dev/build all work end-to-end. Document in README.

## Summary of Changes

- Verified docker compose starts Postgres correctly (removed obsolete `version` attribute from docker-compose.yml)
- Verified `mise run setup` runs migrations
- Verified `mise run dev` starts live reload server (health endpoint responds)
- Verified `mise run build` produces production binary
- No additional services needed for MVP
- README already documents the workflow accurately
