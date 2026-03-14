---
# project-media-consumption-analysis-gsai
title: Create initial database schema & migrations
status: completed
type: task
priority: high
created_at: 2026-03-12T02:37:25Z
updated_at: 2026-03-12T02:49:29Z
parent: project-media-consumption-analysis-hj33
blocked_by:
    - project-media-consumption-analysis-w1nk
---

Replace the sample users migration with the actual specto schema from docs/schema.md. Tables: users, connected_accounts, media_items, enrichments, sync_logs. Run sqlc generate after.

## Summary of Changes

- Created 8 migration pairs (up/down) matching docs/schema.md exactly:
  - 001: users (with auth_provider/auth_subject unique constraint)
  - 002: plugin_states (with status transitions)
  - 003: plugin_credentials (encrypted OAuth tokens)
  - 004: media_items (with FTS index, dedup key, enrichment partial index)
  - 005: tags (shared taxonomy)
  - 006: tag_aliases (variant spelling normalization)
  - 007: media_item_tags (join table with source + confidence)
  - 008: sync_log (append-only audit trail)
- Created comprehensive sqlc queries for all core operations
- Generated type-safe Go code via sqlc
- All code compiles cleanly
