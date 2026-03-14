---
# project-media-consumption-analysis-2rhd
title: 'Investigation: Enrichment architecture & sequencing'
status: in-progress
type: task
priority: high
created_at: 2026-03-14T03:52:42Z
updated_at: 2026-03-14T14:48:50Z
parent: project-media-consumption-analysis-eo0f
---

Resolve key design questions before implementing enrichment:

## Questions to resolve

- [x] Plugin-type split: **Yes — split into SourcePlugin (import only) and EnrichmentProvider (enrich by media type/platform).** Multiple enrichers can run per item. YouTube API becomes an EnrichmentProvider, not a source. The existing core Enricher (LLM) becomes just another EnrichmentProvider. (absorbs bean xx0d)
- [ ] Enrichment trigger timing: sync-time (inline) vs async background job vs hybrid?
- [ ] How do plugin enrichment (Layer 1) and core/LLM enrichment (Layer 2) coordinate? What status transitions?
- [ ] Batch strategy: per-item vs batched calls for each enrichment source
- [ ] Error handling & retry: per-source backoff, partial failure, re-enrichment of failed items
- [ ] Should enrichment be opt-in per plugin or always-on?
- [ ] Config shape: how are enrichment API keys (Last.fm, TMDB) and LLM provider settings passed?

## Context

Two enrichment layers exist in the design:
1. **Plugin enrichment** — platform-specific APIs (Last.fm, TMDB, YouTube Data API) called by each plugin's Enrich() method. Adds authoritative tags (confidence NULL).
2. **Core/LLM enrichment** — universal Genkit-based classification. Fills gaps (mood, topic). Tags stored with confidence 0.0-1.0.

Currently only NoOpEnricher exists for core. YouTube API plugin has a real Enrich() impl. Spotify plugin Enrich() is a passthrough.

## Deliverable

A short decision doc (or updates to docs/enrichment.md) covering each question above with the chosen approach.

## Enrichment API Research Results

### Summary of Changes So Far
- Fixed: YouTube file-import plugin now uses `NewWithEnrich()` instead of `New()` in main.go, so Takeout-imported videos will get YouTube Data API enrichment (duration, tags, categories, stats)
- Cleaned up 12 duplicate test entries from the database (210 real YouTube items remain)

### Enrichment Services Research (by media type)

#### Music
| Service | Auth | Rate Limit | Best For | Tags |
|---------|------|------------|----------|------|
| **Last.fm** | API key (free) | ~5 req/s | Freeform tags (genre, mood), play counts, similar artists | Freeform folksonomy — needs mapping |
| **MusicBrainz** | None (User-Agent required) | 1 req/s strict | Gold-standard IDs, accurate durations, curated genres | Structured genres (newer, limited coverage) |
| **Discogs** | Token (free) | 60 req/min | Structured genre+style taxonomy, physical release data | ~15 genres + ~500 styles — best structured taxonomy |

#### Movies / TV Shows
| Service | Auth | Rate Limit | Best For | Tags |
|---------|------|------------|----------|------|
| **TMDB** | API key (free) | ~40 req/s | Primary source — genres, keywords, runtime, ratings, images | ~19 movie + ~16 TV structured genres + thousands of keywords |
| **OMDB** | API key (free) | 1,000/day | IMDB + RT + Metacritic ratings in one call | Comma-separated genres (from IMDB) |
| **TVMaze** | None | 2 req/s | TV-specific: episodes, schedules, zero-auth | ~27 structured genres, TV only |
| **Trakt** | Client ID (free) | ~1000/5min | Social/engagement data, cross-ref IDs (IMDB+TMDB) | Structured genres + rating distributions |

#### Books
| Service | Auth | Rate Limit | Best For | Tags |
|---------|------|------------|----------|------|
| **Open Library** | None | 1-3 req/s | Primary — massive coverage (30M editions), subjects, covers | Freeform subjects — needs mapping |
| **Google Books** | API key (free) | 1,000/day | Ratings, BISAC-like categories, descriptions | Semi-structured categories with "/" hierarchy |

#### Podcasts
| Service | Auth | Rate Limit | Best For | Tags |
|---------|------|------------|----------|------|
| **Podcast Index** | API key+secret (free) | Undocumented (~few/s) | Open, structured categories (from iTunes taxonomy), episode data | ~100 Apple-defined categories — excellent structure |
| **Apple Podcasts Search** | None | Undocumented | Quick lookup by name, Apple-specific IDs | Genre IDs (structured) |
| Listen Notes | API key | 300 req/month free | Skip — too restrictive for enrichment | — |

#### Anime / Manga
| Service | Auth | Rate Limit | Best For | Tags |
|---------|------|------------|----------|------|
| **AniList** (GraphQL) | None | 90 req/min | Best pick — curated genres, ranked tags with vote counts, relations | 19 genres + hundreds of community-ranked tags |
| Jikan (MAL wrapper) | None | 3 req/s | Broader coverage but less structured | MAL genres/demographics |

#### Games
| Service | Auth | Rate Limit | Best For | Tags |
|---------|------|------------|----------|------|
| **IGDB** | Twitch client credentials | 4 req/s | Best pick — genres + themes taxonomy, game modes, platforms | ~25 genres + ~20 themes — well structured |
| RAWG | API key (free) | 5 req/s | Large DB (800K+), freeform tags with vote counts | Freeform + structured genres |

### Recommended Primary Sources per Media Type
1. **Music**: Last.fm (tags) + MusicBrainz (IDs, duration validation)
2. **Movies**: TMDB (primary) + OMDB (ratings supplement)
3. **TV Shows**: TMDB (primary) + TVMaze (episode detail supplement)
4. **Books**: Open Library (primary) + Google Books (ratings supplement)
5. **Podcasts**: Podcast Index (primary)
6. **Anime/Manga**: AniList (primary)
7. **Games**: IGDB (primary)
8. **YouTube Videos**: YouTube Data API (already implemented)
