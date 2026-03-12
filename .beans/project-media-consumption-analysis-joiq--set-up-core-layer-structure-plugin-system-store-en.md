---
# project-media-consumption-analysis-joiq
title: Set up core layer structure (plugin system, store, enrichment stubs)
status: todo
type: task
priority: normal
created_at: 2026-03-12T02:37:25Z
updated_at: 2026-03-12T02:37:29Z
parent: project-media-consumption-analysis-hj33
blocked_by:
    - project-media-consumption-analysis-w1nk
---

Create the internal package layout per docs/architecture.md: internal/core (plugin registry, store interface, enrichment pipeline stub), internal/server (will hold API routes). Wire up the store layer to the database package.
