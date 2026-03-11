# Goodreads Plugin

## Overview
Goodreads shut down its public API in December 2020 and has not replaced it. The only reliable way to get reading history data is via Goodreads' built-in CSV export. The export contains all shelved books with ratings, dates, shelves, and ISBN identifiers. Open Library's free API can enrich this data with genres/subjects, cover images, and additional metadata using ISBN lookups.

## Access Method
- **Primary**: CSV file export (manual, user-initiated)
- **Fallback**: None. No API, no OAuth, no GDPR export alternative.

## Data Export
- **How to export**: Goodreads → My Books → (left sidebar or bottom) "Import and export" → "Export Library" → downloads `goodreads_library_export.csv`
- **Format**: CSV (UTF-8)
- **Frequency**: Can be re-exported at any time; no rate limit on exports
- **Limitations**:
  - Manual process only — no automation possible
  - No cover image URLs in export
  - No genre/subject data in export
  - Review text may contain commas/newlines (proper CSV quoting is used)
  - ISBN fields are quoted with `=""` prefix to prevent Excel from mangling them (e.g., `="0451526538"`) — must be stripped during parsing

### CSV Fields
All 31 columns in the export:

| Column | Example | Notes |
|---|---|---|
| `Book Id` | `13278990` | Goodreads internal ID |
| `Title` | `The Housing Monster` | |
| `Author` | `prole.info` | Primary author |
| `Author l-f` | `prole.info, prole.info` | Last, First format |
| `Additional Authors` | | Comma-separated |
| `ISBN` | `="160486530X"` | Needs `=""` stripping |
| `ISBN13` | `="9781604865301"` | Needs `=""` stripping |
| `My Rating` | `0` | 0–5, 0 = unrated |
| `Average Rating` | `3.77` | Community average |
| `Publisher` | `PM Press` | |
| `Binding` | `Paperback` | Format type |
| `Number of Pages` | `160` | |
| `Year Published` | `2012` | This edition |
| `Original Publication Year` | `2011` | First publication |
| `Date Read` | `2023/01/15` | YYYY/MM/DD, blank if unread |
| `Date Added` | `2017/12/07` | When added to shelf |
| `Bookshelves` | `currently-reading` | Comma-separated custom shelves |
| `Bookshelves with positions` | `currently-reading (#3)` | With sort position |
| `Exclusive Shelf` | `currently-reading` | One of: `read`, `currently-reading`, `to-read` |
| `My Review` | | HTML-formatted review text |
| `Spoiler` | | Spoiler flag |
| `Private Notes` | | User's private notes |
| `Read Count` | `1` | Times read |
| `Recommended For` | | |
| `Recommended By` | | |
| `Owned Copies` | `0` | |
| `Original Purchase Date` | | |
| `Original Purchase Location` | | |
| `Condition` | | |
| `Condition Description` | | |
| `BCID` | | BookCrossing ID, rarely populated |

## Metadata Enrichment via Open Library

Since the CSV lacks genres, cover images, and subjects, we use Open Library's free API (no auth required) to enrich records by ISBN.

### Key Endpoints

- **Books API**: `GET https://openlibrary.org/api/books?bibkeys=ISBN:{isbn}&format=json&jscmd=data`
  - Returns: title, authors, publishers, subjects, cover URLs, identifiers, number of pages
  - Cover URLs: `https://covers.openlibrary.org/b/isbn/{isbn}-L.jpg` (S/M/L sizes)
  - Includes `subjects` array with genre/topic data

- **Search API**: `GET https://openlibrary.org/search.json?isbn={isbn}&fields=key,title,author_name,subject,first_publish_year,cover_i`
  - Better for subject/genre data, returns work-level info
  - `cover_i` → `https://covers.openlibrary.org/b/id/{cover_i}-L.jpg`

- **Rate Limits**: No official key required. Undocumented but ~100 req/s is safe. Use polite delays (~100ms between requests).

## Available Data Fields

| Platform Field | MediaItem Field | Source | Notes |
|---|---|---|---|
| `Title` | `title` | CSV | |
| `Author` + `Additional Authors` | `creators` | CSV | |
| `ISBN` / `ISBN13` | `externalIds.isbn` | CSV | Strip `=""` wrapper |
| `Book Id` | `externalIds.goodreads` | CSV | |
| `My Rating` | `userRating` | CSV | 0 = null, 1–5 scale |
| `Date Read` | `completedAt` | CSV | |
| `Date Added` | `addedAt` | CSV | |
| `Exclusive Shelf` | `status` | CSV | Map: `read`→completed, `currently-reading`→in-progress, `to-read`→planned |
| `Number of Pages` | `metadata.pageCount` | CSV | |
| `Publisher` | `metadata.publisher` | CSV | |
| `Binding` | `metadata.format` | CSV | |
| `Original Publication Year` | `releaseYear` | CSV | |
| `My Review` | `userReview` | CSV | HTML, needs sanitizing |
| `Read Count` | `metadata.readCount` | CSV | |
| `Private Notes` | `userNotes` | CSV | |
| `subjects` | `genres` | Open Library | From Books or Search API |
| cover URL | `coverImageUrl` | Open Library | Via ISBN lookup |

## Gotchas & Limitations
- **Manual export only** — user must download CSV from Goodreads and upload/import to our app
- **ISBN not always present** — some editions (especially self-published or obscure) lack ISBNs, making Open Library enrichment impossible for those entries
- **`=""` ISBN format** — must strip the Excel-protection wrapper before using as lookup key
- **Date Read often blank** — many users shelve books as "read" without recording the date
- **My Rating = 0 means unrated**, not zero stars
- **Review text is HTML** — contains `<br>` tags, needs sanitization
- **Open Library coverage gaps** — not all ISBNs exist in Open Library; fallback to title+author search may be needed
- **No incremental sync** — every import is a full re-import; must diff against existing data
- **Bookshelves are user-created** — custom shelf names vary wildly, only `Exclusive Shelf` is standardized

## Plugin Classification
- **Auth Type**: FileImport (user uploads CSV)
- **Sync Strategy**: Full re-import (diff against existing records by Goodreads Book Id)
- **Difficulty**: Easy — CSV parsing + optional HTTP enrichment calls
- **MVP Priority**: Yes — books are a core media type, CSV import is trivial to implement, and Goodreads is the dominant book-tracking platform
