---
# project-media-consumption-analysis-474i
title: Netflix plugin (CSV file import)
status: completed
type: feature
priority: normal
created_at: 2026-03-15T01:24:49Z
updated_at: 2026-03-15T01:35:14Z
parent: project-media-consumption-analysis-geho
---

Implement Netflix SourcePlugin supporting both simple CSV and GDPR ViewingActivity.csv import. Parse title strings to extract show/season/episode. Filter trailers via Supplemental Video Type. Use title+date as ExternalID (no Netflix content IDs in export).

## Tasks
- [ ] Create internal/plugins/netflix/ package
- [ ] Implement Plugin struct with SourcePlugin interface
- [ ] Support simple CSV (Title, Date) parsing
- [ ] Support GDPR CSV (10 columns) parsing with duration, device, country
- [ ] Parse TV show titles (split on colon for series/season/episode)
- [ ] Filter out supplemental content (trailers, previews)
- [ ] Filter short durations (< 2 min) as accidental clicks
- [ ] Implement Enrich() as no-op (TMDB enrichment handled by existing provider)
- [ ] Write comprehensive tests with fixture data
- [ ] Register plugin in cmd/web/main.go
- [ ] Add import guide modal for Netflix

## Summary of Changes\nImplemented Netflix CSV import plugin with auto-detection of simple vs GDPR format, TV title parsing, trailer/short-duration filtering. 14 tests passing.
