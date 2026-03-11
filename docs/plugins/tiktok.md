# TikTok Plugin

## Overview

TikTok provides consumption data primarily through its GDPR/DSAR data export (Settings → Account → Download your data). The JSON export includes a "Video Browsing History" section with minimal fields: a timestamp and a video URL. No official API exists for accessing a user's personal watch history. TikTok's Data Portability API is a potential programmatic alternative but requires app approval and is designed for third-party services receiving user-initiated data transfers.

## Access Method

- **Primary**: GDPR/DSAR File Export (JSON)
- **Fallback**: Data Portability API (OAuth, requires TikTok app approval)

## API Details

### Data Portability API

TikTok's Data Portability API (`developers.tiktok.com/doc/data-portability-data-types`) allows users to designate a third-party app to receive one-time or ongoing transfers of their data archive.

- **Base URL**: `https://open.tiktokapis.com/v2/`
- **Auth**: OAuth 2.0 (user-initiated transfer flow)
- **Key Endpoints**: Not a traditional REST API — user triggers a data transfer to your app from TikTok's settings. Your app receives a webhook/callback with a download URL for the data archive.
- **Rate Limits**: Not publicly documented; transfers are user-initiated.
- **Scopes Required**: Data Portability API access (requires app review and approval)
- **Approval**: Must apply through TikTok Developer Portal. Guidelines require demonstrating a legitimate data portability use case.

### Research API (not applicable)

The Research API (`open.tiktokapis.com/v2/research/`) provides public video/user queries for vetted academic researchers. It does **not** provide access to a user's personal watch history. Irrelevant for consumption tracking.

### Other APIs (not applicable)

- **Login Kit**: OAuth login only, no consumption data.
- **Content Posting API**: For publishing, not reading history.
- **Share Kit**: Deep linking, no data access.

## Data Export (GDPR/DSAR)

### How to export
1. Open TikTok app or website → Profile → Settings
2. Go to **Account** → **Download your data**
3. Set **Data format** to **JSON** (not TXT)
4. Select **All data** (custom data selections may omit browsing history)
5. Click **Request data** — takes minutes to a few days
6. Download the `.zip` file when ready

### Format
ZIP archive containing a single `user_data.json` file with sections for different data types.

### Watch History JSON Structure

Located under `Activity` → `Video Browsing History` → `VideoList`:

```json
{
  "Activity": {
    "Video Browsing History": {
      "VideoList": [
        {
          "Date": "2024-04-20 23:17:46",
          "VideoLink": "https://www.tiktokv.com/share/video/7359012345678901234/"
        },
        {
          "Date": "2024-04-20 16:35:40",
          "VideoLink": "https://www.tiktokv.com/share/video/7358098765432109876/"
        }
      ]
    }
  }
}
```

### Other Relevant Sections in Export

The full data export also includes:

| Section | Key | Fields | Useful? |
|---|---|---|---|
| Like List | `Activity.Like List.ItemFavoriteList` | `Date`, `VideoLink` | Yes — liked videos |
| Favorite Videos | `Activity.Favorite Videos.FavoriteVideoList` | `Date`, `VideoLink` | Yes — bookmarked |
| Share History | `Activity.Share History.ShareHistoryList` | `Date`, `SharedContent`, `Link`, `Method` | Marginal |
| Search History | `Activity.Search History.SearchList` | `Date`, `SearchTerm` | Marginal |
| Comment History | `Comment.Comments.CommentsList` | `Date`, `Comment`, `Photo/Video Link` | Marginal |

### Limitations

- **Only 2 fields per watch entry**: `Date` (view timestamp) and `VideoLink` (URL). No video title, creator, duration, category, or engagement time.
- **Export delay**: Can take minutes to days for TikTok to prepare the archive.
- **No incremental export**: Full archive each time. No cursor or "since last export" option.
- **Windows ZIP issue**: The ZIP format TikTok uses cannot be correctly extracted by Windows' built-in unzipper (community-reported; third-party tools like 7-Zip work).
- **Video link format**: Uses `tiktokv.com` domain (not `tiktok.com`). The numeric ID in the URL path is the video ID.

## Available Data Fields

| Platform Field | MediaItem Field | Notes |
|---|---|---|
| `VideoList[].Date` | `consumed_at` | Format: `YYYY-MM-DD HH:MM:SS` (timezone unclear, likely UTC or account timezone) |
| `VideoList[].VideoLink` | `url` | `https://www.tiktokv.com/share/video/{id}/` |
| Video ID from URL | `external_id` | Extract numeric ID from URL path |
| _(not provided)_ | `title` | Must be enriched by fetching video metadata via URL/oEmbed |
| _(not provided)_ | `creator` | Must be enriched |
| _(not provided)_ | `duration` | Must be enriched |
| _(not provided)_ | `time_spent` | Not available at all — TikTok does not export watch duration |
| `"tiktok"` | `platform` | Hardcoded |
| `"video"` | `type` | Hardcoded — TikTok is video-only |
| Like List entries | `raw_metadata.liked` | Boolean flag if video appears in Like List |
| Favorite entries | `raw_metadata.favorited` | Boolean flag if video appears in Favorites |

## Enrichment Strategy

Since the export only gives timestamps and URLs, enrichment is critical:

1. **oEmbed endpoint**: `https://www.tiktok.com/oembed?url={video_url}` — returns `title`, `author_name`, `author_url`, `thumbnail_url`. No auth required. Rate limits unknown but generous for moderate use.
2. **Scrape/resolve the URL**: The `tiktokv.com` share link redirects to the full TikTok URL, which contains the creator username in the path.
3. **Video ID extraction**: Parse the numeric ID from the URL path (`/share/video/(\d+)/`).

## Gotchas & Limitations

- **Extremely sparse data**: Only timestamp + URL per video watched. No metadata in the export itself.
- **No watch duration**: TikTok does not export how long you watched a video. This is a fundamental gap for consumption analysis.
- **No discovery context**: Cannot tell if a video came from FYP, search, profile visit, or a share.
- **Deleted videos**: If a video is deleted from TikTok, the URL will 404 and enrichment will fail. Store enrichment results aggressively.
- **oEmbed rate limiting**: Unknown limits; implement backoff. Consider batch enrichment with delays.
- **Export frequency**: Users must manually re-export. No automated scheduled exports via GDPR flow.
- **Data Portability API approval**: Requires formal application to TikTok. Approval criteria are strict; must demonstrate legitimate data portability purpose. Timeline uncertain.
- **Regional availability**: TikTok data export availability may vary by region due to ongoing regulatory and ban-related changes (especially US).

## Plugin Classification

- **Auth Type**: FileImport (GDPR export) / OAuth (Data Portability API, if approved)
- **Sync Strategy**: Full re-import (parse entire JSON export each time; deduplicate by `external_id`)
- **Difficulty**: Medium — simple parsing but heavy enrichment needed for useful data
- **MVP Priority**: Yes — TikTok is a major consumption platform, especially for younger demographics. File import is zero-auth and works immediately. Enrichment can be deferred to a background job.
