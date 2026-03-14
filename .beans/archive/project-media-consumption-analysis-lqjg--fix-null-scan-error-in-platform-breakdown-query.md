---
# project-media-consumption-analysis-lqjg
title: Fix NULL scan error in platform breakdown query
status: completed
type: bug
priority: normal
created_at: 2026-03-13T20:53:59Z
updated_at: 2026-03-13T20:55:14Z
---

SUM(duration) returns NULL when no items have duration set (e.g. YouTube imports). Dashboard crashes with: cannot scan NULL into *int64. Fix: COALESCE the SUM to 0.

## Summary of Changes\n\nFixed the platform breakdown SQL query in `internal/database/queries.sql`. The `SUM(EXTRACT(EPOCH FROM duration))` returns NULL when no items have a duration (e.g. YouTube imports without duration data). Wrapped with `COALESCE(..., 0)` so it returns 0 instead of NULL. Regenerated sqlc code — the generated type is now `int64` (was crashing trying to scan NULL into `*int64`).
