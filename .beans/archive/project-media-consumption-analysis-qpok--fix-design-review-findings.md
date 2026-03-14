---
# project-media-consumption-analysis-qpok
title: Fix design review findings
status: completed
type: task
priority: normal
created_at: 2026-03-13T20:09:09Z
updated_at: 2026-03-13T20:15:52Z
---

Address high and medium severity findings from software design review:
- [x] Extract shared sync logic from SyncPlugin/SyncPluginWithFile
- [x] Collapse 6 settings handlers into 2 parameterized handlers
- [x] Move timeline filtering into store/query layer
- [x] Replace map[string]any with response structs in API handlers

## Summary of Changes

All 4 design review findings addressed:

1. **Syncer dedup**: Extracted `executeSyncFlow()` — shared pipeline for steps 4-11. `SyncPlugin` and `SyncPluginWithFile` now only differ in credential resolution (~80 lines removed).

2. **Settings handlers**: Collapsed 6 handlers (3 pages + 3 partials) into 2 parameterized handlers using `chi.URLParam(r, "tab")` and a dispatch map. Routes updated from 6 specific paths to 2 parameterized paths.

3. **Timeline filtering**: Added `ListMediaItemsFiltered` SQL query with `sqlc.narg` for optional platform/type/search params. Filtering now happens at the DB level instead of in Go. Removed `filterItems()` function.

4. **Response structs**: Added `responses.go` with typed structs for timeline, insights, sync history, and sync result responses. Replaced `map[string]any` in 6 handlers. Field names are now compile-time checked.
