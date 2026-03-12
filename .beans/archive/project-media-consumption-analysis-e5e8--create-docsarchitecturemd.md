---
# project-media-consumption-analysis-e5e8
title: Create docs/architecture.md
status: completed
type: task
priority: high
created_at: 2026-03-10T22:14:16Z
updated_at: 2026-03-11T11:11:17Z
---

Expand the MVP ASCII diagram into a proper architecture doc: component responsibilities, data flow, plugin registration/sync lifecycle, enrichment pipeline stages. North star document for the project.

## Summary of Changes\n\nCreated docs/architecture.md covering:\n- Deployment model (single-user self-hosted, deployable to Fly/Railway/Docker)\n- Three-layer system design (Client → Server → Core)\n- Plugin system and enrichment pipeline interfaces\n- Sync flow (user-triggered, rate-limited, inline enrichment decoupled in code)\n- File import flow using same Sync() interface\n- Auth architecture (app-level + per-plugin OAuth)\n- Proposed project structure\n- Key design decisions table
