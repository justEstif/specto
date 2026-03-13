---
# project-media-consumption-analysis-k64o
title: End-to-end plugin integration tests
status: completed
type: task
priority: normal
created_at: 2026-03-13T12:53:03Z
updated_at: 2026-03-13T14:40:31Z
parent: project-media-consumption-analysis-6ksu
blocked_by:
    - project-media-consumption-analysis-gjpl
    - project-media-consumption-analysis-jlln
---

Write integration tests that exercise the full plugin lifecycle through the HTTP API layer using httpyac.

## Tasks

- [x] Write plugin listing/detail tests (list plugins, get plugin detail, get unknown plugin)
- [x] Write Spotify GDPR file import lifecycle test (register → import → verify timeline → check sync history)
- [x] Write YouTube Takeout file import lifecycle test (register → import → verify timeline → check sync history)
- [x] Write disconnect plugin test (disconnect → verify state updated)
- [x] Write malformed file import test (bad JSON → verify error)
- [x] Write unauthenticated access tests (all plugin endpoints without session)
- [x] Write insights endpoint tests (summary, platform-breakdown, tags, timeline)
- [x] Create test fixture files for Spotify and YouTube imports
- [x] Verify all tests pass with mise run api-test

Test scenarios:
- Register user → import Spotify GDPR file → verify timeline shows items → check sync history
- Register user → import YouTube Takeout file → verify timeline shows items → check sync history
- Connect Spotify via OAuth → trigger sync → verify items appear (requires mock OAuth server or test tokens)
- Disconnect plugin → verify credentials removed and state updated
- Import malformed file → verify appropriate error response
- Import duplicate data → verify deduplication (items_skipped count)

Update http/ test files with these scenarios. Run with 'mise run api-test'.
