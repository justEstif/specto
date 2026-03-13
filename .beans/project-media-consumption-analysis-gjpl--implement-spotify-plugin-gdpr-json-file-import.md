---
# project-media-consumption-analysis-gjpl
title: Implement Spotify plugin — GDPR JSON file import
status: completed
type: task
priority: high
created_at: 2026-03-13T12:52:23Z
updated_at: 2026-03-13T13:07:22Z
parent: project-media-consumption-analysis-6ksu
---

Implement the Spotify file import path using GDPR Extended Streaming History JSON export. This is the more valuable data source (complete lifetime history with ms_played).

Implementation:
- Create internal/plugins/spotify/ package
- Parse Streaming_History_Audio_*.json files
- Map fields: ts → consumed_at, master_metadata_track_name → title, master_metadata_album_artist_name → creator, spotify_track_uri → external_id, ms_played → time_spent
- Set platform='spotify', type='music'
- Store raw fields (shuffle, skipped, offline, platform, reason_start/end, album) in raw_metadata
- AuthType: AuthFileImport, cursor is ignored (always full import)
- Handle edge cases: missing track names (local files), zero ms_played entries
- Register plugin in main.go
- Unit tests with mock JSON data

Does NOT require OAuth infrastructure — purely file-based.

## Summary of Changes

- Created `internal/plugins/spotify/plugin.go` — Spotify GDPR Extended Streaming History JSON import plugin
- Created `internal/plugins/spotify/plugin_test.go` — 15 tests covering all field mappings and edge cases
- Registered plugin in `cmd/web/main.go`
- Handles music tracks, podcasts, local files (skipped), zero ms_played entries
- All 21 JSON fields parsed with proper nullable type handling
