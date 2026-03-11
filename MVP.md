# Media Consumption Analysis — MVP

## What

A personal dashboard that aggregates your digital media consumption across multiple
platforms — Spotify, YouTube, Netflix, Prime Video, blogs/RSS — normalizes it into a
unified schema, and gives you insight into **what** you're consuming, **how much**, and
**what topics/patterns** dominate your attention.

### Core Features (MVP)

1. **Unified Media Feed** — Pull consumption history from connected platforms into one view
2. **Tagging & Categorization** — Auto-classify content by topic, type (music, video, article, podcast), mood/genre
3. **Consumption Insights** — Visualize patterns: time spent, topic distribution, platform breakdown, trends over time
4. **Shareable Profiles** — Generate a shareable summary of your media diet (think Spotify Wrapped, but for everything)

### What It Is NOT (MVP)

- Not a recommendation engine
- Not a content aggregator/player — you still consume on the original platforms
- Not a social network — sharing is one-directional (you publish a profile, others view it)

---

## Why

**The problem:** Your digital life is fragmented across dozens of platforms. Each one
knows a slice of you, but you don't have a unified picture of your own consumption.
You can't answer simple questions like:

- "What topics have I been gravitating toward this month?"
- "How much time am I spending on passive entertainment vs. learning?"
- "Am I in an echo chamber or diversifying my inputs?"

**The pain:**

- No cross-platform self-awareness
- Mindless consumption — no feedback loop to make it intentional
- Platform silos — each app optimizes for engagement, not your well-being
- You already felt this with Spotify alone (hence the era organizer) — it's 10x worse across all media

**Who is this for (MVP):**

- You. Power users who consume a lot of digital media and want self-awareness.
- Quantified-self enthusiasts.
- People who want to share their media taste/identity.

---

## How

### Data Acquisition (the hard part)

Each platform has different access levels. MVP should focus on what's **actually accessible**:

| Platform        | Data Source                     | Access Method                                                  | Difficulty |
| --------------- | ------------------------------- | -------------------------------------------------------------- | ---------- |
| **Spotify**     | Listening history, liked songs  | OAuth API (you've done this)                                   | ✅ Easy    |
| **YouTube**     | Watch history, liked videos     | Google Takeout export OR YouTube Data API v3 (OAuth)           | 🟡 Medium  |
| **Netflix**     | Viewing history                 | Manual CSV export (account settings) OR Takeout-style download | 🟡 Medium  |
| **TikTok**      | Watch history, likes, favorites | GDPR data download (JSON from settings) OR unofficial API      | 🟡 Medium  |
| **Prime Video** | Watch history                   | Manual export / scraping (no public API)                       | 🔴 Hard    |
| **Blogs/RSS**   | Read articles                   | User-managed RSS feeds + read-tracking (browser extension?)    | 🟡 Medium  |
| **Podcasts**    | Listen history                  | Depends on app — Spotify API covers some, Apple has none       | 🔴 Hard    |

**MVP recommendation:** Start with **Spotify + YouTube** — both have OAuth APIs. Add Netflix
via CSV import. Defer Prime Video and others to post-MVP.

### Normalized Schema (draft)

```
MediaItem {
  id:           uuid
  platform:     "spotify" | "youtube" | "netflix" | ...
  type:         "music" | "video" | "article" | "podcast"
  title:        string
  creator:      string          // artist, channel, author
  consumed_at:  timestamp
  duration:     duration?       // how long the content is
  time_spent:   duration?       // how long you actually engaged
  tags:         string[]        // auto-generated: genre, topic, mood
  url:          string?         // link back to original
  raw_metadata: jsonb           // platform-specific fields
}
```

### Enrichment & Classification

- Use **Last.fm tags** for music (you already do this)
- Use **YouTube video categories + channel topics** from the API
- Use **LLM-based classification** for titles/descriptions → topics (lightweight, can run locally or via API)
- Netflix genres from the export data

### Tech Stack (suggested, based on your experience)

- **Backend:** Go
- **Database:** PostgreSQL (jsonb for raw_metadata flexibility)
- **Frontend:** Web app — Templ + HTMX (type-safe components, compile-time checks)
- **Auth:** OAuth per platform + app-level user accounts
- **Sharing:** Public profile pages with a unique URL

### Plugin Architecture

Each media source is a **plugin** that implements a common interface. The core system
knows nothing about Spotify, YouTube, etc. — it only knows how to receive `MediaItem`s
from plugins and run them through enrichment + display.

```go
// Every plugin implements this interface
type SourcePlugin interface {
    Name() string                          // "spotify", "youtube", etc.
    AuthType() AuthType                    // OAuth, FileImport, APIKey, None
    Sync(ctx context.Context) ([]MediaItem, error)  // fetch & normalize
    Enrich(ctx context.Context, items []MediaItem) ([]MediaItem, error) // optional platform-specific enrichment
}
```

**Why plugins:**
- Add new platforms without touching core code
- Each plugin owns its own auth flow, API quirks, and normalization logic
- Plugins can be developed and released independently
- Community can contribute plugins for platforms we don't use
- Easy to disable/enable per user

**Plugin types by ingestion method:**
| Type | How It Works | Examples |
|------|-------------|----------|
| **OAuth API** | User connects account, plugin pulls data via API | Spotify, YouTube |
| **File Import** | User uploads an export file, plugin parses it | Netflix CSV, TikTok JSON, Prime Video |
| **Browser Extension** | Extension tracks consumption, sends to core | Blogs/RSS, any web-based content |
| **Manual** | User logs entries directly | Books, podcasts without API |

### Architecture (MVP)

```
                    ┌─────────────────────────┐
                    │     Plugin Registry      │
                    │  register / discover /   │
                    │  enable per user         │
                    └────────────┬────────────┘
                                 │
          ┌──────────────────────┼──────────────────────┐
          │                      │                      │
          ▼                      ▼                      ▼
   ┌─────────────┐       ┌─────────────┐       ┌─────────────┐
   │   Plugin:   │       │   Plugin:   │       │   Plugin:   │
   │   Spotify   │       │   YouTube   │       │   Netflix   │  ... more plugins
   │  (OAuth)    │       │  (OAuth)    │       │ (file import)│
   └──────┬──────┘       └──────┬──────┘       └──────┬──────┘
          │                      │                      │
          │         ┌────────────┘                      │
          │         │    ┌─────────────────────────────┘
          ▼         ▼    ▼
   ┌─────────────────────────────────────────────────┐
   │         Core: Normalized MediaItem Store         │
   │              (PostgreSQL + jsonb)                 │
   └─────────────────────┬───────────────────────────┘
                         │
                         ▼
   ┌─────────────────────────────────────────────────┐
   │           Enrichment Pipeline (pluggable)        │
   │   (Last.fm, LLM tagging, category mapping)      │
   └─────────────────────┬───────────────────────────┘
                         │
                         ▼
   ┌─────────────────────────────────────────────────┐
   │           Dashboard / Insights UI                │
   │  - Timeline view                                 │
   │  - Topic/genre breakdown                         │
   │  - Platform distribution                         │
   │  - Shareable profile page                        │
   └─────────────────────────────────────────────────┘
```

---

## Competition

| Product                          | What It Does             | How We Differ                                          |
| -------------------------------- | ------------------------ | ------------------------------------------------------ |
| **Spotify Wrapped**              | Annual listening summary | Single platform, once a year, not actionable           |
| **Last.fm**                      | Music scrobbling & stats | Music only, no video/articles/podcasts                 |
| **Trakt.tv**                     | TV & movie tracking      | Manual tracking, no music/articles, no unified view    |
| **Goodreads**                    | Book tracking            | Books only                                             |
| **Letterboxd**                   | Movie tracking & social  | Movies only, manual logging                            |
| **RescueTime / Screen Time**     | App usage time tracking  | Tracks _time in apps_, not _what content_ you consumed |
| **Your Spotify Era Organizer**   | Spotify era clustering   | Music only, single platform                            |
| **Obsidian / Notion dashboards** | Manual media logs        | Requires manual entry, no automation                   |

**Gap in the market:** Nobody does **automated cross-platform content-level tracking with
topic analysis**. Existing tools are either single-platform, manual-entry, or track app
usage time rather than content.

---

## MVP Milestones

1. **M1 — Core + Plugin System** — Normalized schema, plugin interface, plugin registry, basic DB layer
2. **M2 — First Plugin: Spotify** — Reuse era organizer work, implement as plugin, prove the interface works
3. **M3 — Dashboard v1** — Basic insights UI showing data from any connected plugin
4. **M4 — Second Plugin: YouTube** — OAuth API plugin, validates that the plugin interface generalizes
5. **M5 — Enrichment Pipeline** — LLM-based topic tagging that works across all content regardless of source
6. **M6 — Third Plugin: Netflix or TikTok** — File import plugin, proves the non-OAuth ingestion path
7. **M7 — Shareable Profiles** — Public URLs, "my media diet" pages

**Principle:** Build the core right, then grow one plugin at a time. Each new plugin should
be addable without changing core code — if it isn't, fix the abstraction before moving on.

---

## Open Questions

- [x] What's the primary interface — **Web app**
- [x] Self-hosted only or hosted service? — **Hosted multi-tenant.** You deploy, others sign up and use it.
- [ ] How to handle platforms with no API (Prime Video)? Browser extension? Manual export?
- [x] Privacy model — is all data local, or stored server-side? — **Server-side**, scoped per user, OAuth tokens encrypted at rest.
- [ ] Sharing granularity — share everything, or curate what's public?
