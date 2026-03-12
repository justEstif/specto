---
# project-media-consumption-analysis-wshg
title: Implement plugin registry
status: todo
type: task
priority: high
created_at: 2026-03-12T20:20:51Z
updated_at: 2026-03-12T20:20:51Z
parent: project-media-consumption-analysis-86vz
blocked_by:
    - project-media-consumption-analysis-3vdi
---

Build the PluginRegistry in internal/core/registry.go: NewPluginRegistry(), Register(plugin SourcePlugin), Get(name string), List(). Enforce name uniqueness, validate OAuthConfig for OAuth-type plugins (non-empty client ID, redirect URL, scopes). Plugins register at init-time. Include unit tests.
