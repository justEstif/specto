---
# project-media-consumption-analysis-w8az
title: Implement sync orchestration
status: completed
type: task
priority: high
created_at: 2026-03-12T20:21:15Z
updated_at: 2026-03-12T21:10:15Z
parent: project-media-consumption-analysis-86vz
blocked_by:
    - project-media-consumption-analysis-6o1q
    - project-media-consumption-analysis-wshg
---

Build SyncPlugin(ctx, userID, pluginName) in internal/core/sync.go implementing the full sync flow from architecture.md: (1) rate-limit check (minimum interval between syncs per user/plugin), (2) look up plugin from registry, (3) load and decrypt credentials from store, (4) get cursor from plugin state, (5) call plugin.Sync(credentials, cursor), (6) deduplicate returned items against existing media_items, (7) store new items, (8) call enricher on new items, (9) persist tags, (10) update sync_log (success/failure/item counts), (11) update plugin_state cursor. Handle PluginError codes: auth_expired triggers token refresh + re-encrypt, rate_limited applies backoff, etc. Include unit tests with mock plugin/store.


## Todo

- [x] Refactor: move store domain types and interfaces to core package (eliminate import cycle)

- [x] Design SyncService struct with dependency injection (registry, stores, enricher, logger)
- [x] Implement rate-limit check (minimum 15-min interval between syncs per user/plugin)
- [x] Implement SyncPlugin() orchestration function (full 11-step sync flow)
- [x] Handle PluginError codes: auth_expired, rate_limited, partial_sync, upstream, etc.
- [x] Implement plugin enrichment step (call plugin.Enrich)
- [x] Implement core enrichment step (call Enricher.Enrich + tag persistence)
- [x] Implement sync log tracking (begin, complete/fail with counters)
- [x] Implement cursor management (load, save on success/partial)
- [x] Write comprehensive unit tests with mock plugin/stores
- [x] Run tests and verify compilation


## Summary of Changes

Implemented the full sync orchestration service in `internal/core/syncer.go` with the complete 11-step sync flow:

1. **SyncService struct** — dependency-injected service holding registry, stores, enricher, logger, and configurable min sync interval (default 15min)
2. **Rate limiting** — checks sync_log for running syncs and minimum interval enforcement
3. **Full SyncPlugin() flow** — plugin lookup → credential decrypt → cursor load → plugin.Sync() → store items → plugin.Enrich() → core enrichment → tag persistence → sync log → cursor save
4. **Error handling** — all 7 PluginError codes handled: auth_expired marks plugin disconnected, rate_limited returns RateLimitError with retry hint, partial_sync stores items + saves cursor + logs partial, upstream/invalid_data/permission_denied/file_parse_error log and fail
5. **RateLimitError type** — structured error with Plugin, RetryAfter, and Reason fields
6. **Refactored store interfaces** — moved all domain types (PluginStateInfo, SyncLogEntry, SyncLogResult, MediaItemTagInfo, UserInfo) and store interfaces (MediaItemStore, PluginStateStore, SyncLogStore, TagStore, UserStore) from `internal/core/store/` to `internal/core/stores.go` to eliminate import cycle. Store implementations now reference `core.X` types.
7. **22 unit tests** covering happy path, empty results, rate limiting (too soon + in progress), first sync, cursor management, all error codes, plugin enrichment failure (non-fatal), core enrichment with tag persistence, tag validation filtering, store item failures, credential failures, and constructor defaults

Files created:
- `internal/core/syncer.go` — SyncService, SyncSummary, RateLimitError
- `internal/core/stores.go` — store interfaces and domain types (moved from store package)
- `internal/core/syncer_test.go` — 22 tests with mock implementations

Files modified (refactor):
- `internal/core/store/store.go` — removed moved types/interfaces (now just package doc)
- `internal/core/store/tag_store.go` — removed TagStore interface and MediaItemTagInfo
- `internal/core/store/convert.go` — updated to use core.X types
- `internal/core/store/sync_log_store.go` — updated to use core.X types
- `internal/core/store/plugin_state_store.go` — updated to use core.X types
- `internal/core/store/user_store.go` — updated to use core.X types
- `internal/core/store/sync_log_store_test.go` — updated to use core.X types
