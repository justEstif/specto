---
# project-media-consumption-analysis-775k
title: Rewrite seed data to span 2 years with era-friendly distribution
status: completed
type: task
priority: normal
created_at: 2026-03-15T00:55:45Z
updated_at: 2026-03-15T01:03:29Z
---

Rewrite buildMediaItems() and buildTagAssignments() in cmd/seed/main.go to generate ~185 items spanning 2 years with distinct taste eras for music (4 eras), video (3 eras), and podcast (2 eras).

## Summary of Changes

Rewrote buildMediaItems() and buildTagAssignments() in cmd/seed/main.go:

- 184 total items spanning ~2 years (730 days back from now)
  - 101 music items (73 spotify + 28 lastfm) across 4 distinct eras
  - 58 video items (youtube) across 3 distinct eras
  - 25 podcast items (youtube) across 2 distinct eras
- Music eras: Indie/Dreamy → Hip-Hop/Intense → Electronic/Energetic → R&B/Melancholic
- Video eras: Science/Math → Programming/Design → Philosophy/Contemplative
- Podcast eras: Tech/Business → Culture/Philosophy
- Each era spans 3-6 months with 19-25 items (well above MinItemsPerEra=15)
- Tags use only valid tags from the fixed tag set
- Tag sources mix api (confidence 0) and llm (confidence 0.7-0.95), 3-6 tags per item
- Consumed times vary across different hours/days for heatmap variety
- Compiles cleanly
