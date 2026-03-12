---
# project-media-consumption-analysis-6o1q
title: Build store/repository layer
status: completed
type: task
priority: high
created_at: 2026-03-12T20:21:01Z
updated_at: 2026-03-12T20:44:44Z
parent: project-media-consumption-analysis-86vz
blocked_by:
    - project-media-consumption-analysis-ymie
---

Create internal/core/store/ with repository interfaces and PostgreSQL implementations wrapping the sqlc-generated database.Queries. Includes: MediaItemStore (create with dedup, list, get, update enrichment status), PluginStateStore (get/upsert cursor, get/upsert credentials with encrypt/decrypt), SyncLogStore (begin/complete/fail sync entries), UserStore (wraps existing auth queries). Convert between domain MediaItem and database models. All methods accept context and operate within the caller's transaction boundaries where needed. Include unit/integration tests.

## Todo

- [x] Define repository interfaces (MediaItemStore, PluginStateStore, SyncLogStore, UserStore)
- [x] Implement UUID helper functions (google/uuid <-> pgtype.UUID)
- [x] Implement domain <-> database model conversion functions
- [x] Implement MediaItemStore (create with dedup, get, list, update enrichment status, list pending enrichment)
- [x] Implement PluginStateStore (get/upsert state, get/upsert/delete credentials with encrypt/decrypt, update synced)
- [x] Implement SyncLogStore (begin/complete/fail sync entries, list logs)
- [x] Implement UserStore (get by ID/email/auth, create, update profile)
- [x] Write unit tests for conversion functions and UUID helpers
- [x] Write unit tests for stores using mock/interface of Queries
- [x] Verify all code compiles with go build

## Summary of Changes

Implemented the full store/repository layer in `internal/core/store/`:

- **store.go**: Repository interfaces — `MediaItemStore`, `PluginStateStore`, `SyncLogStore`, `UserStore` — plus domain types `PluginStateInfo`, `SyncLogEntry`, `SyncLogResult`, `UserInfo`
- **queries.go**: `Querier` interface mirroring the 22 sqlc methods used by stores, with compile-time assertion that `database.Queries` satisfies it
- **convert.go**: Bidirectional conversion between domain types and database models, UUID helpers (`uuidToPgx`/`pgxToUUID`), pgtype helpers for Text, Timestamptz, Int4, Interval
- **media_item_store.go**: PostgreSQL implementation — create (with JSON metadata marshaling), get, list (time-range), update enrichment status, list pending enrichment
- **plugin_state_store.go**: PostgreSQL implementation — get/upsert state, update status/synced, list states, credential encrypt/decrypt round-trip using AES-256-GCM crypto layer
- **sync_log_store.go**: PostgreSQL implementation — begin (create running entry), complete, fail, list
- **user_store.go**: PostgreSQL implementation — get by ID/email/auth, create (OAuth), create with password, update profile

46 new unit tests (55 total including existing crypto tests), all passing. Mock Querier enables testing without a database.
