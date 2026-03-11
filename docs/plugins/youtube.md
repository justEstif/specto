# YouTube Plugin

## Overview

YouTube watch history is **not available via the YouTube Data API v3** — Google deprecated the watch history playlist (`HL`) in August 2016. The primary method for obtaining watch history is **Google Takeout** file export. The API remains useful for liked videos, subscriptions, and enriching video metadata (title, channel, duration, tags, category).

## Access Method
- **Primary**: Google Takeout file import (watch history)
- **Fallback**: YouTube Data API v3 via OAuth 2.0 (liked videos, subscriptions, video metadata enrichment)

## API Details

- **Base URL**: `https://www.googleapis.com/youtube/v3`
- **Auth**: OAuth 2.0 (for user-specific data) or API Key (for public video metadata)
- **Daily Quota**: 10,000 units per project (free tier), resets at midnight Pacific Time

### Key Endpoints

| Endpoint | Description | Quota Cost | Pagination |
|---|---|---|---|
| `GET /videos?part=snippet,contentDetails,statistics&id={ids}` | Video metadata (title, channel, duration, tags, category, publish date). Up to 50 IDs per request. | 1 unit | N/A (batch by ID) |
| `GET /playlistItems?part=snippet&playlistId={id}` | Items in a playlist. Use `LL` prefix for Liked Videos playlist. | 1 unit | `pageToken`, 50 items/page |
| `GET /subscriptions?part=snippet&mine=true` | Authenticated user's subscriptions. | 1 unit | `pageToken`, 50 items/page |
| `GET /channels?part=snippet,statistics&id={id}` | Channel metadata. | 1 unit | N/A |
| `GET /search?part=snippet&q={query}` | Search (expensive, avoid for bulk). | 100 units | `pageToken`, 50 items/page |

### OAuth Scopes Required

| Scope | Purpose |
|---|---|
| `https://www.googleapis.com/auth/youtube.readonly` | Read liked videos, subscriptions, playlists |
| `https://www.googleapis.com/auth/youtube` | Full access (not needed for read-only tracking) |

**Note**: No scope grants access to watch history — it is simply not exposed by the API.

### Quota Budget Example

With 10,000 units/day:
- Enrich 500 videos via `videos.list` (50 per request) = **10 units**
- Fetch all liked videos (1,000 videos) = **20 units**
- Fetch subscriptions (200 channels) = **4 units**
- Remaining: **9,966 units** — quota is generous for read-heavy consumption tracking

## Data Export (Google Takeout)

### How to Export
1. Go to [takeout.google.com](https://takeout.google.com)
2. Deselect all, then select **YouTube and YouTube Music**
3. Click "All YouTube data included" → select only **history** (watch history + search history)
4. Choose format: **JSON** (recommended) or HTML
5. Export → download `.zip` → extract `Takeout/YouTube and YouTube Music/history/watch-history.json`

### Watch History JSON Format

```json
[
  {
    "header": "YouTube",
    "title": "Watched Some Video Title",
    "titleUrl": "https://www.youtube.com/watch?v=VIDEO_ID",
    "subtitles": [
      {
        "name": "Channel Name",
        "url": "https://www.youtube.com/channel/CHANNEL_ID"
      }
    ],
    "time": "2025-01-15T14:32:00.000Z",
    "products": ["YouTube"],
    "activityControls": ["YouTube watch history"]
  }
]
```

### Limitations
- **No watch duration** — only records that a video was watched, not how long
- **History depth varies** — typically 6–12 months of recent history; older entries may be missing even if you've had the account for years
- **Manual process** — no API for triggering Takeout; user must export and upload the file
- **Export delay** — large archives can take hours/days to generate
- **Deleted videos** — appear as "Watched a video that has been removed" with no video ID

## Available Data Fields

### From Google Takeout (Watch History)

| Takeout Field | MediaItem Field | Notes |
|---|---|---|
| `titleUrl` (parse `v=` param) | `externalId` | YouTube video ID |
| `title` (strip "Watched " prefix) | `title` | May say "Watched a video that has been removed" |
| `time` | `consumedAt` | ISO 8601 timestamp |
| `subtitles[0].name` | `creator` | Channel name |
| `subtitles[0].url` (parse channel ID) | `creatorId` | Channel ID |
| — | `source` | `"youtube"` |
| — | `mediaType` | `"video"` |

### From API Enrichment (videos.list)

| API Field | MediaItem Field | Notes |
|---|---|---|
| `snippet.title` | `title` | Canonical title (better than Takeout) |
| `snippet.channelTitle` | `creator` | |
| `snippet.publishedAt` | `releasedAt` | |
| `snippet.tags` | `tags` | Array of strings |
| `snippet.categoryId` | `genre` | Requires category ID → name mapping |
| `snippet.thumbnails.medium.url` | `thumbnailUrl` | |
| `contentDetails.duration` | `duration` | ISO 8601 duration (e.g., `PT12M34S`) |
| `statistics.viewCount` | `metadata.viewCount` | |
| `snippet.description` | `metadata.description` | Often long, may want to truncate |

### From API (Liked Videos / Subscriptions)

| Source | MediaItem Field | Notes |
|---|---|---|
| `playlistItems.list` (playlist=LL) | Liked videos with `likedAt` from snippet.publishedAt | Playlist ID `LL` = Liked Videos |
| `subscriptions.list` | Subscription list (channel IDs + names) | Not MediaItems, but useful for filtering/recommendations |

## Gotchas & Limitations

- **Watch history is NOT available via API** — deprecated since Aug 2016. The `HL` (History List) playlist ID no longer works. Google Takeout is the only option.
- **Watch Later playlist** (`WL`) is also inaccessible via API since 2016.
- **Liked Videos playlist** (`LL`) still works via API but may be restricted in some accounts.
- **Takeout has no watch duration** — you know *what* was watched but not *how much*.
- **Takeout history depth is inconsistent** — some users report only 6–8 months despite years of history; there may be a cap on total entries.
- **Deleted/private videos** in Takeout have no video ID — cannot be enriched.
- **Video category IDs** are numeric and region-dependent; requires a separate `videoCategories.list` call to map to names.
- **API quota is generous** for our use case but `search.list` at 100 units/call can burn through it fast — avoid searching, use direct video ID lookups.
- **OAuth consent screen** requires Google verification for sensitive scopes if distributing to >100 users.

## Plugin Classification

- **Auth Type**: FileImport (Takeout) + OAuth 2.0 (API enrichment)
- **Sync Strategy**: Full re-import (Takeout file) + Incremental enrichment (API, by video ID)
- **Difficulty**: Medium — Takeout parsing is straightforward; OAuth setup and quota management add complexity
- **MVP Priority**: **Yes** — YouTube is the most common video platform; Takeout provides a solid data foundation even without real-time sync
