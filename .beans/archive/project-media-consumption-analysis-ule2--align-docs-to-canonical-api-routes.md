---
# project-media-consumption-analysis-ule2
title: Align docs to canonical API routes
status: completed
type: task
priority: normal
created_at: 2026-03-11T16:59:25Z
updated_at: 2026-03-11T17:01:56Z
---

Update existing documentation to align route examples and API references with docs/api.md as the canonical client/server API surface.

Checklist:
- [x] Review docs/auth.md for route mismatches
- [x] Review docs/sharing.md for route mismatches
- [x] Review docs/plugin-guide.md for callback/API mismatch
- [x] Review docs/architecture.md for any stale route examples
- [x] Update docs to distinguish HTML routes from canonical JSON API routes
- [x] Add summary of aligned changes

## Summary of Changes

Aligned API-related documentation to treat `docs/api.md` as the canonical client/server surface. Specifically:
- updated `docs/auth.md` to separate HTML/navigation routes from `/api/v1` JSON routes
- updated plugin auth examples so connect starts at `/api/v1/plugins/{plugin}/connect` while OAuth callbacks remain server-owned
- updated file import and sync route examples to use `/api/v1/plugins/{plugin}/...`
- updated `docs/sharing.md` preview and privacy routes to use `/api/v1/share-profile/preview` and `/api/v1/items/{id}/privacy`
- updated `docs/plugin-guide.md` callback comments to match the canonical route split
- updated `docs/architecture.md` flow diagrams and cross-reference to `docs/api.md`
