---
# project-media-consumption-analysis-vxea
title: Enrichment observability & retry
status: todo
type: task
created_at: 2026-03-14T03:53:07Z
updated_at: 2026-03-14T03:53:07Z
parent: project-media-consumption-analysis-eo0f
blocked_by:
    - project-media-consumption-analysis-2rhd
---

Add observability and retry mechanisms for the enrichment pipeline.

## Tasks

- [ ] Track enrichment status per item (pending → plugin-enriched → enriched / failed)
- [ ] Retry failed enrichments (background job or on-demand)
- [ ] Per-source error logging with rate-limit backoff tracking
- [ ] Enrichment metrics: items enriched, tags assigned, LLM calls, failures, latency
- [ ] Re-enrichment support (when tag set expands or prompts change)

## Reference

See docs/enrichment.md — Error Handling section.
