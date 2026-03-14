---
# project-media-consumption-analysis-3vdi
title: Define core domain types
status: completed
type: task
priority: high
created_at: 2026-03-12T20:20:46Z
updated_at: 2026-03-12T20:28:15Z
parent: project-media-consumption-analysis-86vz
---

Create internal/core/ package with all core domain types: SourcePlugin interface, AuthType enum (oauth, csv_upload, scrape), OAuthConfig, Credentials, MediaItem (domain-level, distinct from sqlc model), MediaType enum, SyncResult, PluginError, ErrorCode. These are the foundational Go types the rest of the core layer builds on.

## Todo\n\n- [x] Create internal/core/ package directory\n- [x] Implement plugin.go — SourcePlugin interface, AuthType, OAuthConfig, Credentials\n- [x] Implement media.go — MediaItem domain type, MediaType enum\n- [x] Implement sync.go types — SyncResult\n- [x] Implement errors.go — PluginError, ErrorCode constants\n- [x] Add unit tests\n- [x] Verify build passes

## Summary of Changes

Created `internal/core/` package with four files:

- **plugin.go** — `SourcePlugin` interface, `AuthType` enum with `String()`, `OAuthConfig`, `Credentials`
- **media.go** — `MediaItem` domain struct (distinct from sqlc model), `MediaType` enum with `Valid()`
- **sync.go** — `SyncResult` struct for plugin sync responses
- **errors.go** — `PluginError` with `Error()`/`Unwrap()`, `ErrorCode` constants (7 codes), `Valid()` method
- **core_test.go** — 15 tests covering all types, enums, error handling, interface compliance

All types match the contracts defined in docs/plugin-guide.md.
