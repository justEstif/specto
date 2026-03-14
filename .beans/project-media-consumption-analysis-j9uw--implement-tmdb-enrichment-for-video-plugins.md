---
# project-media-consumption-analysis-j9uw
title: Implement TMDB enrichment for video plugins
status: todo
type: feature
created_at: 2026-03-14T03:53:03Z
updated_at: 2026-03-14T03:53:03Z
parent: project-media-consumption-analysis-eo0f
blocked_by:
    - project-media-consumption-analysis-2rhd
---

Add TMDB-based plugin enrichment for video content (Netflix, Prime Video).

## Tasks

- [ ] Implement TMDB API client (search/movie, search/tv, movie/{id}, tv/{id})
- [ ] Match items by title (+year if available), store TMDB ID in raw_metadata
- [ ] Map TMDB genres/keywords to fixed tag set
- [ ] Respect rate limits (~40 req/10s)
- [ ] Optional: OMDB client for ratings data (IMDb, RT, Metacritic) in raw_metadata
- [ ] Tests with recorded API responses

## Reference

See docs/enrichment.md — TMDB and OMDB sections.
