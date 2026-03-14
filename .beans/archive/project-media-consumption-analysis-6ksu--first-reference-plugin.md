---
# project-media-consumption-analysis-6ksu
title: First reference plugin
status: completed
type: epic
priority: normal
created_at: 2026-03-11T17:11:07Z
updated_at: 2026-03-13T14:57:06Z
parent: project-media-consumption-analysis-bja8
blocked_by:
    - project-media-consumption-analysis-j0cp
---

Implement Spotify and YouTube as the first two reference plugins to validate the plugin abstraction, sync lifecycle, storage model, and API behavior before scaling to more plugins.

Both plugins have two data paths:
- **File import** (no OAuth needed): Spotify GDPR JSON export, YouTube Google Takeout JSON
- **OAuth API** (needs OAuth infra): Spotify recently-played endpoint, YouTube Data API enrichment

## Execution order
1. OAuth token exchange infrastructure (shared prerequisite for API paths)
2. Spotify GDPR file import (can start immediately, no OAuth needed)
3. YouTube Takeout file import (can start immediately, no OAuth needed)
4. Spotify OAuth API sync (blocked by #1)
5. YouTube API enrichment (blocked by #3)
6. End-to-end integration tests (blocked by #2, #3)

## Summary of Changes\n\nAll 6 child tasks completed: OAuth token exchange, Spotify GDPR import, YouTube Takeout import, Spotify OAuth API sync, YouTube API enrichment, and end-to-end integration tests.
