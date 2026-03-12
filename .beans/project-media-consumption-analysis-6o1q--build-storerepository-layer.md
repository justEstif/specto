---
# project-media-consumption-analysis-6o1q
title: Build store/repository layer
status: todo
type: task
priority: high
created_at: 2026-03-12T20:21:01Z
updated_at: 2026-03-12T20:21:01Z
parent: project-media-consumption-analysis-86vz
blocked_by:
    - project-media-consumption-analysis-ymie
---

Create internal/core/store/ with repository interfaces and PostgreSQL implementations wrapping the sqlc-generated database.Queries. Includes: MediaItemStore (create with dedup, list, get, update enrichment status), PluginStateStore (get/upsert cursor, get/upsert credentials with encrypt/decrypt), SyncLogStore (begin/complete/fail sync entries), UserStore (wraps existing auth queries). Convert between domain MediaItem and database models. All methods accept context and operate within the caller's transaction boundaries where needed. Include unit/integration tests.
