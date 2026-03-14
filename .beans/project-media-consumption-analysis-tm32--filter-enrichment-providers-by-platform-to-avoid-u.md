---
# project-media-consumption-analysis-tm32
title: Filter enrichment providers by platform to avoid unnecessary API calls
status: completed
type: task
priority: normal
created_at: 2026-03-14T21:42:16Z
updated_at: 2026-03-14T21:43:23Z
---

AniList currently queries all video items (YouTube, Netflix, etc.) and TMDB queries anime items. Update Supports() methods to use platform filtering so providers only run on relevant items.

## Summary of Changes\n\nUpdated `Supports()` methods on enrichment providers to filter by platform:\n\n- **AniList**: Now only enriches items from anime/manga platforms (crunchyroll, funimation, hidive, animelab, vrv, mangadex, etc.). Previously ran on ALL video items.\n- **TMDB**: Now skips items from anime platforms (handled by AniList instead). Previously ran on ALL video items including anime.\n- **Last.fm**: No change needed — already filtered to music only.\n\nThis prevents unnecessary API calls that could trigger rate limiting or IP blocks.
