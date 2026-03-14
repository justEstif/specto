---
# project-media-consumption-analysis-2vk5
title: Implement Genkit LLM enricher
status: todo
type: feature
priority: high
created_at: 2026-03-14T03:52:54Z
updated_at: 2026-03-14T03:52:54Z
parent: project-media-consumption-analysis-eo0f
blocked_by:
    - project-media-consumption-analysis-2rhd
---

Replace NoOpEnricher with a real Genkit-based LLM enricher implementation.

## Tasks

- [ ] Add Genkit dependency and provider plugin (googlegenai + ollama)
- [ ] Implement Enricher interface with Genkit GenerateData[TagResult]
- [ ] Build classification prompt (item metadata + existing tags → genre/topic/mood/format)
- [ ] Validate returned tags against fixed tag set, drop hallucinated tags
- [ ] Resolve tag aliases before persisting
- [ ] Store tags with confidence scores (default 0.8 for LLM tags)
- [ ] Support batch classification (multiple items per prompt)
- [ ] Add provider configuration (provider, model, API key, batch_size, max_concurrent)
- [ ] Wire into SyncService replacing NoOpEnricher
- [ ] Tests with mock LLM responses

## Reference

See docs/enrichment.md — Core LLM Enricher section for prompt design, structured output shape, and provider config.
