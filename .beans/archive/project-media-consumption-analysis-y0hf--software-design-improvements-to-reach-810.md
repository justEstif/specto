---
# project-media-consumption-analysis-y0hf
title: Software design improvements to reach 8/10
status: completed
type: task
priority: high
created_at: 2026-03-14T22:43:58Z
updated_at: 2026-03-14T22:56:06Z
---

Four refactors:
1. Split MediaItemStore into CRUD + EnrichmentStore
2. Extract shared block resolution from duplicate renderBlocks
3. Move OAuth HTTP calls from handlers to auth layer
4. Remove pass-through methods in InsightsService + InsightsStore

## Tasks
- [x] Split MediaItemStore into MediaItemStore (CRUD) and EnrichmentStore into MediaItemStore (CRUD) and EnrichmentStore
- [x] Extract shared block data resolution from share.go and share_page.go
- [x] Move OAuth provider HTTP calls from handlers/oauth_login.go to internal/auth/
- [x] Remove pass-through unfiltered methods in InsightsService and InsightsStore
- [x] Verify build and tests pass

## Summary of Changes

### 1. Split MediaItemStore into MediaItemStore (CRUD) + EnrichmentStore
- Extracted 7 enrichment-lifecycle methods into a new `EnrichmentStore` interface in `internal/core/stores.go`
- `MediaItemStore` now has 7 CRUD methods (was 14)
- `EnrichmentWorker` depends on `EnrichmentStore` instead of `MediaItemStore`
- `PgMediaItemStore` implements both interfaces (compile-time checked)
- `app.App` exposes both `MediaItems` and `Enrichment` fields
- Test mocks simplified: worker tests use `mockEnrichmentStore`, handler mocks no longer need enrichment stubs

### 2. Remove pass-through methods in InsightsService + InsightsStore
- Removed 3 unfiltered methods from `InsightsStore` interface (9 → 6 methods)
- Removed 4 pass-through wrapper methods from `InsightsService`
- All methods now accept `InsightsFilter`; callers pass `InsightsFilter{}` for unfiltered
- Store implementations use filtered SQL queries unconditionally (nil filters are no-ops)
- Updated all callers in handlers (insights, home, attention) and all test mocks

### 3. Move OAuth provider HTTP calls to auth package
- Moved `fetchGoogleUserInfo`, `fetchGithubUserInfo`, `fetchGithubPrimaryEmail` from `internal/handlers/oauth_login.go` to `internal/auth/oauth.go`
- Exported as `auth.FetchGoogleUserInfo`, `auth.FetchGithubUserInfo`
- Exported `auth.ProviderUserInfo` type (was handler-internal `oauthProviderUserInfo`)
- Handler now imports and delegates to auth functions

### 4. Extract shared block data resolution
- Introduced `resolvedBlock` type and `resolveBlocks` method in `internal/handlers/share.go`
- Both `renderBlocks` (JSON preview) and `renderPublicBlocks` (HTML template) now call `resolveBlocks` for data fetching
- Adding a new block type requires changes in one place (resolveBlocks) plus the output mappers
- Eliminated ~160 lines of duplicated store calls
