---
# project-media-consumption-analysis-w8az
title: Implement sync orchestration
status: todo
type: task
priority: high
created_at: 2026-03-12T20:21:15Z
updated_at: 2026-03-12T20:21:15Z
parent: project-media-consumption-analysis-86vz
blocked_by:
    - project-media-consumption-analysis-6o1q
    - project-media-consumption-analysis-wshg
---

Build SyncPlugin(ctx, userID, pluginName) in internal/core/sync.go implementing the full sync flow from architecture.md: (1) rate-limit check (minimum interval between syncs per user/plugin), (2) look up plugin from registry, (3) load and decrypt credentials from store, (4) get cursor from plugin state, (5) call plugin.Sync(credentials, cursor), (6) deduplicate returned items against existing media_items, (7) store new items, (8) call enricher on new items, (9) persist tags, (10) update sync_log (success/failure/item counts), (11) update plugin_state cursor. Handle PluginError codes: auth_expired triggers token refresh + re-encrypt, rate_limited applies backoff, etc. Include unit tests with mock plugin/store.
