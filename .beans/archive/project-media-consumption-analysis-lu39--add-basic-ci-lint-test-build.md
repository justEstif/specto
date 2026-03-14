---
# project-media-consumption-analysis-lu39
title: Add basic CI (lint, test, build)
status: completed
type: task
priority: low
created_at: 2026-03-12T02:37:25Z
updated_at: 2026-03-14T19:35:02Z
parent: project-media-consumption-analysis-hj33
blocked_by:
    - project-media-consumption-analysis-w1nk
---

GitHub Actions workflow: go vet, staticcheck/golangci-lint, go test, go build. Run on push/PR.

## Summary of Changes

- Added GitHub Actions CI workflow (`.github/workflows/ci.yml`) that runs on push to main and PRs: sets up Postgres, runs migrations + code gen via mise, then go vet, go test, and build
- Added `pre-commit` mise task that runs templ generate, go vet, go test, and a build check
- Generated git pre-commit hook via `mise generate git-pre-commit` that triggers the pre-commit task on every commit
