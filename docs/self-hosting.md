# Self-Hosting Guide

Deploy Specto on your own server. This guide covers building, configuring,
and running a production instance.

---

## Prerequisites

- **Go 1.25+** (to build from source)
- **PostgreSQL 16+**
- **[mise](https://mise.jdx.dev/)** (for tooling and code generation)

Or use Docker Compose which bundles everything for you.

---

## Deployment Options

### Option 1: Docker Compose (Recommended)

#### 1. Clone the Repository

```bash
git clone https://github.com/justEstif/specto.git
cd specto
```

#### 2. Create Environment File

```bash
cat > .env << 'EOF'
# Required
DATABASE_URL=postgres://specto:changeme@postgres:5432/specto?sslmode=disable
POSTGRES_USER=specto
POSTGRES_PASSWORD=changeme
POSTGRES_DB=specto
ENCRYPTION_KEY=    # openssl rand -hex 32
SESSION_SECRET=    # openssl rand -base64 32
CSRF_KEY=          # openssl rand -base64 32 | head -c 32
PORT=3000
BASE_URL=https://specto.example.com

# Optional — see docs/api-key-setup.md for all credentials
# SPOTIFY_CLIENT_ID=
# SPOTIFY_CLIENT_SECRET=
# GOOGLE_CLIENT_ID=
# GOOGLE_CLIENT_SECRET=
# GITHUB_CLIENT_ID=
# GITHUB_CLIENT_SECRET=
# LASTFM_API_KEY=
# TMDB_API_KEY=
# LLM_PROVIDER=
# LLM_MODEL=
# LLM_API_KEY=
EOF
```

#### 3. Build and Start

```bash
# Build and start all services
docker compose -f compose.prod.yaml up -d --build

# Run database migrations
docker compose -f compose.prod.yaml exec app \
  /app/specto migrate -path /app/migrations -database "$DATABASE_URL" up
```

Or run migrations with the migrate CLI:

```bash
# Install migrate if needed
go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations against the Docker postgres
migrate -path migrations -database "postgres://specto:changeme@localhost:5432/specto?sslmode=disable" up
```

#### 4. Check Logs

```bash
docker compose -f compose.prod.yaml logs -f app
```

---

### Option 2: Binary Deployment

Build and deploy the binary directly on your server.

#### 1. Build the Binary

On your build machine:

```bash
git clone https://github.com/justEstif/specto.git
cd specto

# Install tools and generate code
mise install
mise run setup

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o specto ./cmd/web

# Package with migrations and static assets
tar -czvf specto.tar.gz specto migrations/ static/
```

#### 2. Set Up PostgreSQL

On your server:

```bash
sudo apt update && sudo apt install postgresql postgresql-contrib

sudo -u postgres psql << 'EOF'
CREATE USER specto WITH PASSWORD 'your-secure-password';
CREATE DATABASE specto OWNER specto;
GRANT ALL PRIVILEGES ON DATABASE specto TO specto;
EOF
```

#### 3. Deploy the Binary

```bash
sudo mkdir -p /opt/specto
cd /opt/specto
sudo tar -xzvf /path/to/specto.tar.gz

# Create environment file
sudo tee /opt/specto/.env > /dev/null << 'EOF'
DATABASE_URL=postgres://specto:your-secure-password@localhost:5432/specto?sslmode=disable
ENCRYPTION_KEY=<openssl rand -hex 32>
SESSION_SECRET=<openssl rand -base64 32>
CSRF_KEY=<first 32 chars of openssl rand -base64 32>
PORT=3000
BASE_URL=https://specto.example.com
EOF
sudo chmod 600 /opt/specto/.env
```

#### 4. Run Migrations

```bash
source /opt/specto/.env
migrate -path /opt/specto/migrations -database "$DATABASE_URL" up
```

#### 5. Create Systemd Service

```bash
sudo tee /etc/systemd/system/specto.service > /dev/null << 'EOF'
[Unit]
Description=Specto
After=network.target postgresql.service

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/opt/specto
EnvironmentFile=/opt/specto/.env
ExecStart=/opt/specto/specto
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable specto
sudo systemctl start specto

# Check status
sudo systemctl status specto
sudo journalctl -u specto -f
```

---

## Reverse Proxy Setup

Run behind a reverse proxy for TLS termination in production.

### Caddy

```caddyfile
specto.example.com {
    reverse_proxy localhost:3000
}
```

### Nginx

```nginx
server {
    listen 80;
    server_name specto.example.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name specto.example.com;

    ssl_certificate /etc/letsencrypt/live/specto.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/specto.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

**Important:** Update all OAuth redirect URIs to use your production domain with `https://`.

---

## Environment Variables

### Required

| Variable | Description | Example |
|---|---|---|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://user:pass@host:5432/specto?sslmode=require` |
| `ENCRYPTION_KEY` | 64 hex characters, encrypts stored OAuth tokens | `openssl rand -hex 32` |
| `SESSION_SECRET` | 32+ bytes, signs session cookies | `openssl rand -base64 32` |
| `CSRF_KEY` | Exactly 32 bytes, CSRF protection | `openssl rand -base64 32` (first 32 chars) |
| `PORT` | HTTP listen port | `3000` |
| `BASE_URL` | Public URL of your instance | `https://specto.example.com` |

### Optional

See **[API Key & Credentials Setup Guide](api-key-setup.md)** for step-by-step
instructions on obtaining credentials for each provider. All OAuth providers
and enrichment APIs are optional -- if a credential is unset, that provider
is simply not registered.

---

## Backup and Restore

### Backup Database

```bash
# Plain SQL
pg_dump -U specto -d specto > backup.sql

# Compressed
pg_dump -U specto -d specto | gzip > backup.sql.gz

# Docker Compose
docker compose -f compose.prod.yaml exec postgres \
  pg_dump -U specto -d specto | gzip > backup.sql.gz
```

### Restore Database

```bash
# From SQL file
psql -U specto -d specto < backup.sql

# From compressed
gunzip -c backup.sql.gz | psql -U specto -d specto
```

---

## Monitoring

### Health Check

```bash
# Systemd
sudo journalctl -u specto -f

# Docker
docker compose -f compose.prod.yaml logs -f app
```

### Database Connections

```bash
psql -U specto -d specto -c "SELECT count(*) FROM pg_stat_activity WHERE datname = 'specto';"
```

---

## Troubleshooting

### Database connection refused

1. Ensure PostgreSQL is running
2. Check the `DATABASE_URL` format
3. Verify the user has access to the database

### OAuth login fails

1. Confirm redirect URIs match your `BASE_URL` exactly
2. Ensure you're using `https://` in production
3. Check that the OAuth client ID/secret are correct

### Session/CSRF errors

1. `SESSION_SECRET` must be at least 32 bytes
2. `CSRF_KEY` must be exactly 32 bytes
3. Both must remain consistent across restarts (don't regenerate on each deploy)

### Enrichment not working

1. Check that the relevant API keys are set (LASTFM, TMDB, LLM, etc.)
2. Review logs for enrichment worker errors
3. Verify `LLM_PROVIDER` and `LLM_MODEL` are correct if using LLM enrichment

---

## Security Considerations

1. **Environment variables**: Never commit `.env` files. Use secrets management in production.
2. **Database**: Use a strong password and restrict network access.
3. **HTTPS**: Always use HTTPS in production.
4. **OAuth tokens**: Stored encrypted in the database via `ENCRYPTION_KEY`. Keep this key safe.
5. **Secrets rotation**: Changing `ENCRYPTION_KEY` will invalidate stored OAuth tokens. Changing `SESSION_SECRET` will invalidate active sessions.
