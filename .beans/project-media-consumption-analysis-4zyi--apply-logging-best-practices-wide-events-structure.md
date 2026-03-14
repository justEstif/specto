---
# project-media-consumption-analysis-4zyi
title: Apply logging best practices - wide events & structured logging
status: completed
type: task
priority: normal
created_at: 2026-03-14T18:56:43Z
updated_at: 2026-03-14T19:01:14Z
---

Implement wide event logging pattern across Specto codebase. Replace scattered log.Printf with centralized slog-based structured logging. Add request-scoped wide event middleware, business context, and environment info.

## Plan

- [x] Create centralized logger package (internal/logger) with JSON slog handler
- [x] Create wide event middleware replacing chi's middleware.Logger
- [x] Add logger to Handler struct and wire through app.New
- [x] Replace all log.Printf in handlers with wide event context additions
- [x] Pass configured logger to core services instead of nil
- [x] Add environment context (commit hash, version) to every event
- [x] Add business context helpers for handlers

## Summary of Changes

Implemented structured logging with wide event pattern across the Specto codebase:

1. **`internal/logger/logger.go`** - Centralized JSON slog logger with environment context (service name, version, commit hash). Includes wide event builder that accumulates context throughout request lifecycle.

2. **`internal/middleware/logging.go`** - Wide event middleware replacing chi's `middleware.Logger`. Emits a single structured JSON event per request at completion, including method, path, status, duration, request ID, and any business context added by handlers.

3. **`internal/app/app.go`** - Added `Logger` field to `Config` and `App`. Passes configured logger to all core services (SyncService, EnrichmentCoordinator, EnrichmentWorker) instead of nil.

4. **`internal/handlers/handler.go`** - Added `log` field and `addContext()` helper for handlers to enrich the wide event with business context.

5. **`cmd/web/main.go`** - Logger initialized first, wired throughout. Replaced all `log.Fatal`/`log.Printf` with structured slog calls. Replaced `middleware.Logger` with `WideEventLogger`. Added build-time version injection via `-ldflags`.

6. **All handler files** - Replaced scattered `log.Printf()` calls with `addContext()` calls that add error details to the wide event, so errors appear as fields in the single per-request log line.
