---
# project-media-consumption-analysis-nus6
title: Wire core layer into application bootstrap
status: todo
type: task
created_at: 2026-03-12T20:21:26Z
updated_at: 2026-03-12T20:21:26Z
parent: project-media-consumption-analysis-86vz
blocked_by:
    - project-media-consumption-analysis-w8az
---

Update cmd/web/main.go to initialize the core layer components: create PluginRegistry, initialize store layer (with encryption key from env), create NoOpEnricher, wire SyncService with its dependencies. Refactor existing handlers to go through the store layer instead of calling database.DB directly. This is the integration task that connects all the pieces. Ensure the app still starts and passes existing tests after wiring.
