---
# project-media-consumption-analysis-gqno
title: Implement OAuth token exchange and refresh flow
status: completed
type: task
priority: high
created_at: 2026-03-13T12:52:13Z
updated_at: 2026-03-13T13:07:29Z
parent: project-media-consumption-analysis-6ksu
---

Build the server-side OAuth infrastructure that both Spotify and YouTube plugins need. This includes:

- Complete the ConnectPlugin handler to generate state param, build full auth URL with redirect_uri and scopes
- Add OAuth callback handler (GET /connect/{plugin}/callback) that exchanges the authorization code for tokens
- Implement token refresh logic (access tokens expire — need automatic refresh using refresh_token)
- Store encrypted OAuth tokens via PluginStates.UpsertCredentials
- Update plugin state to 'connected' after successful token exchange

This is a prerequisite for any OAuth plugin's API sync path. The connect handler currently returns just the base auth URL — this task finishes the full flow.

## Summary of Changes

- Created `internal/auth/oauth.go` — OAuthService with BuildAuthURL, ExchangeCode, RefreshToken, GenerateState
- Created `internal/auth/oauth_test.go` — 13 tests
- Updated `internal/auth/session.go` — SetOAuthState/GetOAuthState for CSRF state management
- Updated `internal/app/app.go` — OAuthClientConfig, BaseURL/OAuthClients config, OAuth service wiring
- Updated `internal/handlers/plugins.go` — ConnectPlugin now builds full OAuth URL, added OAuthCallback handler
- Updated `internal/handlers/plugins_test.go` — 7 new OAuth-related tests
- Wired callback route GET /api/v1/plugins/{plugin}/callback in main.go
