---
# project-media-consumption-analysis-ns49
title: Create docs/schema.md
status: completed
type: task
priority: high
created_at: 2026-03-10T22:14:24Z
updated_at: 2026-03-11T11:56:43Z
---

Full data model: MediaItem plus supporting tables (users, plugin configs, sync state, tags). Define raw_metadata jsonb conventions. Include migration strategy.

## Summary of Changes\n\nCreated docs/schema.md covering:\n- 8 tables: users, plugin_states, plugin_credentials, media_items, tags, tag_aliases, media_item_tags, sync_log\n- Two-layer auth: app login (Google/GitHub OAuth) + per-plugin OAuth credentials (encrypted at rest)\n- Plugin state separated from sync log (current state vs audit trail)\n- Normalized tag system with aliases, categories, source tracking, and LLM confidence scores\n- Dedup via (user_id, platform, external_id)\n- Common query patterns (timeline, topic distribution, platform breakdown)\n- Migration strategy using goose\n- AES-256-GCM encryption for plugin credentials
