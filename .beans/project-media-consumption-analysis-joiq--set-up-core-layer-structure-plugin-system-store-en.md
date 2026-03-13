---
# project-media-consumption-analysis-joiq
title: Set up core layer structure (plugin system, store, enrichment stubs)
status: completed
type: task
priority: normal
created_at: 2026-03-12T02:37:25Z
updated_at: 2026-03-13T12:42:32Z
parent: project-media-consumption-analysis-hj33
blocked_by:
    - project-media-consumption-analysis-w1nk
---

Create the internal package layout per docs/architecture.md: internal/core (plugin registry, store interface, enrichment pipeline stub), internal/server (will hold API routes). Wire up the store layer to the database package.

## Summary of Changes\n\nThis task was already completed as part of the core app epic work. The full core layer structure exists:\n\n- `internal/core/plugin.go` - SourcePlugin interface, AuthType, OAuthConfig, Credentials\n- `internal/core/registry.go` - Thread-safe PluginRegistry with Register/Get/List\n- `internal/core/stores.go` - All store interfaces (MediaItemStore, PluginStateStore, SyncLogStore, TagStore, UserStore, InsightsStore)\n- `internal/core/enrichment.go` - Enricher interface + NoOpEnricher stub\n- `internal/core/syncer.go` - SyncService orchestrator (11-step sync flow)\n- `internal/core/store/` - All pgx-backed store implementations with crypto and conversion\n- `internal/core/tags.go` - Fixed tag taxonomy (130 tags across 4 categories)\n- `internal/core/insights.go` - InsightsService with timeline, platform breakdown, tag distribution\n- `internal/app/app.go` - Full dependency wiring\n- Comprehensive test coverage for all packages
