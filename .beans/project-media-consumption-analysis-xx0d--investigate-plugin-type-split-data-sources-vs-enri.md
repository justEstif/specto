---
# project-media-consumption-analysis-xx0d
title: 'Investigate plugin type split: data sources vs enrichment providers'
status: todo
type: task
priority: normal
created_at: 2026-03-13T21:32:33Z
updated_at: 2026-03-13T21:32:33Z
---

Some plugins (like youtube-api) don't actually provide watch history — their real value is enrichment (duration, tags, stats). Consider splitting plugin types into:

1. **Data Source plugins** — import/sync what the user actually consumed (file imports, polling APIs with real history access)
2. **Enrichment plugins** — add metadata to existing items (YouTube API for duration/tags, Last.fm/MusicBrainz for music, TMDB for video, LLMs for classification)

## Questions to investigate

- [ ] Should youtube-api Sync be dropped entirely? It fetches activities, not watch history
- [ ] Which current/planned OAuth connections are really just enrichment sources?
- [ ] Should enrichment plugins have a different interface than SourcePlugin?
- [ ] How does this relate to the existing two-layer enrichment architecture (plugin enrichment vs core/LLM enrichment)?
- [ ] Could auth credentials obtained for enrichment also serve data-fetching where available (e.g. spotify-api polling)?
- [ ] What would the SourcePlugin vs EnrichmentPlugin interfaces look like?

## Context

Current SourcePlugin interface bundles both Sync() and Enrich() together. YouTube API is the clearest example of the mismatch — its Sync returns channel activities (not history), but its Enrich is the only plugin that actually adds real metadata (duration, tags, view counts).
