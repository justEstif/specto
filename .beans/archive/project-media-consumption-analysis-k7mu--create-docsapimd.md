---
# project-media-consumption-analysis-k7mu
title: Create docs/api.md
status: completed
type: task
priority: low
created_at: 2026-03-10T22:14:24Z
updated_at: 2026-03-11T16:49:35Z
---

Internal API surface documentation. Create when frontend is decoupled or before M3.

## Summary of Changes

Created `docs/api.md` as the canonical internal HTTP API surface for the project. The doc:
- defines `/api/v1` as the JSON API base path
- separates HTML/browser routes from JSON client endpoints
- standardizes envelopes, error shapes, status codes, and pagination conventions
- documents plugin, session, timeline, insights, and share-profile endpoints
- chooses a plugin-centric route design to reduce cognitive load and change amplification
- calls out existing route examples in other docs as non-canonical if they diverge from `docs/api.md`
