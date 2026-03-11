# Spotify Plugin

## Overview

Spotify's Web API provides good access to music consumption data including recently played tracks, saved library, and top artists/tracks. However, critical limitations exist: the recently-played endpoint only returns the **last 50 tracks** (no deep history), and as of November 2024, **audio features and recommendations endpoints are deprecated** for new apps. For full listening history, Spotify's GDPR data export is the only reliable source.

## Access Method
- **Primary**: OAuth 2.0 API (Authorization Code flow with PKCE)
- **Fallback**: GDPR Extended Streaming History export (JSON files)

## API Details

- **Base URL**: `https://api.spotify.com/v1`
- **Auth**: OAuth 2.0 Authorization Code with PKCE (recommended) or standard Authorization Code flow
- **Token Refresh**: Refresh tokens are long-lived; access tokens expire in 1 hour
- **Dev Mode Limitation**: Apps in development mode are limited to **25 allowlisted users**. Extended quota requires Spotify approval (criteria tightened May 2025).

### Key Endpoints

| Endpoint | Description | Pagination | Scope Required |
|---|---|---|---|
| `GET /me/player/recently-played` | Last 50 played tracks with timestamps | Cursor-based (`before`/`after` Unix ms timestamps), max `limit=50` | `user-read-recently-played` |
| `GET /me/top/artists` | User's top artists | Offset-based, `limit` max 50, `offset` max 49 | `user-top-read` |
| `GET /me/top/tracks` | User's top tracks | Offset-based, `limit` max 50, `offset` max 49 | `user-top-read` |
| `GET /me/tracks` | User's saved/liked tracks | Offset-based, `limit` max 50 | `user-library-read` |
| `GET /tracks/{id}` | Track details (name, artist, album, duration) | N/A | None (public) |
| `GET /artists/{id}` | Artist details (genres, popularity, images) | N/A | None (public) |
| `GET /audio-features/{id}` | ~~Danceability, energy, tempo, etc.~~ | N/A | **DEPRECATED (Nov 2024)** — returns 403 for new apps |

### Rate Limits

- Spotify does **not** publish exact rate limit numbers. Limits are per-app, not per-endpoint.
- Returns `429 Too Many Requests` with `Retry-After` header (seconds) when exceeded.
- General guidance: stay under ~180 requests/minute for sustained use.
- Some endpoints (e.g., playlist image upload) have stricter custom limits.

### Scopes Required

For our use case:
- `user-read-recently-played` — recently played tracks
- `user-top-read` — top artists and tracks
- `user-library-read` — saved/liked tracks
- `user-read-private` — account details (country, subscription tier)

## Data Export (GDPR)

Spotify offers two export tiers under GDPR/privacy settings:

1. **Account Data** (available in ~5 days): Basic account info + last 12 months of streaming history (partial data).
2. **Extended Streaming History** (takes up to 30 days): Complete lifetime listening history.

### How to request:
1. Go to [spotify.com/account/privacy](https://www.spotify.com/account/privacy)
2. Scroll to "Download your data"
3. Check "Extended streaming history" and submit
4. Wait for email with download link (up to 30 days)

### Format & Fields
- Format: JSON files (`Streaming_History_Audio_*.json`)
- Each entry includes:
  - `ts` — timestamp (UTC)
  - `master_metadata_track_name` — track title
  - `master_metadata_album_artist_name` — artist
  - `master_metadata_album_album_name` — album
  - `spotify_track_uri` — `spotify:track:{id}`
  - `ms_played` — milliseconds played
  - `reason_start` / `reason_end` — how playback started/ended
  - `shuffle`, `skipped`, `offline` — boolean flags
  - `platform` — device/OS used

### Limitations
- Can only request every ~few months
- Takes up to 30 days to receive
- No real-time sync — point-in-time snapshot only

## Available Data Fields

| Platform Field | MediaItem Field | Notes |
|---|---|---|
| `track.id` / `spotify_track_uri` | `external_id` | Spotify track ID |
| `track.name` / `master_metadata_track_name` | `title` | Track name |
| `track.artists[0].name` / `master_metadata_album_artist_name` | `creator` | Primary artist |
| `played_at` / `ts` | `consumed_at` | ISO 8601 timestamp |
| `track.duration_ms` | `duration` | Track length |
| `ms_played` | `time_spent` | Only in GDPR export; API doesn't provide this |
| `track.external_urls.spotify` | `url` | Spotify URL |
| — | `platform` | `"spotify"` (constant) |
| — | `type` | `"music"` (constant) |
| `track.album`, `track.popularity`, artist genres, `shuffle`, `skipped` | `raw_metadata` | Store as JSONB for enrichment |

## Gotchas & Limitations

- **Recently-played returns max 50 tracks** — no way to paginate deeper. Must poll frequently (e.g., every 1-2 hours) to avoid gaps.
- **Audio features/recommendations endpoints deprecated** (Nov 2024) for new apps. Only grandfathered apps retain access.
- **Top artists/tracks only support `time_range`**: `short_term` (4 weeks), `medium_term` (6 months), `long_term` (years) — no custom date ranges.
- **Top endpoint pagination capped** at offset 49 with limit 50, so max 99 items.
- **No play count** in the API — only the GDPR export reveals actual listen counts (via `ms_played`).
- **Dev mode = 25 users max**. Getting extended quota approval is non-trivial and Spotify has tightened criteria as of 2025/2026.
- **Podcasts not in recently-played** — the endpoint explicitly excludes podcast episodes.
- **No webhook/push** — must poll. No way to subscribe to play events.
- **Cursor pagination quirk**: `recently-played` uses Unix millisecond timestamps as cursors (`before`/`after`), not standard next/previous URLs.

## Plugin Classification

- **Auth Type**: OAuth 2.0 (Authorization Code with PKCE)
- **Sync Strategy**: Incremental (cursor-based polling for recently-played; store last `played_at` timestamp). Full re-import for GDPR export.
- **Difficulty**: Medium — OAuth flow is standard, but the 50-track limit requires frequent polling and the dev mode user cap complicates deployment.
- **MVP Priority**: Yes — music is a primary consumption category, Spotify is the dominant platform, and the API is well-documented despite limitations.
