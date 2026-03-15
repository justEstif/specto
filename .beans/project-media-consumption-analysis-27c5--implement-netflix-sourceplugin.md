---
# project-media-consumption-analysis-27c5
title: Implement Netflix SourcePlugin
status: completed
type: feature
priority: normal
created_at: 2026-03-15T01:29:45Z
updated_at: 2026-03-15T01:31:44Z
---

File-import plugin that parses Netflix CSV exports (simple + GDPR formats)

## Summary of Changes\n\nImplemented the Netflix SourcePlugin with full test coverage.\n\n### Files Created\n- `internal/plugins/netflix/plugin.go` — Plugin implementation\n- `internal/plugins/netflix/plugin_test.go` — 14 tests, all passing\n\n### Features\n- Auto-detects simple CSV (2-column) vs GDPR CSV (10-column) format by header inspection\n- Parses headers by name, not column index (handles reordered columns)\n- Filters supplemental video types (trailers/previews) in GDPR format\n- Filters durations < 2 minutes as accidental clicks in GDPR format\n- Parses TV titles (Show: Season X: Episode) into series/season/episode metadata\n- Parses HH:MM:SS duration strings\n- Stores device, country, profile_name, bookmark in RawMetadata\n- No-op Enrich implementation\n- Proper error handling for nil file, empty file, invalid CSV
