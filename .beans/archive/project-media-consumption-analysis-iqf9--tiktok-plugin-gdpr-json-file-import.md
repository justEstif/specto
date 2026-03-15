---
# project-media-consumption-analysis-iqf9
title: TikTok plugin (GDPR JSON file import)
status: completed
type: feature
priority: normal
created_at: 2026-03-15T01:24:54Z
updated_at: 2026-03-15T01:35:14Z
parent: project-media-consumption-analysis-geho
---

Implement TikTok SourcePlugin parsing GDPR data export JSON. Watch history provides only timestamp + video URL per entry. Extract video ID from URL for ExternalID. Also parse Like List and Favorite Videos for metadata flags.

## Tasks
- [ ] Create internal/plugins/tiktok/ package
- [ ] Implement Plugin struct with SourcePlugin interface
- [ ] Parse user_data.json Activity.Video Browsing History.VideoList
- [ ] Extract video ID from tiktokv.com URL path
- [ ] Parse Like List and Favorite Videos, flag in RawMetadata
- [ ] Handle malformed JSON and missing sections gracefully
- [ ] Implement Enrich() as no-op (oEmbed enrichment deferred to background)
- [ ] Write comprehensive tests
- [ ] Register plugin in cmd/web/main.go
- [ ] Add import guide modal for TikTok

## Summary of Changes\nImplemented TikTok GDPR JSON import plugin with video ID extraction, like/favorite flag merging, graceful handling of missing sections. 13 tests passing.
