---
# project-media-consumption-analysis-v5r1
title: Implement missing OAuth flows (Google/GitHub login, YouTube plugin, auto token refresh)
status: completed
type: feature
priority: normal
created_at: 2026-03-13T15:58:34Z
updated_at: 2026-03-13T16:03:55Z
parent: project-media-consumption-analysis-hj33
---

Fill gaps before manual testing: (1) Google OAuth app login, (2) GitHub OAuth app login, (3) YouTube OAuth plugin connection, (4) Wire auto token refresh into sync flow

## Tasks

- [x] 1. Google OAuth app login (handlers + routes)
- [ ] 2. GitHub OAuth app login (handlers + routes)
- [ ] 3. YouTube OAuth plugin (youtube-api plugin + env var wiring)
- [ ] 4. Wire auto token refresh into SyncService before calling plugin.Sync()
- [ ] 5. Update mise.toml with placeholder env vars for all OAuth providers
- [ ] 6. Build and test compilation

## Summary of Changes

Implemented all 4 missing OAuth flows:

1. **Google OAuth app login** — handlers fetch Google userinfo, upsert user, create session
2. **GitHub OAuth app login** — handlers fetch GitHub user + emails, upsert user, create session
3. **YouTube API plugin** (youtube-api) — OAuth plugin using YouTube Data API v3 activities endpoint
4. **Auto token refresh** — SyncService.tryRefreshToken() proactively refreshes OAuth tokens before sync

New files: internal/handlers/oauth_login.go, internal/plugins/youtube/api.go
Modified: internal/auth/oauth.go, internal/core/syncer.go, internal/app/app.go, cmd/web/main.go, mise.toml
