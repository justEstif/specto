---
# project-media-consumption-analysis-b3nr
title: Create docs/plugin-guide.md
status: completed
type: task
priority: high
created_at: 2026-03-10T22:14:24Z
updated_at: 2026-03-11T11:33:34Z
---

Document the SourcePlugin interface contract: lifecycle, auth flows (OAuth vs file import vs extension), error handling, rate limiting, testing plugins in isolation. Must be solid before M1 so M2-M6 don't force rewrites.

## Summary of Changes\n\nCreated docs/plugin-guide.md covering:\n- SourcePlugin interface (Sync, Enrich, AuthType, AuthConfig)\n- Core types: SyncResult with opaque cursor for incremental sync, Credentials, MediaItem, OAuthConfig\n- Normalized PluginError with 7 error codes and defined core behaviors\n- Partial sync flow with cursor-based resumption\n- Plugin lifecycle: compile-time registration, sync orchestration, deduplication\n- Full examples: Netflix CSV import plugin, Spotify OAuth plugin skeleton\n- Plugin checklist and testing patterns
