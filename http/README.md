# HTTP Request Files

Executable API documentation using [httpyac](https://httpyac.github.io/).

## Usage

```bash
# Run all requests against local server
mise run api-test

# Run a specific file
httpyac send http/health.http -e local

# Run a specific request by name
httpyac send http/health.http --name "Health check" -e local
```

## Environments

- `local` — `http://localhost:3000` (default dev server)

## Prerequisites

- Running dev server: `mise run dev`
- Running PostgreSQL: `docker compose up -d`
- Migrations applied: `mise run db-migrate`

## File organization

| File                  | Description                                      |
| --------------------- | ------------------------------------------------ |
| `00-session.http`     | Unauthenticated session check                    |
| `health.http`         | Health endpoint                                  |
| `auth.http`           | Registration, login, logout flows                |
| `plugins.http`        | Plugin listing, detail, auth-type validation     |
| `spotify-import.http` | Spotify GDPR import lifecycle (import -> verify) |
| `youtube-import.http` | YouTube Takeout import lifecycle                 |
| `import-errors.http`  | Malformed files, missing fields, 404s            |
| `insights.http`       | Insights endpoints (summary, breakdown, tags)    |
| `unauth.http`         | All endpoints return 401 without session         |

Test fixtures are in `fixtures/`. See `docs/development-workflow.md` for conventions.
