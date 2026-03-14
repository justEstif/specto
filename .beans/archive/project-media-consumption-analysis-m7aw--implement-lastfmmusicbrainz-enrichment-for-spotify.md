---
# project-media-consumption-analysis-m7aw
title: Implement Last.fm/MusicBrainz enrichment for Spotify plugin
status: completed
type: feature
priority: normal
created_at: 2026-03-14T03:52:59Z
updated_at: 2026-03-14T17:59:26Z
parent: project-media-consumption-analysis-eo0f
blocked_by:
    - project-media-consumption-analysis-2rhd
---

Add real plugin enrichment to the Spotify plugin using Last.fm and MusicBrainz APIs.

## Tasks

- [ ] Implement Last.fm API client (track.getTopTags, artist.getTopTags)
- [ ] Implement MusicBrainz API client (recording genres, release-group genres)
- [ ] Dedupe strategy: fetch artist tags once, apply to all tracks by that artist
- [ ] Map freeform Last.fm/MusicBrainz tags to fixed tag set via fuzzy matching
- [ ] Respect rate limits (Last.fm: 5 req/s, MusicBrainz: 1 req/s)
- [ ] Wire into Spotify plugin Enrich() method (currently passthrough)
- [ ] Tests with recorded API responses

## Reference

See docs/enrichment.md — Last.fm and MusicBrainz sections.

## Summary of Changes\n\nImplemented Last.fm + MusicBrainz enrichment provider:\n- Created `internal/plugins/lastfm/provider.go` with three-phase enrichment (artist tags, track tags, MB genres)\n- 90+ tag alias mappings for freeform → fixed tag normalization\n- Rate limiting: 5 req/s (Last.fm), 1 req/s (MusicBrainz)\n- Artist dedup: tags fetched once per unique artist\n- 24 tests covering enrichment, errors, tag normalization, rate limiting\n- Wired into app.go (conditional on LASTFM_API_KEY)
