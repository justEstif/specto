---
# project-media-consumption-analysis-jj5m
title: Implement TMDB enrichment provider
status: completed
type: feature
priority: normal
created_at: 2026-03-14T17:51:09Z
updated_at: 2026-03-14T17:58:24Z
parent: project-media-consumption-analysis-eo0f
---

Create the TMDB enrichment provider that implements the EnrichmentProvider interface for movie and TV items. Includes TMDB API client, genre/keyword mapping, format tagging, tests, and wiring into app.go and main.go.

## Tasks

- [x] Read existing codebase: core types, plugin interface, tags, errors, app wiring
- [x] Create internal/plugins/tmdb/provider.go with TMDB client and enrichment logic
- [x] Create internal/plugins/tmdb/provider_test.go with comprehensive tests
- [x] Wire TMDB provider into app.go Config and coordinator
- [x] Wire TMDB_API_KEY env var in cmd/web/main.go
- [x] Verify go build and go test pass

## Summary of Changes

Implemented the TMDB enrichment provider for movie and TV items.

### Files Created
- `internal/plugins/tmdb/provider.go` — TMDB client, genre/keyword mapping, format tagging, search/details API
- `internal/plugins/tmdb/provider_test.go` — 22 tests covering all enrichment scenarios

### Files Modified
- `internal/app/app.go` — Added `TMDBAPIKey` to Config, wired tmdb.New() into providers slice
- `cmd/web/main.go` — Read `TMDB_API_KEY` env var and pass to app.Config

### All 22 tests pass, all packages build cleanly.
