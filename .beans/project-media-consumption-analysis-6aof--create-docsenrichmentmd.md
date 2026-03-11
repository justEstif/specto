---
# project-media-consumption-analysis-6aof
title: Create docs/enrichment.md
status: completed
type: task
priority: normal
created_at: 2026-03-10T22:14:24Z
updated_at: 2026-03-11T12:19:34Z
---

Document enrichment pipeline: Last.fm integration, LLM classification prompts, tag taxonomy, handling conflicts between sources. Shapes the schema design.

## Summary of Changes\n\nCreated docs/enrichment.md covering:\n- Fixed 4-category tag taxonomy (genre, topic, mood, format) with open tags within\n- Enricher interface and chain ordering (authoritative → probabilistic)\n- 7 enrichment sources: Last.fm, MusicBrainz, TMDB, OMDB, Open Library, YouTube API, TikTok oEmbed\n- LLM enricher using go-ai SDK (OpenAI/Anthropic/Ollama) with structured JSON output\n- Prompt design, confidence scores, cost estimation\n- Pipeline execution with graceful failure handling\n- Tag alias resolution flow\n- Batch processing strategies per enricher\n- Configuration YAML structure
