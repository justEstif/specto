---
# project-media-consumption-analysis-4k3k
title: Implement software design review fixes
status: completed
type: task
priority: normal
created_at: 2026-03-12T21:25:26Z
updated_at: 2026-03-12T21:29:00Z
---

Three changes: 1) Refactor auth to use core.UserStore/UserInfo, make Handler depend on App 2) Remove global mutable state 3) Add GetByExternalID to MediaItemStore

## Summary of Changes

### 1. Auth layer refactored to use core domain types
- `internal/auth/auth.go`: Now uses `core.UserStore`/`core.UserInfo` instead of `database.Queries`/`database.User`. Auth logic wrapped in `auth.Service` struct with injected dependencies.
- `internal/auth/session.go`: Replaced global `SessionStore` with `SessionManager` struct. Uses `google/uuid.UUID` instead of `pgtype.UUID`.

### 2. Handler and middleware decoupled from database layer
- `internal/handlers/handler.go`: `Handler` now depends on `*app.App` instead of `*database.Queries`.
- `internal/handlers/auth.go`: Uses `app.Auth` service and `core.UserInfo` for responses.
- `internal/middleware/auth.go`: Uses `*auth.Service` instead of `*database.Queries`.

### 3. Global mutable state eliminated
- `internal/app/app.go`: Accepts `Config` struct instead of reading env vars. No more `error` return (config validated in main). Wires `auth.Service` and `SessionManager`.
- `cmd/web/main.go`: All env reads consolidated here. Passes `app.Config` to `app.New()`.

### 4. GetByExternalID added to MediaItemStore
- New SQL query `GetMediaItemByExternalID` in `queries.sql`, regenerated with sqlc.
- Added to `core.MediaItemStore` interface, `store.PgMediaItemStore`, `store.Querier`, and both test mocks.
- `syncer.go`: `enrichAndTagItems` now uses `GetByExternalID` instead of abusing `Create` as a lookup.

All tests pass. Build succeeds.
