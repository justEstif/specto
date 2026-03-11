# Apple Music Plugin

## Overview
Apple Music provides two data access paths: a REST API (Apple Music API / MusicKit) for recent listening history and catalog data, and a privacy data export via [privacy.apple.com](https://privacy.apple.com) for complete historical play data. The API only returns recently played items (capped at ~240 tracks with no timestamps), making the privacy export the primary source for full listening history.

## Access Method
- **Primary**: File Import (Apple Privacy Data Export)
- **Fallback**: OAuth-like API (Apple Music API with Developer Token + Music User Token)

## API Details

- **Base URL**: `https://api.music.apple.com`
- **Auth**: Developer Token (JWT signed with Apple private key) + Music User Token (obtained via MusicKit.js `authorize()` flow, lasts ~6 months)
- **Key Endpoints** (for consumption history):
  - `GET /v1/me/recent/played/tracks` — Returns recently played tracks (songs). Max `limit=30` per request, offset-based pagination. Returns song metadata (name, artist, album, artwork, duration, ISRC) but **no play timestamps or play counts**.
  - `GET /v1/me/recent/played` — Returns recently played resources (albums, stations, playlists). Max `limit=10`, offset-based pagination via `next` cursor. Also no timestamps.
  - `GET /v1/me/library/songs` — User's library songs. Not play history.
  - `GET /v1/catalog/{storefront}/songs/{id}` — Catalog lookup by ID.
- **Rate Limits**: Not officially documented. Returns `429 Too Many Requests` when exceeded. Community reports suggest moderate limits (~60 req/min range).
- **Scopes Required**: No formal OAuth scopes. Requires:
  1. Apple Developer Program membership ($99/yr)
  2. MusicKit API key (private key + key ID + team ID → JWT developer token)
  3. Music User Token (user must authenticate via MusicKit.js or MusicKit native SDK)

## Data Export (Apple Privacy Data Export)

- **How to export**:
  1. Go to [privacy.apple.com](https://privacy.apple.com) and sign in
  2. Select "Get a copy of your data"
  3. Check **"Apple Media Services Information"** only
  4. Choose max file size (default 1GB), click Continue
  5. Submit request — takes **up to 7 days** to prepare
  6. Download all ZIP parts when notified via email
- **Format**: ZIP archive containing CSV files
- **Key file**: `Apple Music Activity/Apple Music - Play History Daily Tracks.csv`
- **Fields included**: Track title, artist, album, play date/time, duration, genre, and more
- **Limitations**:
  - Up to 7 days processing time
  - Manual process (no programmatic access to privacy portal)
  - Cannot request more than once every few days
  - Export covers full account lifetime

## Available Data Fields

### From API (`/v1/me/recent/played/tracks`)

| Platform Field | MediaItem Field | Notes |
|---|---|---|
| `attributes.name` | `title` | Track name |
| `attributes.artistName` | `artist` | Primary artist |
| `attributes.albumName` | `album` | Album name |
| `attributes.durationInMillis` | `duration` | Duration in ms |
| `attributes.artwork.url` | `thumbnailUrl` | Template URL with `{w}` and `{h}` tokens |
| `attributes.genreNames` | `genres` | Array of genre strings |
| `attributes.isrc` | `isrc` | International Standard Recording Code |
| `attributes.releaseDate` | `releaseDate` | YYYY-MM-DD |
| `attributes.url` | `externalUrl` | Apple Music catalog URL |
| `id` | `platformId` | Apple Music catalog ID |
| _(not available)_ | `playedAt` | **Not provided by API** |

### From Privacy Export (CSV)

| Platform Field | MediaItem Field | Notes |
|---|---|---|
| Track name | `title` | — |
| Artist name | `artist` | — |
| Album name | `album` | — |
| Play date/time | `playedAt` | Actual timestamp available |
| Duration | `duration` | — |
| Genre | `genres` | — |
| Content type | `mediaType` | Song, music video, etc. |

## Gotchas & Limitations

- **API has no play timestamps**: The recently played endpoints return tracks in rough recency order but provide zero timestamp data. You cannot determine *when* a track was played.
- **API history is shallow**: Maximum ~240 recent tracks retrievable (30 per page × ~8 pages before data repeats/ends). No access to full history.
- **Developer token complexity**: Requires Apple Developer Program membership ($99/yr), generating a private key, and creating a JWT. Token expires (configurable, max 6 months).
- **Music User Token expires**: The user auth token lasts ~6 months, then user must re-authenticate via MusicKit.js browser flow. No refresh token mechanism.
- **No webhooks or real-time events**: Cannot subscribe to play events. Must poll.
- **Privacy export is manual and slow**: Up to 7 days wait, no API to automate it.
- **Privacy export CSV format may change**: Apple doesn't formally document the CSV schema; field names/structure can change between exports.
- **No play count per track**: Neither the API nor the export provides an aggregate play count — each row in the CSV is a single play event.

## Plugin Classification

- **Auth Type**: FileImport (primary), JWT + UserToken (fallback for recent plays)
- **Sync Strategy**: Full re-import (privacy export); no viable incremental strategy via API due to missing timestamps
- **Difficulty**: Hard — privacy export requires manual user action + 7-day wait; API auth chain is complex and provides limited data
- **MVP Priority**: Yes — Apple Music is a major streaming platform, but recommend **file import only** for MVP (privacy export CSV parsing). API integration can be deferred to post-MVP due to complexity and data limitations.
