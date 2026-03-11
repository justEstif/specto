# Amazon Prime Video Plugin

## Overview
Amazon Prime Video has **no public API** for accessing personal viewing history. Data can be obtained through three methods: Amazon's "Request My Data" export (official), GDPR/DSAR requests (official, EU/CA), or browser-based scraping of the viewing history page. The most practical approach for our plugin is file import from the data export, with an optional browser-script-assisted export as a faster alternative.

## Access Method
- **Primary**: File Import (Amazon "Request My Data" export)
- **Fallback**: Browser console script (scrapes viewing history page via internal API interception)

## Data Export

### Amazon "Request My Data" (Official)
1. Go to [amazon.com/hz/privacy-central/data-requests/preview.html](https://www.amazon.com/hz/privacy-central/data-requests/preview.html)
2. Select **"Prime Video"** from the category list (or request all data)
3. Submit request — Amazon sends a download link via email
4. **Delivery time**: Typically 1–5 days, can take up to 30 days
5. **Format**: ZIP archive containing CSV files
6. **Relevant file**: `Digital.PrimeVideo.ViewingHistory/Digital.PrimeVideo.ViewingHistory.csv`

#### Export Fields (CSV)
| CSV Column | Description |
|---|---|
| `Playback Hour` | Timestamp of when content was watched (hourly granularity) |
| `Title` | Full title string (includes show name, season, episode info for TV) |
| `Marketplace` | Regional marketplace (e.g., `Amazon.com`) |
| `Operating System` | Device OS used for playback |
| `Browser` | Browser or app used |
| `City` | Playback location city |
| `State` | Playback location state/region |
| `Country` | Playback location country |
| `ISP` | Internet service provider |
| `Device Type` | Device category (e.g., FireTV, Browser, Mobile) |

### GDPR / DSAR Request
- Same portal as above; EU users get GDPR-mandated response within 30 days
- May include additional data categories not in the standard export
- Same CSV format

### Browser Script Export (Community Tool)
The [watch-history-exporter-for-amazon-prime-video](https://github.com/twocaretcat/watch-history-exporter-for-amazon-prime-video) browser console script provides richer data by intercepting the internal Prime Video API:

1. Navigate to Prime Video → My Stuff → Watch History
2. Paste script into browser console
3. Script monkey-patches `fetch()` to capture internal API responses while auto-scrolling
4. Exports JSON or CSV

#### Browser Script Fields
| Field | Description |
|---|---|
| `dateWatched` | Full timestamp with millisecond precision |
| `type` | `Movie` or `Series` |
| `title` | Show/movie title (includes season for series) |
| `episodeTitle` | Episode name (series only) |
| `id` | Global Title Identifier (GTI) — `amzn1.dv.gti.*` |
| `episodeId` | Episode-level GTI (series only) |
| `path` | Prime Video detail page URL path |
| `episodePath` | Episode detail page URL path |
| `imageUrl` | Poster/thumbnail URL |

## Available Data Fields

### Mapping to MediaItem Schema (Browser Script — recommended)
| Platform Field | MediaItem Field | Notes |
|---|---|---|
| `title` | `title` | Needs parsing for series — contains "ShowName - Season X" |
| `episodeTitle` | `episodeTitle` | Empty for movies |
| `type` | `mediaType` | `Movie` or `Series` |
| `dateWatched` | `consumedAt` | Unix timestamp (ms) — full precision |
| `id` (GTI) | `platformId` | Amazon's unique content identifier |
| `episodeId` | `platformEpisodeId` | Unique per episode |
| `imageUrl` | `thumbnailUrl` | Direct CDN URL |
| `path` | `platformUrl` | Prefix with `https://www.amazon.com/gp/video` |

### Mapping to MediaItem Schema (Official Data Export)
| Platform Field | MediaItem Field | Notes |
|---|---|---|
| `Title` | `title` | Raw string, needs parsing for show/season/episode |
| `Playback Hour` | `consumedAt` | Hourly granularity only — no minute/second precision |
| — | `mediaType` | Must be inferred from title string parsing |
| — | `platformId` | **Not included** in official export |
| `Device Type` | `device` | Optional metadata |

## Gotchas & Limitations
- **No public API exists** — Amazon does not offer OAuth or developer API for viewing history
- **Official export has hourly granularity only** — `Playback Hour` rounds to the hour, so multiple views in the same hour are ambiguous
- **No content IDs in official export** — Title string is the only identifier; must fuzzy-match to deduplicate or cross-reference with external databases (TMDb/IMDb)
- **Title parsing is fragile** — Format varies by locale and content type (e.g., "The Boys - Season 3" vs "The Boys Season 3 Episode 5: ...")
- **Export delivery is slow** — 1–30 days wait, no programmatic trigger
- **Browser script requires manual user action** — Cannot be automated server-side; user must paste script in console
- **Watch history page requires authentication** — No way to access without active session cookies
- **Marketplace segmentation** — Users with multiple regional accounts have separate histories
- **No duration/progress data** — Neither method provides watch duration or completion percentage
- **Rate limiting on internal API** — Browser script auto-scrolling may hit throttling on very large histories

## Plugin Classification
- **Auth Type**: FileImport (user uploads CSV/JSON from data export or browser script)
- **Sync Strategy**: Full re-import (no incremental sync possible without API)
- **Difficulty**: Medium (title parsing and deduplication are the main challenges)
- **MVP Priority**: Yes — Prime Video is a major streaming platform; file import is straightforward to implement even without an API
