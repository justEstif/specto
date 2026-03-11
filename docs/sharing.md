# Sharing

## Overview

Users can create a live public profile page showing their media consumption — a
cross-platform "media identity" page. The core principle:

**Default to private. Users explicitly opt content into their public profile.**

Nobody should feel judged by their share page. Users control exactly what's visible.

---

## Design Principles

1. **Opt-in, not opt-out** — Nothing is public until the user enables it
2. **Aggregates over specifics** — Show "top genres" not "watched Love Island 47 times"
3. **User curates their identity** — The share page is how you *want* to be seen, not a raw data dump
4. **No surprises** — Preview exactly what others will see before publishing
5. **Revocable** — Unpublish instantly, URL stops working

---

## Profile Page

### URL Structure

```
/share/{profile-slug}
```

- `profile-slug` is user-chosen (like a username), stored in `users.profile_slug`
- No auth required to view
- Returns 404 if profile is unpublished or slug doesn't exist

### What Can Be Shared

Users compose their profile from **blocks** — modular sections they can enable/disable
and reorder:

| Block | What It Shows | Privacy Level |
|-------|--------------|---------------|
| **Top Genres** | Genre distribution chart (e.g., "45% rock, 20% electronic") | Safe — aggregated |
| **Top Topics** | Topic breakdown | Safe — aggregated |
| **Mood Profile** | Mood distribution (e.g., "mostly chill and contemplative") | Safe — aggregated |
| **Platform Mix** | Which platforms they use + percentage | Safe — aggregated |
| **Top Creators** | Most consumed artists/channels/authors (top N) | Medium — reveals taste |
| **Recent Favorites** | User-curated list of items they want to highlight | Safe — explicitly chosen |
| **Listening Stats** | Total hours, items consumed, streaks | Safe — aggregated |
| **Reading List** | Books read (from Goodreads) | Medium — reveals taste |
| **Currently Into** | User-written blurb about current media interests | Safe — user-authored |

### What Is Never Shared

- Individual watch/listen history (the full timeline)
- Consumption timestamps (when you watched something)
- Time spent per item
- Platform credentials or account info
- Items the user hasn't explicitly opted into showing
- Raw metadata

---

## Sharing Controls

### Profile Settings

```
Profile:
  [x] Enable public profile
  URL: /share/estifanos
  
Blocks (drag to reorder):
  [x] Top Genres          — last 30 days
  [x] Mood Profile        — last 30 days
  [x] Top Creators        — top 10, last 30 days
  [ ] Platform Mix         — disabled
  [ ] Top Topics           — disabled
  [x] Recent Favorites     — 5 items, manually curated
  [x] Currently Into       — free text
  [ ] Listening Stats      — disabled
  [ ] Reading List         — disabled
```

### Per-Block Controls

Each block has:
- **Enable/disable** toggle
- **Time range** — last 7 days, 30 days, 90 days, all time
- **Count** — how many items to show (for top-N blocks)
- **Platform filter** — include/exclude specific platforms from this block

### Content Exclusions

Users can exclude content from ALL sharing calculations:

- **Exclude by platform** — "Never include Netflix data in my profile"
- **Exclude by tag** — "Hide anything tagged 'romance'"
- **Exclude individual items** — Flag specific items as "private" in the timeline view

Excluded content is filtered out before any aggregation — it won't affect percentages
or top-N lists on the share page.

---

## Data Model

### Profile Configuration

Stored in a `share_profiles` table:

```sql
CREATE TABLE share_profiles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    enabled     BOOLEAN NOT NULL DEFAULT false,
    blocks      JSONB NOT NULL DEFAULT '[]',    -- ordered list of block configs
    excluded_platforms TEXT[] DEFAULT '{}',       -- platforms to exclude globally
    excluded_tags      TEXT[] DEFAULT '{}',       -- tags to exclude globally
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Block config shape:

```json
[
  {
    "type": "top_genres",
    "enabled": true,
    "time_range": "30d",
    "platforms": null
  },
  {
    "type": "top_creators",
    "enabled": true,
    "time_range": "30d",
    "count": 10,
    "platforms": ["spotify", "youtube"]
  },
  {
    "type": "recent_favorites",
    "enabled": true,
    "item_ids": ["uuid-1", "uuid-2", "uuid-3"]
  },
  {
    "type": "currently_into",
    "enabled": true,
    "text": "Deep into 70s prog rock and Korean cinema right now."
  }
]
```

### Item-Level Privacy

```sql
ALTER TABLE media_items ADD COLUMN private BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX idx_media_items_private ON media_items(user_id, private)
    WHERE private = true;
```

Items marked `private = true` are excluded from all share profile calculations.

---

## Rendering

### Server-Side

The share page is server-rendered (Templ + HTMX). No client-side data fetching
for public pages — everything is pre-computed.

### Query Pattern

Each block type has a query function that respects exclusions:

```go
func TopGenres(ctx context.Context, userID uuid.UUID, cfg BlockConfig, exclusions Exclusions) ([]GenreCount, error) {
    // Query media_item_tags joined with media_items
    // WHERE user_id = $1
    //   AND consumed_at within time_range
    //   AND platform NOT IN (excluded_platforms)
    //   AND private = false
    //   AND tag NOT IN (excluded_tags)
    // GROUP BY tag, ORDER BY count DESC
    // LIMIT cfg.Count
}
```

### Preview

Before publishing, users see an exact preview of their share page:

```
GET /settings/share/preview    (authenticated, shows exactly what /share/{slug} will show)
```

---

## Routes

```go
// Public (no auth)
r.Get("/share/{slug}", handlers.PublicProfile)

// Authenticated
r.Route("/settings/share", func(r chi.Router) {
    r.Get("/", handlers.ShareSettings)            // configure blocks
    r.Put("/", handlers.UpdateShareSettings)       // save config
    r.Get("/preview", handlers.SharePreview)       // preview public page
    r.Post("/enable", handlers.EnableProfile)      // publish
    r.Post("/disable", handlers.DisableProfile)    // unpublish
})

// Item privacy (from timeline view)
r.Post("/api/items/{id}/private", handlers.ToggleItemPrivate)
```

---

## Open Questions from MVP.md

> **Sharing granularity — share everything, or curate what's public?**

**Answer**: Curated. Block-based composition with platform/tag/item exclusions.
Default to private, opt-in to sharing. Aggregates preferred over individual items.
