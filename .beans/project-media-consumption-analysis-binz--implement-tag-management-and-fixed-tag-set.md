---
# project-media-consumption-analysis-binz
title: Implement tag management and fixed tag set
status: todo
type: task
created_at: 2026-03-12T20:21:04Z
updated_at: 2026-03-12T20:21:04Z
parent: project-media-consumption-analysis-86vz
blocked_by:
    - project-media-consumption-analysis-3vdi
---

Define the canonical fixed tag set from docs/enrichment.md as Go data in internal/core/tags.go: genre tags (action, comedy, drama, etc.), topic tags, mood tags, format tags. Implement tag validation (reject unknown tags), tag alias resolution (look up tag_aliases before persisting), and get-or-create logic for the tags table. This is used by both the enrichment pipeline and sync orchestration. Include unit tests.
