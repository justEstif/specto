---
# project-media-consumption-analysis-l5g8
title: Implement Spotify plugin — OAuth API sync
status: todo
type: task
priority: normal
created_at: 2026-03-13T12:52:37Z
updated_at: 2026-03-13T12:53:07Z
parent: project-media-consumption-analysis-6ksu
blocked_by:
    - project-media-consumption-analysis-gqno
---

Implement the Spotify OAuth API sync path using the recently-played endpoint.

Implementation:
- Add OAuth sync to the spotify plugin (same package, may be a separate plugin registration or a dual-mode plugin)
- Call GET /me/player/recently-played?limit=50&after={cursor}
- Cursor: Unix ms timestamp of last played_at
- Map fields: track.id → external_id, track.name → title, track.artists[0].name → creator, played_at → consumed_at, track.duration_ms → duration
- Store album, popularity, track URL in raw_metadata
- Handle 401 (auth_expired), 429 (rate_limit with Retry-After), 5xx (upstream)
- Provide NewWithBaseURL() constructor for testing with httptest.Server
- Unit tests covering: successful sync, incremental sync with cursor, auth expired, rate limit, upstream error

Requires OAuth infrastructure task (gqno) to be completed first.

Limitation: only returns last 50 tracks — best used with frequent polling.
