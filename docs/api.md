# API

## Overview

This document defines the **canonical HTTP API surface** between the Server layer and any client
(web UI, HTMX interactions, CLI, future SPA/mobile app).

The goal is to keep the API **small, consistent, and plugin-centric**:

- clients should not need to know plugin-specific auth quirks
- the server should expose a stable JSON contract even if internals change
- route shape should minimize change amplification across frontend, server, and docs

This is an **internal product API**, not a public third-party integration surface.

---

## Design Principles

Borrowing from *A Philosophy of Software Design*, the API should be a **deep module**:

a small number of obvious endpoints hiding the messy details of OAuth, file imports,
sync cursors, enrichment status, and per-plugin differences.

### Chosen shape: plugin-centric resource API

We group most platform operations under:

```
/api/v1/plugins/{plugin}/...
```

This keeps the interface simple for clients:
- the client identifies the plugin once
- the same route family works for Spotify, YouTube, Netflix, etc.
- auth type differences are pushed downward into server/core/plugin implementations

### Rejected alternative: many top-level route families

Example:
- `/connect/{plugin}/login`
- `/sync/{plugin}`
- `/import/{plugin}`
- `/disconnect/{plugin}`

This works, but it leaks implementation categories into the interface and increases
cognitive load. A plugin-centric API is easier to discover and document.

---

## API Conventions

### Base path

All JSON endpoints live under:

```
/api/v1
```

### Authentication

- Browser clients use the app session cookie
- Mutating requests require CSRF protection
- Future programmatic clients may use bearer tokens, but session auth is the MVP default

### Content types

Requests and responses use JSON unless explicitly documented otherwise.

```http
Content-Type: application/json
Accept: application/json
```

File uploads use `multipart/form-data`.

### Time format

All timestamps are RFC3339 UTC strings.

Example:

```json
"2026-03-11T16:00:00Z"
```

### Pagination

Collection endpoints use cursor or offset/limit depending on access pattern:

- timeline-style feeds: `limit` + `offset` for MVP simplicity
- plugin sync state: no pagination
- future large collections may adopt cursor pagination

### Error shape

All non-2xx JSON responses should use a consistent envelope:

```json
{
  "error": {
    "code": "rate_limit",
    "message": "Spotify rate limit hit",
    "details": {
      "retry_after_seconds": 30
    }
  }
}
```

Common error codes:

- `unauthorized`
- `forbidden`
- `not_found`
- `validation_error`
- `rate_limit`
- `auth_expired`
- `permission_denied`
- `file_parse_error`
- `upstream_error`
- `internal_error`

### Success envelopes

For single resources:

```json
{
  "data": { ... }
}
```

For collections:

```json
{
  "data": [ ... ],
  "meta": {
    "limit": 50,
    "offset": 0,
    "total": 231
  }
}
```

For actions:

```json
{
  "data": {
    "status": "ok"
  }
}
```

---

## Boundary: HTML routes vs JSON API

Not every route in the app should be part of the JSON API.

### HTML / browser-navigation routes

These render pages or perform OAuth redirects:

- `/login`
- `/auth/google/login`
- `/auth/google/callback`
- `/auth/github/login`
- `/auth/github/callback`
- `/share/{slug}`
- `/settings`
- `/settings/share`

### JSON API routes

These power authenticated interactions and future non-HTML clients:

- `/api/v1/session`
- `/api/v1/plugins`
- `/api/v1/plugins/{plugin}`
- `/api/v1/plugins/{plugin}/connect`
- `/api/v1/plugins/{plugin}/import`
- `/api/v1/plugins/{plugin}/disconnect`
- `/api/v1/plugins/{plugin}/sync`
- `/api/v1/timeline`
- `/api/v1/insights/*`
- `/api/v1/share-profile`
- `/api/v1/items/{id}/privacy`

This split keeps page rendering concerns out of the client API while still allowing the
web UI to call a clean JSON layer.

---

## Resource Model

The main resources exposed to clients are:

- **session** — who is logged in
- **plugins** — connection state and sync state per platform
- **timeline items** — normalized consumed media
- **insights** — aggregates derived from media items and tags
- **share profile** — public profile configuration

---

## Session API

### `GET /api/v1/session`

Returns the currently authenticated user and high-level app state.

#### Response

```json
{
  "data": {
    "user": {
      "id": "usr_123",
      "email": "user@example.com",
      "display_name": "Estifanos",
      "avatar_url": "https://...",
      "profile_slug": "estifanos"
    }
  }
}
```

### `DELETE /api/v1/session`

Logs the user out.

#### Response

```json
{
  "data": {
    "status": "logged_out"
  }
}
```

---

## Plugin API

Plugins are the most important API surface because they unify OAuth sources,
file imports, and future API-key integrations.

### Plugin object

```json
{
  "name": "spotify",
  "display_name": "Spotify",
  "auth_type": "oauth",
  "status": "connected",
  "enabled": true,
  "connected": true,
  "last_synced_at": "2026-03-11T15:20:00Z",
  "error_message": null,
  "sync": {
    "status": "idle",
    "last_run_status": "success"
  }
}
```

### `GET /api/v1/plugins`

List all registered plugins with per-user state.

#### Response

```json
{
  "data": [
    {
      "name": "spotify",
      "display_name": "Spotify",
      "auth_type": "oauth",
      "status": "connected",
      "enabled": true,
      "connected": true,
      "last_synced_at": "2026-03-11T15:20:00Z",
      "error_message": null
    },
    {
      "name": "netflix",
      "display_name": "Netflix",
      "auth_type": "file_import",
      "status": "disconnected",
      "enabled": true,
      "connected": false,
      "last_synced_at": null,
      "error_message": null
    }
  ]
}
```

### `GET /api/v1/plugins/{plugin}`

Returns a single plugin's state and capabilities.

#### Response

```json
{
  "data": {
    "name": "youtube",
    "display_name": "YouTube",
    "auth_type": "oauth",
    "status": "connected",
    "connected": true,
    "capabilities": {
      "can_connect": true,
      "can_disconnect": true,
      "can_import": false,
      "can_sync": true,
      "supports_incremental_sync": true
    }
  }
}
```

### `POST /api/v1/plugins/{plugin}/connect`

Starts an OAuth connection flow.

For browser clients, this returns a redirect URL rather than forcing route knowledge
into the frontend.

#### Response

```json
{
  "data": {
    "redirect_url": "https://accounts.spotify.com/authorize?..."
  }
}
```

#### Notes

- Only valid for `auth_type = oauth`
- The actual callback target remains server-owned
- The client should redirect the browser to `redirect_url`

### `POST /api/v1/plugins/{plugin}/import`

Uploads a file for file-import plugins.

#### Request

`multipart/form-data`

Fields:
- `file` — required

#### Response

```json
{
  "data": {
    "plugin": "netflix",
    "status": "connected",
    "imported": true
  }
}
```

#### Notes

- For MVP, import and sync can be the same operation
- Keeping a separate import endpoint leaves room for validation-only flows later

### `DELETE /api/v1/plugins/{plugin}/disconnect`

Disconnects a plugin and deletes stored credentials.

#### Response

```json
{
  "data": {
    "plugin": "spotify",
    "status": "disconnected"
  }
}
```

### `POST /api/v1/plugins/{plugin}/sync`

Triggers a sync for a connected plugin.

#### Response

```json
{
  "data": {
    "plugin": "spotify",
    "status": "success",
    "items_added": 42,
    "items_skipped": 10,
    "items_updated": 3,
    "enrichment_status": "completed",
    "last_synced_at": "2026-03-11T16:10:00Z"
  }
}
```

#### Possible status values

- `success`
- `partial`
- `rate_limited`
- `failed`

#### Rate-limited example

```json
{
  "error": {
    "code": "rate_limit",
    "message": "Plugin sync is temporarily rate-limited",
    "details": {
      "retry_after_seconds": 900
    }
  }
}
```

### `GET /api/v1/plugins/{plugin}/sync-history`

Returns recent sync runs for this plugin.

#### Response

```json
{
  "data": [
    {
      "started_at": "2026-03-11T16:00:00Z",
      "completed_at": "2026-03-11T16:00:18Z",
      "status": "success",
      "items_added": 42,
      "items_skipped": 10,
      "items_updated": 3,
      "error_code": null,
      "error_message": null
    }
  ]
}
```

---

## Timeline API

The timeline is the normalized feed of consumed media.

### Timeline item object

```json
{
  "id": "itm_123",
  "platform": "spotify",
  "type": "music",
  "title": "Breathe",
  "creator": "Pink Floyd",
  "consumed_at": "2026-03-10T22:04:00Z",
  "duration_seconds": 163,
  "time_spent_seconds": 163,
  "url": "https://open.spotify.com/track/...",
  "external_id": "spotify:track:...",
  "enrichment_status": "enriched",
  "private": false,
  "tags": [
    {
      "name": "progressive-rock",
      "category": "genre",
      "source": "spotify",
      "confidence": null
    },
    {
      "name": "melancholic",
      "category": "mood",
      "source": "llm",
      "confidence": 0.82
    }
  ]
}
```

### `GET /api/v1/timeline`

Returns paginated items for the dashboard timeline.

#### Query params

- `from` — optional RFC3339 timestamp
- `to` — optional RFC3339 timestamp
- `platform` — optional repeated or comma-separated filter
- `type` — optional repeated or comma-separated filter
- `q` — optional full-text query over title/creator
- `limit` — default `50`, max `100`
- `offset` — default `0`

#### Example

```http
GET /api/v1/timeline?platform=spotify,youtube&type=music,video&limit=50&offset=0
```

#### Response

```json
{
  "data": [
    {
      "id": "itm_123",
      "platform": "spotify",
      "type": "music",
      "title": "Breathe",
      "creator": "Pink Floyd",
      "consumed_at": "2026-03-10T22:04:00Z",
      "enrichment_status": "enriched",
      "private": false,
      "tags": ["progressive-rock", "melancholic"]
    }
  ],
  "meta": {
    "limit": 50,
    "offset": 0,
    "total": 231
  }
}
```

### `POST /api/v1/items/{id}/privacy`

Sets whether an item should be excluded from sharing.

#### Request

```json
{
  "private": true
}
```

#### Response

```json
{
  "data": {
    "id": "itm_123",
    "private": true
  }
}
```

---

## Insights API

Insights are aggregates computed from normalized items and tags.

### `GET /api/v1/insights/summary`

Returns top-level dashboard numbers.

#### Response

```json
{
  "data": {
    "total_items": 4218,
    "total_time_spent_seconds": 948322,
    "top_platform": "spotify",
    "top_type": "music"
  }
}
```

### `GET /api/v1/insights/platform-breakdown`

#### Response

```json
{
  "data": [
    {
      "platform": "spotify",
      "type": "music",
      "count": 1880,
      "total_duration_seconds": 502311
    },
    {
      "platform": "youtube",
      "type": "video",
      "count": 740,
      "total_duration_seconds": 183000
    }
  ]
}
```

### `GET /api/v1/insights/tags`

Returns aggregate tag counts.

#### Query params

- `category` — optional (`genre`, `topic`, `mood`, `format`)
- `from` / `to` — optional time window
- `limit` — default `20`

#### Response

```json
{
  "data": [
    {
      "name": "rock",
      "category": "genre",
      "count": 184
    },
    {
      "name": "science",
      "category": "topic",
      "count": 91
    }
  ]
}
```

### `GET /api/v1/insights/timeline`

Returns time-bucketed consumption data for charts.

#### Query params

- `bucket` — `day`, `week`, or `month`
- `from` / `to` — optional
- `platform` — optional filter
- `type` — optional filter

#### Response

```json
{
  "data": [
    {
      "bucket_start": "2026-03-01T00:00:00Z",
      "count": 42,
      "time_spent_seconds": 17280
    },
    {
      "bucket_start": "2026-03-02T00:00:00Z",
      "count": 35,
      "time_spent_seconds": 14400
    }
  ]
}
```

---

## Share Profile API

This API manages the authenticated user's public profile configuration.

### Share profile object

```json
{
  "enabled": true,
  "profile_slug": "estifanos",
  "excluded_platforms": ["netflix"],
  "excluded_tags": ["romance"],
  "blocks": [
    {
      "type": "top_genres",
      "enabled": true,
      "time_range": "30d"
    },
    {
      "type": "top_creators",
      "enabled": true,
      "time_range": "30d",
      "count": 10,
      "platforms": ["spotify", "youtube"]
    }
  ]
}
```

### `GET /api/v1/share-profile`

Returns the current user's share configuration.

### `PUT /api/v1/share-profile`

Replaces the current user's share configuration.

#### Request

```json
{
  "enabled": true,
  "excluded_platforms": ["netflix"],
  "excluded_tags": ["romance"],
  "blocks": [
    {
      "type": "top_genres",
      "enabled": true,
      "time_range": "30d"
    }
  ]
}
```

#### Response

```json
{
  "data": {
    "enabled": true,
    "profile_slug": "estifanos"
  }
}
```

### `GET /api/v1/share-profile/preview`

Returns the exact block data that would render on the public profile.

#### Response

```json
{
  "data": {
    "blocks": [
      {
        "type": "top_genres",
        "title": "Top Genres",
        "items": [
          { "name": "rock", "count": 45 },
          { "name": "electronic", "count": 20 }
        ]
      }
    ]
  }
}
```

---

## Suggested Status Codes

| Status | When |
|--------|------|
| `200 OK` | Successful read or action |
| `201 Created` | New server-side resource created |
| `204 No Content` | Delete/logout with no body |
| `400 Bad Request` | Invalid request shape or params |
| `401 Unauthorized` | No valid session |
| `403 Forbidden` | Valid session, insufficient access |
| `404 Not Found` | Resource or plugin not found |
| `409 Conflict` | Action invalid in current state |
| `422 Unprocessable Entity` | Validation error for syntactically valid input |
| `429 Too Many Requests` | App or upstream rate limit |
| `500 Internal Server Error` | Unexpected server failure |
| `502 Bad Gateway` | Upstream platform failure surfaced by server |

---

## Notes on Canonicalization

Some existing docs show route examples such as:

- `/connect/{plugin}/login`
- `/sync/{plugin}`
- `/settings/share/...`
- `/api/items/{id}/private`

Those examples are useful, but this file is the **canonical API surface** for the
client/server boundary going forward. If implementation or other docs diverge, this
file should be treated as the source of truth and the others should be aligned.

---

## Why this API is intentionally small

A good API should reduce cognitive load, not mirror every internal component.

This surface stays deliberately compact by:
- exposing **plugin** operations instead of provider-specific flows
- exposing **insights** as aggregates rather than leaking query internals
- exposing **share profile** as one document-shaped resource instead of many tiny endpoints
- separating **HTML routes** from **JSON routes** so clients know which layer they are using

If we later add background jobs, new plugins, or a SPA frontend, clients should still
mostly interact with the same resource shapes defined here.
