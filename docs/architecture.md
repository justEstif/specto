# Architecture

## Overview

Media Consumption Analysis is a **single-user, self-hosted** application that aggregates
media consumption data from multiple platforms, enriches it with topic/genre metadata,
and surfaces insights through a web interface.

The system is designed around three principles:

1. **Plugin-driven ingestion** — Every data source is a plugin. Core knows nothing about Spotify or YouTube.
2. **Decoupled enrichment** — Raw data is stored first, enriched separately. Enrichment can fail without losing data.
3. **Client-agnostic core** — Core logic and HTTP server are separate layers. The web UI is just one possible client.

---

## Deployment Model

**Hosted, multi-tenant.** One instance serves multiple users. You deploy it, others sign
up and connect their own platform accounts.

Deployable as:
- Docker container (Compose with Postgres)
- Single binary + external Postgres (Fly.io, Railway, bare metal, etc.)

### Multi-tenancy Requirements

- **User accounts** — Sign up / login (OAuth via Google/GitHub, or email/password)
- **Data isolation** — All queries scoped by `user_id`. No user can see another's data.
- **Token encryption** — OAuth tokens encrypted at rest, per-user encryption keys
- **Per-user rate limiting** — Sync rate limits scoped per user per plugin
- **Storage awareness** — Track per-user item counts (quotas can be added later if needed)

---

## System Layers

```
┌──────────────────────────────────────────────────────────────────┐
│                         Client Layer                             │
│              (Web UI, CLI, or any HTTP consumer)                 │
│                  Talks to Server via JSON API                    │
└──────────────────────────┬───────────────────────────────────────┘
                           │ HTTP (JSON)
┌──────────────────────────▼───────────────────────────────────────┐
│                         Server Layer                             │
│               HTTP API — routes, auth, validation                │
│             No domain logic — delegates to Core                  │
└──────────────────────────┬───────────────────────────────────────┘
                           │ Go function calls
┌──────────────────────────▼───────────────────────────────────────┐
│                          Core Layer                              │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────────┐  │
│  │   Plugin     │  │  Enrichment  │  │   Insights / Query     │  │
│  │   System     │  │  Pipeline    │  │   Engine               │  │
│  └──────┬──────┘  └──────┬───────┘  └────────────┬───────────┘  │
│         │                │                        │              │
│  ┌──────▼────────────────▼────────────────────────▼───────────┐  │
│  │                    Store (DB layer)                         │  │
│  │              PostgreSQL + jsonb + migrations                │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

### Client Layer

The web UI is a consumer of the Server's JSON API, not part of the core application.
For MVP: Templ + HTMX. Templ gives type-safe, component-based HTML that compiles to Go —
no runtime template errors. But the API contract means we can swap to a SPA, CLI,
or mobile app without touching Core or Server.

### Server Layer

Thin HTTP layer. Responsibilities:

- Route requests to Core methods
- Handle OAuth callback routes (delegating token exchange to plugins)
- Validate request parameters
- Serialize responses as JSON
- Serve the web client (static files / templates)

**Does NOT** contain domain logic, sync orchestration, or enrichment logic.

### Core Layer

All domain logic lives here. Four sub-systems:

#### Plugin System

Manages the lifecycle of source plugins. See [plugin-guide.md](./plugin-guide.md) for
the full interface contract.

```go
type SourcePlugin interface {
    Name() string
    AuthType() AuthType                                          // OAuth, FileImport, APIKey, None
    AuthConfig() *OAuthConfig                                    // OAuth details (if applicable)
    Sync(ctx context.Context, creds Credentials) ([]MediaItem, error)
    Enrich(ctx context.Context, items []MediaItem) ([]MediaItem, error) // optional
}
```

The Plugin Registry holds registered plugins and manages:
- Plugin discovery and registration (compile-time for MVP)
- Per-plugin credential storage
- Sync invocation and rate limiting

#### Enrichment (two layers)

Enrichment is split between plugins and core:

**Plugin enrichment** — Each plugin's `Enrich()` calls platform-specific APIs
(Spotify plugin → Last.fm/MusicBrainz, Netflix plugin → TMDB). Plugins own their
domain-specific enrichment. Adds authoritative genre/format tags.

**Core enrichment** — Universal LLM-based classification via Genkit (Google's AI
framework for Go). Runs on ALL items after plugin enrichment. Fills mood/topic gaps.
Only assigns from a **fixed tag set** — no runtime tag creation.

```go
// Core's LLM enricher uses Genkit for structured output
result, _, err := genkit.GenerateData[TagResult](ctx, g,
    ai.WithModel(model),
    ai.WithPrompt(buildTagPrompt(item, existingTags)),
)
```

See [enrichment.md](./enrichment.md) for the full pipeline, tag taxonomy, and
enrichment source details.

#### Insights / Query Engine

Aggregates enriched data for the dashboard:
- Time-series consumption patterns
- Topic/genre distribution
- Platform breakdown
- Content type split (music vs video vs article vs podcast)

For MVP, this is likely direct SQL queries behind Go functions. No OLAP engine needed
at single-user scale.

#### Store (DB Layer)

PostgreSQL with a repository pattern. Core tables:

- `users` — user accounts, auth method, profile settings
- `media_items` — normalized consumption data, scoped by `user_id` (see [schema.md](./schema.md))
- `plugin_configs` — per-user, per-plugin credentials and settings (tokens encrypted)
- `sync_log` — per-user sync history, timestamps, item counts, errors
- `tags` — tag taxonomy and aliases

All platform-specific data lives in `raw_metadata` (jsonb). Structured fields are only
for data that's universal across platforms.

---

## Data Flow

### Sync Flow (user-triggered)

```
User clicks "Sync Spotify"
        │
        ▼
   Server: POST /api/sync/spotify
        │
        ▼
   Core: Check rate limit (last sync timestamp)
        │
        ├── Too soon → 429 (rate limited)
        │
        ▼
   Core: plugin.Sync(ctx, creds)
        │
        ▼
   Plugin: Call Spotify API → normalize to []MediaItem
        │
        ▼
   Core: Store raw MediaItems (status: "pending")
        │
        ▼
   Plugin: plugin.Enrich(ctx, creds, items) — platform-specific (Last.fm, TMDB, etc.)
        │
        ▼
   Core: Store items (status: "plugin-enriched")
        │
        ▼
   Core: LLM enrichment via Genkit — universal mood/topic classification
        │
        ▼
   Core: Validate tags against fixed set, resolve aliases
        │
        ▼
   Core: Update items (status: "enriched")
        │
        ▼
   Server: Return sync summary (items added, enriched, errors)
```

**Enrichment is inline-after-sync for MVP** but decoupled in code. Plugin enrichment
runs first (platform-specific), then core LLM enrichment (universal). Moving to a
background worker later means changing *when* it's called, not *how*.

**Future: cron sync.** The plugin interface is trigger-agnostic — `Sync()` doesn't know
or care if it was called by an HTTP handler or a scheduler. Adding cron is ~100-200 lines
wrapping the same sync path with `robfig/cron` or similar.

### File Import Flow

```
User uploads Netflix CSV
        │
        ▼
   Server: POST /api/import/netflix (multipart file)
        │
        ▼
   Core: plugin.Sync(ctx, creds{File: uploadedFile})
        │
        ▼
   Plugin: Parse CSV → normalize to []MediaItem
        │
        ▼
   (same enrichment + store flow as above)
```

File import plugins use the same `Sync()` interface — the "credentials" just include
a file handle instead of OAuth tokens.

---

## Rate Limiting

Per-plugin rate limits to respect upstream APIs and prevent accidental spam:

| Scope | Strategy |
|-------|----------|
| **Sync trigger** | Minimum interval between syncs per user per plugin (e.g., 15 min). Stored in `sync_log`. |
| **Upstream API** | Per-plugin responsibility. Plugins handle their own backoff/retry. Core provides a shared HTTP client with configurable rate limiting. |
| **Global** | Optional global rate limits to protect shared API keys (if any) across all users. |

---

## Auth Architecture

Two levels of auth:

1. **App-level (user accounts)** — Users sign up and log in to the application. Supported
   methods:
   - OAuth login (Google / GitHub) — recommended, avoids password management
   - Email + password (optional fallback)
   - Session-based auth (HTTP-only cookies) for the web UI
   - API key or Bearer token for programmatic access

2. **Platform OAuth (per-user, per-plugin)** — Each OAuth plugin defines its scopes and
   callback path. The Server handles the OAuth redirect dance; the plugin handles token
   exchange and refresh. Tokens stored **encrypted at rest** in `plugin_configs`, scoped
   to the authenticated user.

All data access goes through a `user_id` scope — no query touches the DB without it.

See [auth.md](./auth.md) for details.

---

## Shareable Profiles (M7)

Public read-only pages served at `/share/<profile-slug>`. The owner configures:
- Which platforms/content types to include
- Time range
- Whether to show specific items or only aggregates

No authentication required to view. No write access. Essentially a static snapshot
regenerated on demand or on sync.

See [sharing.md](./sharing.md) for the privacy model.

---

## Project Structure (proposed)

```
media-consumption-analysis/
├── cmd/
│   └── server/              # main.go — binary entrypoint
├── internal/
│   ├── core/
│   │   ├── plugin.go        # SourcePlugin interface, registry
│   │   ├── enrichment.go    # Enricher interface, pipeline
│   │   ├── sync.go          # sync orchestration, rate limiting
│   │   ├── insights.go      # query/aggregation logic
│   │   └── store/           # DB layer, repositories, migrations
│   ├── server/
│   │   ├── routes.go        # HTTP API routes
│   │   ├── handlers.go      # request handlers (delegate to core)
│   │   └── middleware.go     # auth, logging, etc.
│   └── plugins/
│       ├── spotify/          # Spotify plugin
│       ├── youtube/          # YouTube plugin
│       └── netflix/          # Netflix plugin
├── web/                      # client: templates, static assets
├── docs/                     # architecture, guides
├── migrations/               # SQL migrations
└── config.example.yaml       # configuration template
```

---

## Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Multi-tenant | Yes, from day one | Shared hosted service. All data scoped by `user_id`. |
| Plugin interface | `Sync()` returns `[]MediaItem` | Simple, trigger-agnostic. Works for OAuth, file import, manual. |
| Enrichment | Two layers: plugin-specific + core LLM | Plugins own their domain, core is slim. Fixed tag set for deterministic insights. |
| Sync trigger | User-initiated + rate limit | Simplest correct behavior. Cron drops in later without interface changes. |
| Client coupling | JSON API boundary | Core doesn't know about HTTP. Server doesn't know about domain logic. Client is replaceable. |
| Raw metadata | jsonb column | Platform-specific fields without schema bloat. Structured fields only for universal data. |
