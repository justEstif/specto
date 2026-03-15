# Production Deployment with Dokku

Deploy Specto on a local server using Dokku and expose it via Cloudflare Tunnel — no open ports required.

This guide assumes you already have Dokku and Cloudflare Tunnel configured.
See the [Dokku + Cloudflare Tunnel Setup Guide](https://github.com/justEstif/dokku-cloudflare-setup)
for initial server setup if needed.

---

## 1. Create the App

On the Dokku server:

```bash
dokku apps:create specto
dokku domains:set specto specto.estifanos.cc
```

## 2. Set Up PostgreSQL

```bash
# Install the postgres plugin (if not already installed)
sudo dokku plugin:install https://github.com/dokku/dokku-postgres.git

# Create the database
dokku postgres:create specto-db

# Link it to the app (this sets DATABASE_URL automatically)
dokku postgres:link specto-db specto

# Verify the database was created and linked
dokku postgres:info specto-db
dokku config:get specto DATABASE_URL
```

The `postgres:create` command spins up a PostgreSQL 16 container with a
randomly generated user and password. The `postgres:link` command injects
`DATABASE_URL` into the app's environment — no manual connection string needed.

## 3. Configure Environment Variables

```bash
# Required
dokku config:set specto \
  ENCRYPTION_KEY="$(openssl rand -hex 32)" \
  SESSION_SECRET="$(openssl rand -base64 32)" \
  CSRF_KEY="$(openssl rand -base64 32 | head -c 32)" \
  PORT=5000 \
  BASE_URL=https://specto.estifanos.cc

# Optional — OAuth providers (leave unset to skip)
dokku config:set specto \
  SPOTIFY_CLIENT_ID=your-id \
  SPOTIFY_CLIENT_SECRET=your-secret \
  GOOGLE_CLIENT_ID=your-id \
  GOOGLE_CLIENT_SECRET=your-secret \
  GITHUB_CLIENT_ID=your-id \
  GITHUB_CLIENT_SECRET=your-secret

# Optional — same Google credentials for YouTube plugin
dokku config:set specto \
  YOUTUBE_CLIENT_ID=your-google-id \
  YOUTUBE_CLIENT_SECRET=your-google-secret

# Optional — enrichment APIs
dokku config:set specto \
  LASTFM_API_KEY=your-key \
  TMDB_API_KEY=your-key \
  IGDB_CLIENT_ID=your-twitch-id \
  IGDB_CLIENT_SECRET=your-twitch-secret

# Optional — LLM enrichment
dokku config:set specto \
  LLM_PROVIDER=googlegenai \
  LLM_MODEL=gemini-2.5-flash \
  LLM_API_KEY=your-key
```

## 4. Deploy from Dev Machine

```bash
cd specto
git remote add dokku dokku@192.168.0.29:specto
git push dokku main
```

Dokku will detect the `Dockerfile` and build automatically.

## 5. Run Migrations

```bash
# SSH into the server and run migrations inside the container
ssh dokku-server
dokku run specto /app/specto migrate -path /app/migrations -database "\$DATABASE_URL" up
```

Or install the migrate CLI on the server and run against the linked database:

```bash
DATABASE_URL=$(dokku config:get specto DATABASE_URL)
migrate -path migrations -database "$DATABASE_URL" up
```

## 6. Verify

Visit `https://specto.estifanos.cc` — you should see the login page.

---

## Update OAuth Redirect URIs

After deploying to production, you **must** update the redirect URIs for
each OAuth provider to use your production domain. Replace all
`http://localhost:3000` URIs with `https://specto.estifanos.cc`.

| Provider | Setting location | Redirect URI |
|---|---|---|
| Google (login + YouTube) | [Google Cloud Console](https://console.cloud.google.com/apis/credentials) | `https://specto.estifanos.cc/auth/google/callback` |
| GitHub | [GitHub Developer Settings](https://github.com/settings/developers) | `https://specto.estifanos.cc/auth/github/callback` |
| Spotify | [Spotify Developer Dashboard](https://developer.spotify.com/dashboard) | `https://specto.estifanos.cc/auth/spotify/callback` |

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
