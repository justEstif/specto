---
# project-media-consumption-analysis-l5g8
title: Implement Spotify plugin — OAuth API sync
status: completed
type: task
priority: normal
created_at: 2026-03-13T12:52:37Z
updated_at: 2026-03-13T13:43:22Z
parent: project-media-consumption-analysis-6ksu
blocked_by:
    - project-media-consumption-analysis-gqno
---

Implement the Spotify OAuth API sync path using the recently-played endpoint.

## Plan\n\n- [ ] Create  struct with  /  constructors\n- [ ] Implement  returning ,  returning ,  returning Spotify OAuth config\n- [ ] Implement  calling \n- [ ] Map API response fields to \n- [ ] Handle 401 (auth_expired), 429 (rate_limit with Retry-After), 5xx (upstream) errors\n- [ ] Implement cursor logic: store last  Unix ms as cursor\n- [ ] Leave  as passthrough for now\n- [ ] Write unit tests using httptest.Server\n- [ ] Register plugin in \n- [ ] Run tests and verify compilation\n\nImplementation:
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

## Summary of Changes

Implemented the Spotify OAuth API sync plugin (`spotify-api`) in `internal/plugins/spotify/api.go`:

- **New `APIPlugin` struct** with `NewAPI()` and `NewAPIWithBaseURL()` constructors
- **Name**: `spotify-api`, **AuthType**: `AuthOAuth`, **AuthConfig**: Spotify OAuth with `user-read-recently-played` scope
- **Sync()**: Calls `GET /me/player/recently-played?limit=50&after={cursor}`
  - Maps track.id, track.name, artists, duration_ms, played_at, album, popularity to `core.MediaItem`
  - Cursor: Unix ms timestamp of the latest `played_at`
  - HasMore: true when Spotify returns a `next` URL
- **Error handling**: 401 → `auth_expired`, 403 → `permission_denied`, 429 → `rate_limit` (with Retry-After parsing, defaults to 30s), 5xx → `upstream`, bad JSON → `invalid_data`
- **Enrich()**: Passthrough (no platform enrichment yet)
- **22 unit tests** covering: success, cursor forwarding, HasMore, auth expired, no token, rate limit, rate limit without header, forbidden, server error, bad gateway, invalid JSON, empty items, multiple artists, URL fallback, unexpected status, helper functions
- **Registered** in `cmd/web/main.go` alongside existing `spotify` (file import) plugin
- **OAuth callback route** added: `GET /api/v1/plugins/{plugin}/callback`
- **Env vars**: `SPOTIFY_CLIENT_ID`, `SPOTIFY_CLIENT_SECRET`, `BASE_URL` documented in `mise.toml`
- All 37 spotify tests pass (22 new + 15 existing), full project builds clean
