# Self-Hosting Guide

Deploy Specto on your own server. This guide covers building, configuring,
and running a production instance.

---

## Prerequisites

- **Go 1.25+** (to build from source)
- **PostgreSQL 16+**
- **[mise](https://mise.jdx.dev/)** (for tooling and code generation)

Or use Docker Compose which bundles PostgreSQL for you.

---

## 1. Build

```bash
# Install tools (templ, sqlc, migrate, etc.)
mise install

# Generate code and run migrations
mise run setup

# Build the production binary
mise run build
# → outputs bin/app
```

---

## 2. Database

### Option A: Docker Compose

```bash
docker-compose up -d
```

This starts PostgreSQL 16 on `localhost:5432` with database `specto_dev`,
user `postgres`, password `postgres`. For production, change these defaults.

### Option B: External PostgreSQL

Point `DATABASE_URL` to any PostgreSQL 16+ instance:

```
DATABASE_URL=postgres://user:password@host:5432/specto?sslmode=require
```

### Run Migrations

```bash
mise run db-migrate
```

---

## 3. Environment Variables

### Required

| Variable | Description | Example |
|---|---|---|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://user:pass@host:5432/specto?sslmode=require` |
| `ENCRYPTION_KEY` | 64 hex characters, encrypts stored OAuth tokens | Generate with `openssl rand -hex 32` |
| `SESSION_SECRET` | 32+ bytes, signs session cookies | Generate with `openssl rand -base64 32` |
| `CSRF_KEY` | Exactly 32 bytes, CSRF protection | Generate with `openssl rand -base64 32` (use first 32 chars) |
| `PORT` | HTTP listen port | `3000` |
| `BASE_URL` | Public URL of your instance | `https://specto.example.com` |

### Optional: OAuth & API Keys

All OAuth providers and enrichment APIs are optional. If a credential is
unset, that provider is simply not registered.

See **[API Key & Credentials Setup Guide](api-key-setup.md)** for
step-by-step instructions on obtaining credentials for each provider.

For production, replace all `http://localhost:3000` redirect URIs with your
actual domain using `https://`.

**Quick reference of env vars:**

```bash
# App OAuth (user login) — enables "Sign in with Google/GitHub"
GOOGLE_CLIENT_ID=""
GOOGLE_CLIENT_SECRET=""
GITHUB_CLIENT_ID=""
GITHUB_CLIENT_SECRET=""

# Plugin OAuth (platform connections)
SPOTIFY_CLIENT_ID=""
SPOTIFY_CLIENT_SECRET=""
YOUTUBE_CLIENT_ID=""          # same as GOOGLE_CLIENT_ID
YOUTUBE_CLIENT_SECRET=""      # same as GOOGLE_CLIENT_SECRET

# Enrichment APIs
LASTFM_API_KEY=""
TMDB_API_KEY=""
OMDB_API_KEY=""               # optional, supplements TMDB
IGDB_CLIENT_ID=""             # Twitch client ID
IGDB_CLIENT_SECRET=""         # Twitch client secret
# AniList — no key needed
# MusicBrainz — no key needed

# LLM enrichment
LLM_PROVIDER=""               # googlegenai | openai
LLM_MODEL=""                  # e.g. gemini-2.0-flash, gpt-4o-mini
LLM_API_KEY=""
LLM_BASE_URL=""               # optional: for Ollama or custom endpoints

# Enrichment worker tuning
ENRICHMENT_BATCH_SIZE="50"
ENRICHMENT_POLL_INTERVAL="5s"
ENRICHMENT_MAX_RETRIES="3"
ENRICHMENT_MIN_CONFIDENCE="0.7"
```

---

## 4. Run

```bash
# Start the server
./bin/app
```

The app listens on `PORT` (default `3000`). Put a reverse proxy (nginx,
Caddy, etc.) in front for TLS termination in production.

### Reverse Proxy Example (Caddy)

```
specto.example.com {
    reverse_proxy localhost:3000
}
```

---

## 5. Verify

1. Visit `https://your-domain.com` — you should see the login page
2. Create an account with email/password or OAuth (if configured)
3. Connect a platform plugin (Spotify, YouTube) from the dashboard
4. Trigger a sync and confirm media items appear
