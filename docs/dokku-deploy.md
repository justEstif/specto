# Production Deployment with Dokku

Deploy Specto on a local server using Dokku and expose it via Cloudflare Tunnel — no open ports required.

This guide assumes you already have Dokku and Cloudflare Tunnel configured.
See the [Dokku + Cloudflare Tunnel Setup Guide](https://github.com/justEstif/dokku-cloudflare-setup)
for initial server setup if needed.

---

## How It Works

Dokku uses the **Heroku Go buildpack** to build and deploy:

- `bin/go-pre-compile` — installs templ + sqlc and runs code generation before the Go build
- `bin/go-post-compile` — installs the migrate CLI into the release slug
- `Procfile` — defines the web process and runs migrations automatically on each deploy

No Dockerfile needed. Dokku detects `go.mod` and uses the Go buildpack.

---

## 1. Automated Setup

The `bin/dokku-setup` script automates app creation, database setup, and environment configuration from your dev machine:

```bash
./bin/dokku-setup
```

This runs over SSH and handles everything: creates the app, sets the domain, creates and links PostgreSQL, and generates required secrets (encryption key, session secret, CSRF key).

**Custom Dokku host:**

```bash
DOKKU_HOST=your-server-ip ./bin/dokku-setup
```

**Optional environment variables:** Create a `bin/.env.production` file with `KEY=value` pairs (one per line) for OAuth credentials, API keys, and other optional config. The script loads this file automatically:

```
SPOTIFY_CLIENT_ID=your-id
SPOTIFY_CLIENT_SECRET=your-secret
GOOGLE_CLIENT_ID=your-id
GOOGLE_CLIENT_SECRET=your-secret
GITHUB_CLIENT_ID=your-id
GITHUB_CLIENT_SECRET=your-secret
YOUTUBE_CLIENT_ID=your-google-id
YOUTUBE_CLIENT_SECRET=your-google-secret
LASTFM_API_KEY=your-key
TMDB_API_KEY=your-key
IGDB_CLIENT_ID=your-twitch-id
IGDB_CLIENT_SECRET=your-twitch-secret
LLM_PROVIDER=googlegenai
LLM_MODEL=gemini-2.5-flash
LLM_API_KEY=your-key
```

You can also pass a custom env file path: `./bin/dokku-setup /path/to/.env.production`

> **Note:** The postgres Dokku plugin must already be installed on the server. If not: `sudo dokku plugin:install https://github.com/dokku/dokku-postgres.git`

## 2. Deploy

```bash
cd specto
git remote add dokku dokku@192.168.0.29:specto
git push dokku main
```

Dokku will:

1. Detect `go.mod` and use the Go buildpack
2. Run `bin/go-pre-compile` (templ + sqlc generation)
3. Build `cmd/web/` → `bin/web`
4. Run `bin/go-post-compile` (install migrate CLI)
5. Run the `release` phase from `Procfile` (database migrations)
6. Start the `web` process

Migrations run automatically on every deploy — no manual step needed.

## 3. Verify

Visit `https://specto.estifanos.cc` — you should see the login page.

---

## Update OAuth Redirect URIs

After deploying to production, you **must** update the redirect URIs for
each OAuth provider to use your production domain. Replace all
`http://localhost:3000` URIs with `https://specto.estifanos.cc`.

| Provider                 | Setting location                                                          | Redirect URI                                        |
| ------------------------ | ------------------------------------------------------------------------- | --------------------------------------------------- |
| Google (login + YouTube) | [Google Cloud Console](https://console.cloud.google.com/apis/credentials) | `https://specto.estifanos.cc/auth/google/callback`  |
| GitHub                   | [GitHub Developer Settings](https://github.com/settings/developers)       | `https://specto.estifanos.cc/auth/github/callback`  |
| Spotify                  | [Spotify Developer Dashboard](https://developer.spotify.com/dashboard)    | `https://specto.estifanos.cc/auth/spotify/callback` |

You can add multiple redirect URIs to support both development and production.

---

## Quick Reference

```bash
# Logs
dokku logs specto -t

# Restart
dokku ps:restart specto

# Check config
dokku config:show specto

# Database shell
dokku postgres:connect specto-db

# Database backup
dokku postgres:export specto-db > backup.sql.gz

# Database restore
dokku postgres:import specto-db < backup.sql.gz

# Deploy new changes
git push dokku main
```

---

## Troubleshooting

**App not accessible?**

- Check Dokku logs: `dokku logs specto -t`
- Check cloudflared: `sudo systemctl status cloudflared`
- Verify DNS: `dig +short specto.estifanos.cc`

**OAuth login fails?**

- Confirm redirect URIs match `BASE_URL` exactly (see table above)
- Ensure you're using `https://` in production

**Database connection errors?**

- Verify link: `dokku postgres:info specto-db`
- Check DATABASE_URL: `dokku config:get specto DATABASE_URL`
