---
# project-media-consumption-analysis-ntba
title: Create docs/data-sources.md
status: completed
type: task
priority: high
created_at: 2026-03-10T22:14:24Z
updated_at: 2026-03-11T12:09:23Z
---

Per-platform research: exact API endpoints, OAuth scopes, rate limits, data format samples, known gotchas for Spotify, YouTube, Netflix, TikTok, etc.

## Summary of Changes\n\nCreated docs/plugins/ with 10 platform research docs via parallel subagents:\n\n| Platform | Access Method | MVP Priority | Difficulty |\n|----------|--------------|-------------|------------|\n| Spotify | OAuth API + GDPR export | Yes | Easy |\n| YouTube | Google Takeout + API enrichment | Yes | Medium |\n| Netflix | File import (GDPR CSV) | Yes | Medium |\n| TikTok | GDPR JSON export + oEmbed | No | Medium |\n| Apple Music | Privacy data export + MusicKit API | No | Hard |\n| Prime Video | Amazon data export / browser script | No | Hard |\n| Twitch | OAuth API (no watch history) | No | Medium |\n| Podcasts | Spotify API + platform-specific | Partial | Medium |\n| Reddit | OAuth API (no view history) | No | Medium |\n| Goodreads | CSV export + Open Library | Yes | Easy |
