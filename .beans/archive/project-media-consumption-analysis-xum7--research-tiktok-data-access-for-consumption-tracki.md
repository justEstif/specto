---
# project-media-consumption-analysis-xum7
title: Research TikTok data access for consumption tracking plugin
status: completed
type: task
priority: normal
created_at: 2026-03-11T12:06:54Z
updated_at: 2026-03-11T12:07:00Z
---

Research TikTok GDPR/DSAR data export JSON format, fields in watch history, and API options (Data Portability API). Write plugin doc to docs/plugins/tiktok.md.

## Summary of Changes

Researched TikTok data access methods and wrote docs/plugins/tiktok.md covering:

- **GDPR/DSAR export**: Primary method. JSON format with Video Browsing History containing only `Date` + `VideoLink` per entry. Also includes Like List, Favorites, Share History.
- **Data Portability API**: OAuth-based alternative requiring TikTok app approval. User-initiated transfers.
- **Research API**: Not applicable (public video queries only, no personal watch history).
- **Enrichment strategy**: oEmbed endpoint for title/creator metadata since export is extremely sparse.
- **Key limitation**: No watch duration data available from any TikTok source.
- **Plugin classification**: FileImport, full re-import sync, medium difficulty, MVP priority.
