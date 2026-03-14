---
# project-media-consumption-analysis-ym6u
title: OAuth token exchange and refresh infrastructure
status: completed
type: feature
priority: normal
created_at: 2026-03-13T13:00:28Z
updated_at: 2026-03-13T13:06:46Z
---

Create OAuthService for token exchange/refresh, update ConnectPlugin handler, add OAuthCallback handler, wire callback route, write tests.

## Tasks
- [x] Read existing files to understand current state
- [x] Add BaseURL and OAuthClients to app.Config and App struct
- [x] Create internal/auth/oauth.go with OAuthService
- [x] Update ConnectPlugin handler with real OAuth URL building
- [x] Add OAuthCallback handler
- [x] Wire callback route in cmd/web/main.go
- [x] Write tests for OAuthService (oauth_test.go)
- [x] Write tests for handlers (plugins_test.go)
- [x] Verify all tests pass

## Summary of Changes

Implemented the complete OAuth token exchange and refresh infrastructure:

### New files
- `internal/auth/oauth.go` — OAuthService with BuildAuthURL, ExchangeCode, RefreshToken, and GenerateState
- `internal/auth/oauth_test.go` — 13 tests covering all OAuth service methods and error cases

### Modified files
- `internal/auth/session.go` — Added SetOAuthState/GetOAuthState for CSRF state storage in sessions
- `internal/app/app.go` — Added OAuthClientConfig type, BaseURL/OAuthClients to Config, OAuth field to App struct, wiring in New()
- `internal/handlers/plugins.go` — Replaced ConnectPlugin stub with real OAuth URL building; added OAuthCallback handler
- `internal/handlers/plugins_test.go` — Updated TestConnectPluginOAuth to verify full URL params; added 7 new callback tests
- `cmd/web/main.go` — Added BASE_URL/SPOTIFY_CLIENT_ID/etc env loading, OAuthClients config, callback route

### Test results
- `internal/auth/`: 13 tests pass (all new)
- `internal/handlers/`: 27 tests pass (7 new OAuth tests + all existing tests still pass)
- Full project: all tests pass
