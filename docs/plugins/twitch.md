# Twitch Plugin

## Overview
Twitch provides a well-documented Helix API with OAuth 2.0 authentication. We can retrieve **followed channels**, **VODs/videos** (by broadcaster), and **clips** (by broadcaster or game). However, Twitch **does not expose a user's personal watch history** — there is no endpoint to see which streams or VODs a user has actually viewed. The best proxy is the user's followed channels list combined with polling live streams and fetching VODs from those channels.

## Access Method
- **Primary**: OAuth 2.0 API (Twitch Helix)
- **Fallback**: None (no data export or GDPR download for watch history)

## API Details

- **Base URL**: `https://api.twitch.tv/helix`
- **Auth**: OAuth 2.0 (Authorization Code flow for user tokens). Every request requires both `Authorization: Bearer <token>` and `Client-Id: <client_id>` headers.

### Key Endpoints

| Endpoint | Method | Description | Auth | Pagination |
|---|---|---|---|---|
| `/users` | GET | Get user profile (id, login, display_name, profile_image_url, created_at) | App or User token | N/A |
| `/channels/followed` | GET | List channels the user follows (broadcaster_id, broadcaster_name, followed_at). Sorted by most recently followed. | User token, `user:read:follows` | Cursor-based, max 100/page |
| `/videos` | GET | Get videos by id, user_id, or game_id. Returns VODs (archive), highlights, uploads. Fields: id, title, description, created_at, published_at, url, duration, view_count, language, type, muted_segments. | App or User token | Cursor-based, max 100/page |
| `/clips` | GET | Get clips by broadcaster_id, game_id, or clip id. Filterable by started_at/ended_at. Max ~1000 results via pagination. | App or User token | Cursor-based, max 100/page |
| `/streams` | GET | Get currently live streams (by user_id, game_id, etc). Useful for polling followed channels' live status. | App or User token | Cursor-based, max 100/page |

### Rate Limits
- Token-bucket algorithm, 1 point per request (default), refills per minute
- Separate buckets for app access tokens and user access tokens
- User access token limits: per client ID, per user, per minute
- Response headers: `Ratelimit-Limit`, `Ratelimit-Remaining`, `Ratelimit-Reset`
- Exceeding limit returns HTTP 429

### Scopes Required
- `user:read:follows` — required for Get Followed Channels
- `user:read:email` — optional, to include email in Get Users response
- No scope needed for Get Videos, Get Clips, Get Streams (public data, app token works)

## Data Export
Twitch does not offer a data export or GDPR download that includes personal viewing/watch history.

## Available Data Fields

| Platform Field | MediaItem Field | Notes |
|---|---|---|
| `broadcaster_name` (follows) | `title` | Channel name as content source |
| `followed_at` | `consumed_at` | When user followed; proxy for interest, not viewing |
| Video `title` | `title` | VOD/highlight/upload title |
| Video `duration` | `duration` | ISO 8601 duration string (e.g., `3m21s`) |
| Video `created_at` | `consumed_at` | When VOD was created (not when user watched) |
| Video `type` | `mediaType` | `archive` (VOD), `highlight`, `upload` |
| Video `url` | `url` | Direct link to video |
| Video `language` | `language` | ISO 639-1 code |
| Video `view_count` | — | Total views, not user-specific |
| Video `user_name` | `creator` | Broadcaster who owns the video |
| Clip `title` | `title` | Clip title |
| Clip `duration` | `duration` | Clip duration in seconds |
| Clip `created_at` | `consumed_at` | When clip was created |
| Clip `broadcaster_name` | `creator` | Source channel |
| Clip `game_id` | `genre` | Can resolve to game name via Get Games |

## Gotchas & Limitations
- **No watch history API**: Twitch does not track or expose what a user has watched. There is no equivalent of YouTube's watch history. This is the biggest gap.
- **Follows ≠ watching**: Following a channel only indicates interest, not actual consumption.
- **VODs are broadcaster-scoped**: You can list a channel's VODs, but not "VODs I've watched." Must iterate over followed channels to approximate.
- **VOD retention**: Non-partner/affiliate VODs are deleted after 7-14 days. Partner/affiliate VODs last 60 days. Highlights persist indefinitely.
- **Clips pagination cap**: Get Clips returns ~1000 results max even with pagination. Use `started_at`/`ended_at` windows to work around this.
- **No duration tracking**: Even if we list VODs from followed channels, we cannot know how long the user actually watched.
- **EventSub for real-time**: `stream.online`/`stream.offline` events can track when followed channels go live, but not whether the user watches.

## Plugin Classification
- **Auth Type**: OAuth 2.0 (Authorization Code flow)
- **Sync Strategy**: Incremental (cursor-based pagination on follows; time-windowed on clips/VODs)
- **Difficulty**: Medium — API is clean and well-documented, but the absence of watch history data means we can only approximate consumption
- **MVP Priority**: No — without actual watch history, the data is low-signal for consumption tracking. Consider as a "connected accounts" feature showing followed channels rather than a true consumption plugin.
