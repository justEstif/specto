---
# project-media-consumption-analysis-wshg
title: Implement plugin registry
status: completed
type: task
priority: high
created_at: 2026-03-12T20:20:51Z
updated_at: 2026-03-12T20:33:18Z
parent: project-media-consumption-analysis-86vz
blocked_by:
    - project-media-consumption-analysis-3vdi
---

Build the PluginRegistry in internal/core/registry.go: NewPluginRegistry(), Register(plugin SourcePlugin), Get(name string), List(). Enforce name uniqueness, validate OAuthConfig for OAuth-type plugins (non-empty client ID, redirect URL, scopes). Plugins register at init-time. Include unit tests.

\n## Todo\n\n- [x] Create internal/core/registry.go with PluginRegistry struct\n- [x] Implement Register() with name uniqueness and OAuthConfig validation\n- [x] Implement Get() and List()\n- [x] Add unit tests\n- [x] Verify build passes

## Summary of Changes

Created `internal/core/registry.go` with:

- `PluginRegistry` struct (thread-safe via sync.RWMutex)
- `NewPluginRegistry()` constructor
- `Register()` with name uniqueness enforcement and OAuthConfig validation
- `Get()` lookup by name
- `List()` returns sorted plugin names

12 unit tests in `registry_test.go` covering happy path, duplicates, empty names, all OAuth validation branches, and sorted listing.
