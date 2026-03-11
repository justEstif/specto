# Netflix Plugin

## Overview
Netflix has **no public API** for accessing user viewing history. Data is available through two export methods: a simple CSV download from the viewing activity page, and a comprehensive GDPR/privacy data request. The GDPR export is the richer source, providing duration, device info, and country data alongside titles and timestamps.

## Access Method
- **Primary**: File Export (GDPR data request via `netflix.com/account/getmyinfo`)
- **Fallback**: Simple CSV export from `netflix.com/viewingactivity` (fewer fields)

## Data Export

### Method 1: Simple Viewing Activity CSV
1. Log in to Netflix, go to **Account → Profile → Viewing activity** (`netflix.com/viewingactivity`)
2. Click **Download all** at the bottom of the page
3. Downloads a CSV file (`NetflixViewingHistory.csv`)

**Format**: CSV (comma-separated, UTF-8)
**Fields**: Only 2 columns:
- `Title` — show/episode name (e.g., `Black Mirror: Season 1: The National Anthem`)
- `Date` — date watched (e.g., `10/01/2019`, format `MM/DD/YYYY`)

**Limitations**: No duration, no device info, no timestamps (date only), per-profile only.

### Method 2: GDPR / Privacy Data Request (Recommended)
1. Go to `https://www.netflix.com/account/getmyinfo`
2. Submit a data request (requires account owner)
3. Wait up to **30 days** (often arrives in 24–72 hours)
4. Download ZIP file; navigate to `CONTENT_INTERACTION/ViewingActivity.csv`

**Format**: CSV (comma-separated, UTF-8)
**Fields** (10 columns):

| Column | Description |
|---|---|
| `Profile Name` | Which profile watched |
| `Start Time` | Timestamp in **UTC** (e.g., `2023-06-01 18:30:00`) |
| `Duration` | Watch duration (e.g., `01:23:45` = HH:MM:SS) |
| `Attributes` | Metadata flags (often empty) |
| `Title` | Full title string (show: season: episode combined) |
| `Supplemental Video Type` | Blank for regular content; populated for trailers/previews |
| `Device Type` | Device description (e.g., `Samsung Smart TV`, `Chrome on Mac`) |
| `Bookmark` | Playback position when stopped |
| `Latest Bookmark` | Most recent bookmark position |
| `Country` | Country code where viewed (e.g., `US`, `GB`) |

The ZIP also contains other useful files: `MyList.csv`, `Ratings.csv`, `SearchHistory.csv`, `ClickstreamData/`, etc.

## Available Data Fields

Mapping from GDPR `ViewingActivity.csv` to our MediaItem schema:

| Platform Field | MediaItem Field | Notes |
|---|---|---|
| `Title` | `title` | Requires parsing — format is `Show: Season X: Episode Name` for TV; plain title for films |
| `Start Time` | `consumed_at` | UTC timestamp; convert to user's local timezone |
| `Duration` | `duration` | HH:MM:SS format; parse to seconds/minutes |
| `Profile Name` | `user_profile` | Maps to account sub-profile |
| `Device Type` | `device` | Free-text, inconsistent naming across devices |
| `Country` | `country` | 2-letter country code |
| `Supplemental Video Type` | — | Use to **filter out** trailers/previews (non-empty = skip) |
| `Bookmark` / `Latest Bookmark` | `progress` | Can determine if content was finished |
| — | `media_type` | Not provided; infer from Title structure (`:` separators = TV show) |
| — | `platform_id` | Not provided; Netflix doesn't include content IDs in export |

## Parsing Notes

- **Title splitting**: TV shows follow `Show Name: Season X: Episode Title` format. Split on `:` to extract series name, season, and episode. Films have no `:` separator (usually).
- **Supplemental Video Type**: Filter rows where this column is non-empty to exclude trailers, recaps, and promotional content.
- **Duration**: Short durations (< 2 min) typically indicate accidental clicks or previews; consider filtering.
- **Simple CSV vs GDPR**: The simple export only has Title + Date. Always prefer the GDPR export for analysis.

## Gotchas & Limitations

- **No public API** — Netflix shut down their public API in 2014. No OAuth, no programmatic access.
- **No content IDs** — exports contain titles only, no Netflix internal IDs. Title matching is required for metadata enrichment (via TMDB/OMDB).
- **Title format is locale-dependent** — non-English accounts may have localized titles.
- **UTC timestamps only** — must convert to local timezone for time-of-day analysis.
- **GDPR request delay** — can take up to 30 days; cannot be automated or repeated frequently.
- **Device names are messy** — inconsistent naming (e.g., `Samsung 2015 Smart TV` vs `Samsung SmartTV`).
- **Per-account export** — includes all profiles; filter by `Profile Name` for per-user analysis.
- **No ratings in ViewingActivity** — ratings are in a separate `Ratings.csv` file in the GDPR export.
- **Column order may vary** — some exports have columns in different positions; parse by header name, not index.

## Plugin Classification
- **Auth Type**: FileImport
- **Sync Strategy**: Full re-import (manual file upload each time)
- **Difficulty**: Medium (parsing title strings, handling locale differences, no content IDs)
- **MVP Priority**: Yes — Netflix is a top streaming platform; file import is straightforward despite lack of API
