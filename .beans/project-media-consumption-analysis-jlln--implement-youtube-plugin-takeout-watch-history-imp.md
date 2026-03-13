---
# project-media-consumption-analysis-jlln
title: Implement YouTube plugin — Takeout watch history import
status: completed
type: task
priority: high
created_at: 2026-03-13T12:52:46Z
updated_at: 2026-03-13T13:07:24Z
parent: project-media-consumption-analysis-6ksu
---

Implement YouTube watch history import from Google Takeout JSON export. This is the primary data source since the YouTube API does not expose watch history.

Implementation:
- Create internal/plugins/youtube/ package
- Parse watch-history.json (array of objects)
- Map fields: titleUrl (parse v= param) → external_id, title (strip 'Watched ' prefix) → title, subtitles[0].name → creator, time → consumed_at
- Set platform='youtube', type='video'
- Handle edge cases: deleted videos ('Watched a video that has been removed' — no video ID), missing subtitles array, YouTube Music entries (header='YouTube Music')
- AuthType: AuthFileImport, cursor ignored (always full import)
- Store channel ID (from subtitles[0].url), products, header in raw_metadata
- Register plugin in main.go
- Unit tests with mock Takeout JSON

Does NOT require OAuth infrastructure — purely file-based.

## Summary of Changes\n\nCreated the YouTube Takeout watch history import plugin:\n\n### Files created:\n- `internal/plugins/youtube/plugin.go` — Full SourcePlugin implementation\n- `internal/plugins/youtube/plugin_test.go` — 14 comprehensive tests\n\n### Files modified:\n- `cmd/web/main.go` — Added youtube import and plugin registration\n\n### Implementation details:\n- Parses Google Takeout watch-history.json (top-level JSON array)\n- Maps all fields: title (strips 'Watched ' prefix), creator (from subtitles), video ID (from URL query param), consumed time (RFC3339Nano)\n- Determines MediaType: YouTube Music → music, YouTube → video\n- Strips ' - Topic' suffix from YouTube Music auto-generated channel names\n- Skips deleted videos, ads (From Google Ads), and entries without titleUrl\n- Stores header, products, channel_url, channel_id in RawMetadata\n- All 14 tests pass

## Summary of Changes

- Created `internal/plugins/youtube/plugin.go` — YouTube Google Takeout watch history JSON import plugin
- Created `internal/plugins/youtube/plugin_test.go` — 14 tests covering all field mappings and edge cases
- Registered plugin in `cmd/web/main.go`
- Handles deleted videos, ads, YouTube Music entries, missing subtitles, blank Topic creators
- Video ID extraction via net/url, channel ID extraction from URL path
