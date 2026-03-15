---
# project-media-consumption-analysis-pajy
title: Add Netflix, TikTok, Goodreads, AniList seed data
status: completed
type: task
priority: normal
created_at: 2026-03-15T01:53:12Z
updated_at: 2026-03-15T01:56:11Z
---

Add ~65 new media items and tag assignments for 4 new platforms to the seed script

## Summary of Changes

Updated cmd/seed/main.go to add seed data for 4 new platforms:

- Plugin states: Added netflix, tiktok, goodreads, anilist to the plugin state loop and sync logs loop
- Netflix: 20 items (type video) - Breaking Bad, Stranger Things, The Crown, Black Mirror, Dark, Squid Game, etc.
- TikTok: 15 items (type video) - short 1-3 minute clips with creator handles and engagement metadata
- Goodreads: 15 items (type book) - Dune, The Stranger, Sapiens, Meditations, etc. with ISBN/pages/shelf metadata
- AniList: 15 items - 10 anime (type video) + 5 manga (type book) - Steins;Gate, Spirited Away, Berserk, etc.
- Tag assignments: 3-5 tags per item using genre, mood, topic, and format categories with llm source and 0.7-0.95 confidence
- All items spread across the existing 2-year timespan
- No existing items or logic modified
