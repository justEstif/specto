# OAuth Credentials Setup Guide

This guide walks you through creating OAuth credentials for each provider
Specto uses. You only need to set up the providers you want to use — all
are optional.

Once you have the credentials, uncomment and fill in the values in `mise.toml`.

---

## Table of Contents

- [Google (App Login + YouTube Plugin)](#google-app-login--youtube-plugin)
- [GitHub (App Login)](#github-app-login)
- [Spotify (Plugin Connection)](#spotify-plugin-connection)
- [Summary: Redirect URIs](#summary-redirect-uris)
- [Summary: Environment Variables](#summary-environment-variables)

---

## Google (App Login + YouTube Plugin)

Google Cloud Console credentials are shared between Google OAuth login and
the YouTube API plugin. You create **one** OAuth client in Google Cloud and
use the same Client ID/Secret for both `GOOGLE_*` and `YOUTUBE_*` env vars.

### 1. Create a Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Click the project dropdown at the top → **New Project**
3. Name it something like `specto-dev`, click **Create**
4. Select the newly created project from the dropdown

### 2. Enable Required APIs

1. Go to **APIs & Services → Library** (or [direct link](https://console.cloud.google.com/apis/library))
2. Search for and enable each:
   - **YouTube Data API v3** (for the YouTube plugin)
   - No additional API needed for Google Sign-In — it uses the default OAuth2 userinfo endpoint

### 3. Configure the OAuth Consent Screen

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

### 4. Create OAuth Client Credentials

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

### 5. Set Environment Variables

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

## GitHub (App Login)

### 1. Create a GitHub OAuth App

1. Go to [GitHub → Settings → Developer settings → OAuth Apps](https://github.com/settings/developers)
   - Or: click your profile picture → **Settings** → scroll down to **Developer settings** → **OAuth Apps**
2. Click **New OAuth App** (or **Register a new application** if it's your first)
3. Fill in:
   - **Application name**: `Specto Local Dev`
   - **Homepage URL**: `http://localhost:3000`
   - **Authorization callback URL**: `http://localhost:3000/auth/github/callback`
4. Click **Register application**

### 2. Get Client ID and Secret

1. After registration, you'll see the **Client ID** on the app page
2. Click **Generate a new client secret**
3. Copy the secret immediately (it won't be shown again)

### 3. Set Environment Variables

```toml
# In mise.toml
GITHUB_CLIENT_ID = "your-github-client-id"
GITHUB_CLIENT_SECRET = "your-github-client-secret"
```

---

## Spotify (Plugin Connection)

### 1. Create a Spotify App

1. Go to [Spotify Developer Dashboard](https://developer.spotify.com/dashboard)
2. Log in with your Spotify account (or create one)
3. Click **Create App**
4. Fill in:
   - **App name**: `Specto`
   - **App description**: `Media consumption tracker`
   - **Redirect URI**: `http://127.0.0.1:3000/api/v1/plugins/spotify-api/callback`
   - Check the **Developer Terms of Service** checkbox
5. Click **Create**

### 2. Get Client ID and Secret

1. On the app overview page, you'll see the **Client ID**
2. Click **Show Client Secret** to reveal the secret
3. Copy both values

### 3. Configure Redirect URI (if not set during creation)

1. On the app overview page, click **Edit Settings**
2. Under **Redirect URIs**, add:
   ```
   http://localhost:3000/api/v1/plugins/spotify-api/callback
   ```
3. Click **Save**

### 4. Set Environment Variables

```toml
# In mise.toml
SPOTIFY_CLIENT_ID = "your-spotify-client-id"
SPOTIFY_CLIENT_SECRET = "your-spotify-client-secret"
```

> **Note**: Spotify apps start in **Development Mode**, limited to 25
> users. This is fine for local development and testing. If you need more
> users, request a quota extension from the dashboard.

---

## Summary: Redirect URIs

| Provider         | Redirect URI                                                | Purpose        |
| ---------------- | ----------------------------------------------------------- | -------------- |
| Google (login)   | `http://localhost:3000/auth/google/callback`                | App login      |
| Google (YouTube) | `http://localhost:3000/api/v1/plugins/youtube-api/callback` | YouTube plugin |
| GitHub           | `http://localhost:3000/auth/github/callback`                | App login      |
| Spotify          | `http://localhost:3000/api/v1/plugins/spotify-api/callback` | Spotify plugin |

For production, replace `http://localhost:3000` with your actual domain
and make sure to use `https://`.

---

## Summary: Environment Variables

All variables in `mise.toml` (uncomment and fill in):

```toml
# App OAuth (user login)
GOOGLE_CLIENT_ID = ""
GOOGLE_CLIENT_SECRET = ""
GITHUB_CLIENT_ID = ""
GITHUB_CLIENT_SECRET = ""

# Plugin OAuth (platform connections)
SPOTIFY_CLIENT_ID = ""
SPOTIFY_CLIENT_SECRET = ""
YOUTUBE_CLIENT_ID = ""          # same as GOOGLE_CLIENT_ID
YOUTUBE_CLIENT_SECRET = ""      # same as GOOGLE_CLIENT_SECRET
```

Each provider is independently optional. If a variable pair is empty or
unset, the corresponding login button or plugin connect action will
gracefully fail with a "provider not configured" message.
