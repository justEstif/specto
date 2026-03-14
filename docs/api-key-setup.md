# API Key & Credentials Setup Guide

This guide walks you through obtaining API keys and OAuth credentials for each
provider Specto uses. You only need to set up the providers you want — all
are optional.

Once you have the credentials, uncomment and fill in the values in `mise.toml`.

---

## Table of Contents

- [OAuth Providers](#oauth-providers)
  - [Google (App Login + YouTube Plugin)](#google-app-login--youtube-plugin)
  - [GitHub (App Login)](#github-app-login)
  - [Spotify (Plugin Connection)](#spotify-plugin-connection)
- [Enrichment API Providers](#enrichment-api-providers)
  - [Last.fm](#lastfm)
  - [TMDB](#tmdb)
  - [OMDB](#omdb)
  - [Podcast Index](#podcast-index)
  - [IGDB (Twitch)](#igdb-twitch)
  - [AniList](#anilist)
  - [MusicBrainz](#musicbrainz)
- [LLM Enrichment](#llm-enrichment)
  - [Google Gemini (googlegenai)](#google-gemini-googlegenai)
  - [Ollama (local)](#ollama-local)
- [Summary: Redirect URIs](#summary-redirect-uris)
- [Summary: Environment Variables](#summary-environment-variables)

---

## OAuth Providers

### Google (App Login + YouTube Plugin)

Google Cloud Console credentials are shared between Google OAuth login and
the YouTube API plugin. You create **one** OAuth client in Google Cloud and
use the same Client ID/Secret for both `GOOGLE_*` and `YOUTUBE_*` env vars.

#### 1. Create a Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Click the project dropdown at the top → **New Project**
3. Name it something like `specto-dev`, click **Create**
4. Select the newly created project from the dropdown

#### 2. Enable Required APIs

1. Go to **APIs & Services → Library** ([direct link](https://console.cloud.google.com/apis/library))
2. Search for and enable each:
   - **YouTube Data API v3** (for the YouTube plugin)
   - No additional API needed for Google Sign-In — it uses the default OAuth2 userinfo endpoint

#### 3. Configure the OAuth Consent Screen

1. Go to **APIs & Services → OAuth consent screen** ([direct link](https://console.cloud.google.com/apis/credentials/consent))
2. Choose **External** user type (unless this is for an internal org), click **Create**
3. Fill in required fields:
   - **App name**: `Specto` (or whatever you like)
   - **User support email**: your email
   - **Developer contact email**: your email
4. Click **Save and Continue**
5. On the **Scopes** step, click **Add or Remove Scopes** and add:
   - `openid`
   - `email`
   - `profile`
   - `https://www.googleapis.com/auth/youtube.readonly`
6. Click **Save and Continue** through the remaining steps
7. Under **Test users**, add your own Google email (while the app is in "Testing" mode, only listed test users can log in)

#### 4. Create OAuth Client Credentials

1. Go to **APIs & Services → Credentials** ([direct link](https://console.cloud.google.com/apis/credentials))
2. Click **+ Create Credentials → OAuth client ID**
3. Application type: **Web application**
4. Name: `Specto Local Dev`
5. Under **Authorized redirect URIs**, add both:
   ```
   http://localhost:3000/auth/google/callback
   http://localhost:3000/api/v1/plugins/youtube-api/callback
   ```
6. Click **Create**
7. Copy the **Client ID** and **Client Secret** (the secret is only shown once since April 2025 — save it immediately)

#### 5. Set Environment Variables

```toml
# In mise.toml
GOOGLE_CLIENT_ID = "your-client-id.apps.googleusercontent.com"
GOOGLE_CLIENT_SECRET = "GOCSPX-your-client-secret"
YOUTUBE_CLIENT_ID = "your-client-id.apps.googleusercontent.com"     # same as GOOGLE_CLIENT_ID
YOUTUBE_CLIENT_SECRET = "GOCSPX-your-client-secret"                 # same as GOOGLE_CLIENT_SECRET
```

> **Note**: `GOOGLE_*` is used for app login (Sign in with Google).
> `YOUTUBE_*` is used for the YouTube plugin OAuth connection. They use
> the same Google Cloud OAuth client but are read separately so you could
> use different clients if needed.

---

### GitHub (App Login)

#### 1. Create a GitHub OAuth App

1. Go to [GitHub → Settings → Developer settings → OAuth Apps](https://github.com/settings/developers)
   - Or: click your profile picture → **Settings** → scroll down to **Developer settings** → **OAuth Apps**
2. Click **New OAuth App** (or **Register a new application** if it's your first)
3. Fill in:
   - **Application name**: `Specto Local Dev`
   - **Homepage URL**: `http://localhost:3000`
   - **Authorization callback URL**: `http://localhost:3000/auth/github/callback`
4. Click **Register application**

#### 2. Get Client ID and Secret

1. After registration, you'll see the **Client ID** on the app page
2. Click **Generate a new client secret**
3. Copy the secret immediately (it won't be shown again)

#### 3. Set Environment Variables

```toml
# In mise.toml
GITHUB_CLIENT_ID = "your-github-client-id"
GITHUB_CLIENT_SECRET = "your-github-client-secret"
```

---

### Spotify (Plugin Connection)

#### 1. Create a Spotify App

1. Go to [Spotify Developer Dashboard](https://developer.spotify.com/dashboard)
2. Log in with your Spotify account (or create one)
3. Click **Create App**
4. Fill in:
   - **App name**: `Specto`
   - **App description**: `Media consumption tracker`
   - **Redirect URI**: `http://127.0.0.1:3000/api/v1/plugins/spotify-api/callback`
   - Check the **Developer Terms of Service** checkbox
5. Click **Create**

#### 2. Get Client ID and Secret

1. On the app overview page, you'll see the **Client ID**
2. Click **Show Client Secret** to reveal the secret
3. Copy both values

#### 3. Configure Redirect URI (if not set during creation)

1. On the app overview page, click **Edit Settings**
2. Under **Redirect URIs**, add:
   ```
   http://localhost:3000/api/v1/plugins/spotify-api/callback
   ```
3. Click **Save**

#### 4. Set Environment Variables

```toml
# In mise.toml
SPOTIFY_CLIENT_ID = "your-spotify-client-id"
SPOTIFY_CLIENT_SECRET = "your-spotify-client-secret"
```

> **Note**: Spotify apps start in **Development Mode**, limited to 25
> users. This is fine for local development and testing. If you need more
> users, request a quota extension from the dashboard.

---

## Enrichment API Providers

These providers add metadata (genres, tags, duration, ratings) to your media
items. Each is optional — if the API key is not set, the provider is simply
not registered and items of that type skip that enrichment source.

### Last.fm

Enriches **music** items (Spotify tracks) with genre and tag data from
Last.fm's community-driven tag database.

**Rate limit**: 5 requests/second.

#### 1. Create an API Account

1. Go to [Last.fm API Account Creation](https://www.last.fm/api/account/create)
2. Log in with your Last.fm account (or create one at [last.fm/join](https://www.last.fm/join))
3. Fill in:
   - **Application name**: `Specto`
   - **Application description**: `Media consumption tracker`
   - **Application homepage**: `http://localhost:3000` (or leave blank)
   - **Callback URL**: leave blank (not needed — we only use the REST API)
4. Click **Submit**

#### 2. Get Your API Key

1. After submission, you'll see your **API Key** and **Shared Secret**
2. Copy the **API Key** (the shared secret is not needed for read-only tag queries)

#### 3. Set Environment Variable

```toml
# In mise.toml
LASTFM_API_KEY = "your-lastfm-api-key"
```

---

### TMDB

Enriches **movie** and **TV** items (Netflix, Prime Video) with genres,
keywords, ratings, and metadata from The Movie Database.

**Rate limit**: ~40 requests per 10 seconds (per API key).

#### 1. Create a TMDB Account

1. Go to [TMDB Sign Up](https://www.themoviedb.org/signup)
2. Create an account and verify your email

#### 2. Request an API Key

1. Go to [TMDB API Settings](https://www.themoviedb.org/settings/api)
2. Click **Create** under the API section
3. Choose **Developer** (not Professional)
4. Fill in the application form:
   - **Type of use**: Personal
   - **Application name**: `Specto`
   - **Application URL**: `http://localhost:3000`
   - **Application summary**: `Media consumption tracker — enriches movie/TV metadata`
5. Accept the terms of use
6. Your **API Key (v3 auth)** will be shown on the API page

#### 3. Set Environment Variable

```toml
# In mise.toml
TMDB_API_KEY = "your-tmdb-api-key"
```

---

### OMDB

Optional supplement to TMDB. Adds IMDb, Rotten Tomatoes, and Metacritic
ratings to movie/TV items.

**Rate limit**: 1,000 requests/day (free tier).

#### 1. Get an API Key

1. Go to [OMDB API Key](https://www.omdbapi.com/apikey.aspx)
2. Choose the **Free** tier (1,000 daily limit)
3. Enter your email and click **Submit**
4. Check your email for the activation link and click it
5. Your API key will be shown after activation

#### 2. Set Environment Variable

```toml
# In mise.toml
OMDB_API_KEY = "your-omdb-api-key"
```

---

### Podcast Index

Enriches **podcast** items with genre categories and episode metadata from
the open Podcast Index database.

**Rate limit**: No strict published limit, but be reasonable (~10 req/s).

#### 1. Get API Credentials

1. Go to [Podcast Index API](https://api.podcastindex.org/)
2. Click **Get a Free API Key**
3. Fill in your name and email
4. You'll receive an **API Key** and **API Secret** by email

#### 2. Set Environment Variables

```toml
# In mise.toml
PODCAST_INDEX_API_KEY = "your-podcast-index-api-key"
PODCAST_INDEX_API_SECRET = "your-podcast-index-api-secret"
```

> **Note**: Podcast Index API authentication uses both the key and secret to
> generate a request signature. Both values are required.

---

### IGDB (Twitch)

Enriches **game** items with genres, themes, and game metadata from IGDB
(owned by Twitch/Amazon).

**Rate limit**: 4 requests/second.

#### 1. Create a Twitch Developer Application

IGDB uses Twitch OAuth2 client credentials for authentication.

1. Go to [Twitch Developer Console](https://dev.twitch.tv/console/apps)
2. Log in with your Twitch account (or create one at [twitch.tv](https://www.twitch.tv/))
3. Click **Register Your Application**
4. Fill in:
   - **Name**: `Specto`
   - **OAuth Redirect URLs**: `http://localhost:3000` (required but not used — IGDB uses client credentials flow)
   - **Category**: **Application Integration**
5. Click **Create**

#### 2. Get Client ID and Secret

1. On the application list, click **Manage** next to your app
2. Copy the **Client ID**
3. Click **New Secret** to generate a client secret
4. Copy the secret immediately

#### 3. Set Environment Variables

```toml
# In mise.toml
IGDB_CLIENT_ID = "your-twitch-client-id"
IGDB_CLIENT_SECRET = "your-twitch-client-secret"
```

> **Note**: IGDB uses the Twitch client credentials OAuth2 flow. The app
> exchanges the client ID and secret for a bearer token automatically —
> no user interaction needed.

---

### AniList

Enriches **anime** and **manga** items with genres, tags, and metadata from
AniList's comprehensive database.

**No API key required.** AniList's GraphQL API is public and unauthenticated
for read-only queries. No setup needed — this provider is always registered.

**Rate limit**: 90 requests/minute.

**Endpoint**: `https://graphql.anilist.co`

---

### MusicBrainz

Supplements Last.fm for **music** enrichment with genre data from the
MusicBrainz open music encyclopedia.

**No API key required.** MusicBrainz is an open database with a public API.
No setup needed — this provider is always registered when Last.fm is active.

**Rate limit**: 1 request/second (strict — requests must include a
`User-Agent` header identifying your app).

**Endpoint**: `https://musicbrainz.org/ws/2/`

> **Note**: MusicBrainz enforces rate limiting strictly. The enrichment
> provider handles this automatically with a 1 req/s throttle.

---

## LLM Enrichment

The LLM enricher is a universal classifier that runs **after** all API
providers. It uses existing tags as context to fill in gaps (mood, topic,
format) that platform APIs don't cover.

### Google Gemini (googlegenai)

#### 1. Get an API Key

1. Go to [Google AI Studio](https://aistudio.google.com/apikey)
2. Sign in with your Google account
3. Click **Create API Key**
4. Select your Google Cloud project (or create one)
5. Copy the generated API key

#### 2. Set Environment Variables

```toml
# In mise.toml
LLM_PROVIDER = "googlegenai"
LLM_MODEL = "gemini-2.0-flash"
LLM_API_KEY = "your-google-ai-api-key"
```

> **Note**: Gemini 2.0 Flash is recommended for the best cost/quality
> trade-off. The free tier allows 15 requests/minute and 1M tokens/day.

---

### Ollama (local)

Run LLM enrichment locally with no API key or cloud dependency.

#### 1. Install Ollama

1. Go to [ollama.com](https://ollama.com/) and install for your platform
2. Start Ollama: `ollama serve`
3. Pull a model: `ollama pull llama3.1:8b` (or any model that supports structured output)

#### 2. Set Environment Variables

```toml
# In mise.toml
LLM_PROVIDER = "ollama"
LLM_MODEL = "llama3.1:8b"
# LLM_API_KEY not needed for Ollama
LLM_OLLAMA_BASE_URL = "http://localhost:11434"
```

> **Note**: Local models are slower and may produce lower-quality tag
> classifications than cloud models. For best results, use a model with
> at least 8B parameters.

---

## Summary: Redirect URIs

| Provider         | Redirect URI                                                | Purpose         |
| ---------------- | ----------------------------------------------------------- | --------------- |
| Google (login)   | `http://localhost:3000/auth/google/callback`                | App login       |
| Google (YouTube) | `http://localhost:3000/api/v1/plugins/youtube-api/callback` | YouTube plugin  |
| GitHub           | `http://localhost:3000/auth/github/callback`                | App login       |
| Spotify          | `http://localhost:3000/api/v1/plugins/spotify-api/callback` | Spotify plugin  |
| Twitch (IGDB)    | `http://localhost:3000` (not used, just required)           | IGDB enrichment |

For production, replace `http://localhost:3000` with your actual domain
and make sure to use `https://`.

---

## Summary: Environment Variables

All variables in `mise.toml` (uncomment and fill in what you need):

```toml
# App OAuth (user login)
GOOGLE_CLIENT_ID = ""
GOOGLE_CLIENT_SECRET = ""
GITHUB_CLIENT_ID = ""
GITHUB_CLIENT_SECRET = ""

# Plugin OAuth (platform connections)
SPOTIFY_CLIENT_ID = ""
SPOTIFY_CLIENT_SECRET = ""
YOUTUBE_CLIENT_ID = ""                    # same as GOOGLE_CLIENT_ID
YOUTUBE_CLIENT_SECRET = ""                # same as GOOGLE_CLIENT_SECRET

# Enrichment API providers
LASTFM_API_KEY = ""
TMDB_API_KEY = ""
OMDB_API_KEY = ""                         # optional, supplements TMDB
PODCAST_INDEX_API_KEY = ""
PODCAST_INDEX_API_SECRET = ""
IGDB_CLIENT_ID = ""                       # Twitch client ID
IGDB_CLIENT_SECRET = ""                   # Twitch client secret
# AniList — no key needed (public API)
# MusicBrainz — no key needed (public API)

# LLM enrichment
LLM_PROVIDER = ""                         # googlegenai | ollama
LLM_MODEL = ""                            # e.g. gemini-2.0-flash, llama3.1:8b
LLM_API_KEY = ""                          # not needed for ollama
# LLM_OLLAMA_BASE_URL = "http://localhost:11434"

# Enrichment worker tuning
ENRICHMENT_BATCH_SIZE = "50"
ENRICHMENT_POLL_INTERVAL = "5s"
ENRICHMENT_MAX_RETRIES = "3"
ENRICHMENT_MIN_CONFIDENCE = "0.7"
```

Each provider is independently optional. If a variable is empty or unset,
the corresponding provider is simply not registered and that enrichment
source is skipped.
