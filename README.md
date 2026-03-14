# Specto

Personal media consumption analysis dashboard. Aggregates your digital media history across platforms (Spotify, YouTube, Netflix, etc.), normalizes it into a unified schema, and surfaces insights into what you consume, how much, and what patterns dominate your attention.

See [docs/MVP.md](docs/MVP.md) for the full product scope.

## Tech Stack

- **[Go](https://go.dev/)** + **[Chi](https://github.com/go-chi/chi)** — HTTP router
- **[Templ](https://templ.guide/)** — Type-safe Go templating
- **[HTMX](https://htmx.org/)** — Dynamic interactions
- **[DaisyUI](https://daisyui.com/)** + **[Tailwind CSS](https://tailwindcss.com/)** — UI components & utility CSS
- **[PostgreSQL](https://www.postgresql.org/)** — Database
- **[sqlc](https://sqlc.dev/)** — Compile-time type-safe SQL
- **[golang-migrate](https://github.com/golang-migrate/migrate)** — Database migrations
- **[httpyac](https://httpyac.github.io/)** — API documentation & testing
- **[mise](https://mise.jdx.dev/)** — Tool & task management

## Prerequisites

- [mise](https://mise.jdx.dev/) installed
- [Docker](https://www.docker.com/) for PostgreSQL

## Quick Start

### 1. Install Tools

```bash
mise install
```

### 2. Start PostgreSQL

```bash
docker-compose up -d
```

### 3. Setup Project

```bash
mise run setup
```

### 4. Start Development

```bash
# Terminal 1 — watch templ files (optional)
mise run templ

# Terminal 2 — dev server with live reload
mise run dev
```

### 5. Visit Application

Open [http://localhost:3000](http://localhost:3000)

## Available Tasks

Run `mise tasks` to see all available tasks:

| Task | Description |
|------|-------------|
| `mise run dev` | Start development server with live reload |
| `mise run templ` | Generate templ files |
| `mise run db-migrate` | Run database migrations |
| `mise run db-rollback` | Rollback last migration |
| `mise run sqlc` | Generate type-safe SQL code |
| `mise run setup` | Complete project setup |
| `mise run build` | Build production binary |
| `mise run api-test` | Run httpyac API tests |

## Project Structure

```
.
├── cmd/web/              # Application entry point
├── internal/
│   ├── handlers/         # HTTP handlers
│   ├── middleware/        # Custom middleware
│   └── database/         # Database connection & queries
├── components/           # Templ templates
├── migrations/           # Database migrations
├── http/                 # httpyac API request files
│   └── environments/     # Environment configs
├── docs/                 # Project documentation
├── mise.toml             # Tool & task configuration
└── docker-compose.yml    # PostgreSQL setup
```

## Documentation

- [MVP](docs/MVP.md) — Product scope and features
- [Architecture](docs/architecture.md) — System design and layers
- [API](docs/api.md) — HTTP API contract
- [Schema](docs/schema.md) — Database schema
- [Self-Hosting](docs/self-hosting.md) — Deploy your own instance
- [Development Workflow](docs/development-workflow.md) — How to develop
- [Plugin Guide](docs/plugin-guide.md) — Plugin interface

## License

MIT
