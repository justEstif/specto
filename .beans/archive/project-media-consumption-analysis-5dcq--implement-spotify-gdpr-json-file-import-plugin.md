---
# project-media-consumption-analysis-5dcq
title: Implement Spotify GDPR JSON file import plugin
status: completed
type: feature
priority: normal
created_at: 2026-03-13T12:59:26Z
updated_at: 2026-03-13T13:03:10Z
---

Create the Spotify file import plugin at internal/plugins/spotify/plugin.go with tests and register in main.go. The plugin parses Spotify Extended Streaming History JSON exports.

## Tasks
- [x] Create internal/plugins/spotify/plugin.go with SourcePlugin implementation
- [x] Create internal/plugins/spotify/plugin_test.go with comprehensive tests
- [x] Register plugin in cmd/web/main.go
- [x] Verify all tests pass

## Summary of Changes

Implemented the Spotify GDPR JSON file import plugin:

- **plugin.go**: Full SourcePlugin implementation that parses Extended Streaming History JSON arrays, maps music and podcast entries to MediaItems, skips unidentifiable local files, and stores rich metadata
- **plugin_test.go**: 15 tests covering all field mappings, edge cases (empty file, invalid JSON, empty array, nil file, mixed entries, zero ms_played, track with URI but no name, cursor ignored), and interface methods
- **main.go**: Registered spotify.New() in the plugin registry with error handling

All 15 tests pass.
