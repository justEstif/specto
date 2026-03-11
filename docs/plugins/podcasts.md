# Podcasts Plugin (Multi-Platform)

## Overview

Podcast listening data is fragmented across platforms with no unified API. **Spotify** is the best option — its Web API exposes saved episodes and playback position. **Overcast** offers an extended OPML export with episode-level data. **Pocket Casts** supports OPML subscription export but not listening history via export; history access requires unofficial API usage or GDPR request. **Apple Podcasts** has no API and no built-in export — only Apple's privacy data request works. **Google Podcasts** shut down April 2024; only OPML subscription export was available before shutdown.

## Platform Breakdown

---

### 1. Spotify (Podcasts via Web API)

#### Access Method
- **Primary**: OAuth 2.0 API (same auth as music — podcast scopes bundled)
- **Fallback**: GDPR Extended Streaming History export (includes podcast episodes)

#### API Details

- **Base URL**: `https://api.spotify.com/v1`
- **Auth**: OAuth 2.0 Authorization Code with PKCE
- **Scopes Required**: `user-library-read`, `user-read-playback-position`

| Endpoint | Description | Pagination |
|---|---|---|
| `GET /me/episodes` | Saved episodes in user's library | Offset-based, `limit` max 50 |
| `GET /me/shows` | Saved/followed podcast shows | Offset-based, `limit` max 50 |
| `GET /episodes/{id}` | Episode details (name, description, duration, show, release date) | N/A |
| `GET /shows/{id}` | Show details (name, publisher, description, total episodes) | N/A |
| `GET /shows/{id}/episodes` | List episodes for a show | Offset-based, `limit` max 50 |

- **Rate Limits**: Same as Spotify music — rolling 30-second window, ~180 requests/30s per app
- **Recently Played**: The `GET /me/player/recently-played` endpoint currently does **not** support podcast episodes. Saved episodes is the best proxy.

#### Available Data Fields

| Spotify Field | MediaItem Field | Notes |
|---|---|---|
| `episode.name` | `title` | Episode title |
| `episode.show.name` | `series` | Podcast show name |
| `episode.show.publisher` | `creator` | Publisher/creator |
| `episode.duration_ms` | `duration` | In milliseconds |
| `episode.release_date` | `releaseDate` | ISO date string |
| `episode.description` | `description` | Episode description |
| `episode.resume_point.fully_played` | `completed` | Boolean — requires `user-read-playback-position` |
| `episode.resume_point.resume_position_ms` | `progress` | Playback position in ms |

#### Gotchas & Limitations
- No "recently played" support for podcasts — only saved episodes
- No timestamp of *when* the user listened, only resume position
- GDPR export includes full podcast streaming history with timestamps (request via privacy.spotify.com, takes ~30 days)
- Dev mode limited to 25 users

#### Plugin Classification
- **Auth Type**: OAuth (shared with Spotify music plugin)
- **Sync Strategy**: Incremental (track saved episodes + resume points)
- **Difficulty**: Easy (API already documented for music plugin)
- **MVP Priority**: Yes — already integrated with Spotify auth

---

### 2. Apple Podcasts

#### Access Method
- **Primary**: Apple Privacy Data Request (via privacy.apple.com)
- **Fallback**: None. No API, no in-app export.

#### Data Export
- **How**: Go to privacy.apple.com → "Request a copy of your data" → Select "Apple Media Services information" (includes Podcasts)
- **Format**: CSV files
- **Delivery time**: Up to 7 days
- **What's included**: Subscription list, play history with timestamps, episode details
- **Frequency limit**: Apple limits data requests (typically a few per month)

#### Available Data Fields

| Apple Field | MediaItem Field | Notes |
|---|---|---|
| Podcast name | `series` | Show title |
| Episode name | `title` | Episode title |
| Play date/time | `consumedAt` | When listened |
| Play duration | `duration` | Time spent listening |
| Completion status | `completed` | Inferred from duration vs total |

#### Gotchas & Limitations
- **No API whatsoever** — Apple Podcasts has no developer API for listener data
- Privacy data request is manual and slow (days)
- CSV format/schema is undocumented and may change
- No programmatic/automated sync possible
- macOS app stores data in CoreData SQLite DB (`~/Library/Group Containers/...`) — fragile, undocumented, version-dependent

#### Plugin Classification
- **Auth Type**: FileImport (manual CSV upload from Apple privacy request)
- **Sync Strategy**: Full re-import
- **Difficulty**: Hard (manual process, undocumented format)
- **MVP Priority**: No — manual-only, no automation path

---

### 3. Pocket Casts

#### Access Method
- **Primary**: OPML export (subscriptions only)
- **Fallback**: Unofficial sync API (undocumented, may break)

#### Data Export
- **OPML Export**: Settings → Export → OPML. Exports podcast subscriptions only (show title + RSS feed URL). **Does not include listening history.**
- **GDPR/Privacy Request**: Can request full data including listening history via privacy request to Pocket Casts support
- **Unofficial API**: Pocket Casts uses a sync API at `https://api.pocketcasts.com` — endpoints exist for history, starred episodes, and listening stats. Since the apps are open source (GitHub: Automattic/pocket-casts-android), the API surface can be reverse-engineered.

#### Known Unofficial API Endpoints

| Endpoint | Description | Notes |
|---|---|---|
| `POST /user/login` | Auth with email/password | Returns token |
| `POST /user/history` | Listening history | Paginated, returns episode UUIDs + timestamps |
| `POST /user/starred` | Starred episodes | Episode list |
| `POST /user/stats/summary` | Listening stats | Total time listened, etc. |

#### Available Data Fields

| Pocket Casts Field | MediaItem Field | Notes |
|---|---|---|
| Episode title | `title` | From history/episode data |
| Podcast title | `series` | Show name |
| Played up to (seconds) | `progress` | Playback position |
| Duration | `duration` | Episode length |
| Published date | `releaseDate` | Episode publish date |
| Playing status | `completed` | Completed/in-progress |
| Last played timestamp | `consumedAt` | When listened |

#### Gotchas & Limitations
- Official export is subscriptions only — no history
- Unofficial API is undocumented and may change without notice
- Apps are open source (Automattic/pocket-casts-android, Automattic/pocket-casts-ios) which helps with API discovery
- No OAuth — uses email/password auth (token-based)
- Rate limits unknown

#### Plugin Classification
- **Auth Type**: APIKey (email/password → token)
- **Sync Strategy**: Incremental (history endpoint is paginated)
- **Difficulty**: Medium (unofficial API, but open-source apps help)
- **MVP Priority**: No — unofficial API risk, smaller user base

---

### 4. Overcast

#### Access Method
- **Primary**: Extended OPML export (via account page)
- **Fallback**: None. No API.

#### Data Export
- **How**: Log in at overcast.fm → Account → "Export Your Data" → choose "All data" for extended OPML
- **Format**: OPML (XML) with extended attributes for playlists and episode data
- **Two export options**:
  - **OPML**: Subscriptions only (show name + RSS feed URL)
  - **All data**: Extended OPML with playlists, episode progress, played status
- **Rate limit**: ~10 requests/day for the "All data" export (per Marco Arment)
- **Automation**: Can be scripted — login via web session, download OPML file (see IndieWeb examples in Python/Node.js)

#### Available Data Fields

| Overcast Field | MediaItem Field | Notes |
|---|---|---|
| Episode title (OPML `text`) | `title` | Episode name |
| Podcast title (parent `text`) | `series` | Show name |
| RSS feed URL (`xmlUrl`) | `externalUrl` | Feed URL for metadata enrichment |
| Played status | `completed` | Extended OPML attribute |
| Progress | `progress` | Playback position (extended OPML) |
| Playlist membership | `tags` | Which playlists the episode is in |

#### Gotchas & Limitations
- **No API** — export-only
- Extended OPML format is not formally documented
- Rate limited to ~10 exports/day
- iOS-only app (plus web player) — smaller cross-platform audience
- Web scraping for automation is fragile (session cookies)
- No listen timestamps — only current played/unplayed status

#### Plugin Classification
- **Auth Type**: FileImport (OPML upload) or web session scraping
- **Sync Strategy**: Full re-import (snapshot of current state)
- **Difficulty**: Medium (OPML parsing is straightforward, automation is fragile)
- **MVP Priority**: No — niche audience, no timestamps

---

### 5. Google Podcasts (Discontinued)

#### Access Method
- **Status**: **Shut down April 2, 2024**. No longer accessible.
- Migration tool was available until July 2024 to export OPML or migrate to YouTube Music.

#### What Was Available
- **OPML export**: Subscription list only (via Google Takeout or in-app migration tool)
- **Migration to YouTube Music**: Transferred subscriptions (not history) to YouTube Music
- **Google Takeout**: Could export subscriptions as OPML before shutdown

#### Gotchas & Limitations
- **Service is dead** — no data access possible for new users
- Users who migrated to YouTube Music only got subscriptions, not listening history
- Users who exported OPML can import subscriptions into other apps
- No historical listening data was ever exportable

#### Plugin Classification
- **Auth Type**: None (discontinued)
- **Sync Strategy**: N/A
- **Difficulty**: N/A
- **MVP Priority**: No — service no longer exists

---

## Summary Comparison

| Platform | API | Export | History | Timestamps | Automation | MVP? |
|---|---|---|---|---|---|---|
| **Spotify** | ✅ Full API | ✅ GDPR | ✅ Saved + resume | ⚠️ No listen time | ✅ OAuth | **Yes** |
| **Apple Podcasts** | ❌ None | ⚠️ Privacy request | ✅ Via request | ✅ In export | ❌ Manual only | No |
| **Pocket Casts** | ⚠️ Unofficial | ⚠️ Subs only | ✅ Via unofficial API | ✅ In API | ⚠️ Fragile | No |
| **Overcast** | ❌ None | ✅ Extended OPML | ⚠️ Status only | ❌ No timestamps | ⚠️ Scriptable | No |
| **Google Podcasts** | ❌ Dead | ❌ Dead | ❌ Never | ❌ Never | ❌ Dead | No |

## Recommendation

For MVP, **Spotify podcast data** should be added as an extension of the existing Spotify music plugin — it uses the same OAuth flow and adds minimal complexity. Post-MVP, **Pocket Casts** is the most promising due to its open-source apps enabling API discovery. Apple Podcasts and Overcast should be file-import only plugins due to lack of APIs.
