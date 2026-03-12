---
# project-media-consumption-analysis-nus6
title: Wire core layer into application bootstrap
status: completed
type: task
priority: normal
created_at: 2026-03-12T20:21:26Z
updated_at: 2026-03-12T21:22:12Z
parent: project-media-consumption-analysis-86vz
blocked_by:
    - project-media-consumption-analysis-w8az
---

Update cmd/web/main.go to initialize the core layer components: create PluginRegistry, initialize store layer (with encryption key from env), create NoOpEnricher, wire SyncService with its dependencies. Refactor existing handlers to go through the store layer instead of calling database.DB directly. This is the integration task that connects all the pieces. Ensure the app still starts and passes existing tests after wiring.

\n## Todo\n\n- [x] Survey existing cmd/web/main.go, handlers, auth code\n- [x] Create core app struct that holds all core dependencies\n- [x] Initialize store layer with encryption key from env\n- [x] Create PluginRegistry, NoOpEnricher, SyncService, InsightsService\n- [x] Refactor handlers to use core stores instead of database.DB\n- [x] Update main.go to wire everything together\n- [x] Ensure app compiles and tests pass\n- [x] Run go vet


## Summary of Changes

Wired the core layer into the application bootstrap:

- **`internal/app/app.go`**: New `App` struct holding all core dependencies (PluginRegistry, SyncService, InsightsService, all 5 store interfaces). `New()` reads `ENCRYPTION_KEY` from env and initializes the full dependency graph.
- **`internal/handlers/handler.go`**: New `Handler` struct with dependency injection via `New(db)`. All handlers converted from package-level functions to methods.
- **`internal/handlers/auth.go`**: Refactored `Register`, `Login`, `Logout`, `Session` to methods on `*Handler`, using `h.DB` instead of `database.DB` global.
- **`internal/handlers/health.go`**: Refactored `Health` to method.
- **`internal/handlers/home.go`**: Refactored `Home` to method.
- **`internal/middleware/auth.go`**: `RequireAuth` now takes `*database.Queries` as parameter and returns middleware closure, eliminating global DB dependency.
- **`cmd/web/main.go`**: Initializes `app.New()`, creates `handlers.New()`, passes DB to `RequireAuth()` middleware. Added `defer database.Close()` and plugin list logging.

No global mutable state remains in handlers or middleware — all dependencies are injected.
