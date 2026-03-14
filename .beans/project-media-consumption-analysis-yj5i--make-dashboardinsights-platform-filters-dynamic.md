---
# project-media-consumption-analysis-yj5i
title: Make dashboard/insights platform filters dynamic
status: completed
type: bug
priority: normal
created_at: 2026-03-14T23:45:42Z
updated_at: 2026-03-14T23:48:07Z
---

Platform filter dropdowns are hardcoded in dashboard.templ and insights.templ. Should be dynamic based on registry or user data.

## Summary of Changes

Replaced hardcoded platform filter dropdowns in dashboard.templ and insights.templ with dynamic lists populated from the plugin registry via `Registry.Platforms()`.

**Files changed:**
- `components/dashboard.templ` — Added `Platforms []string` to `DashboardData`, made `dashboardFilterBar` accept and loop over platforms
- `components/insights.templ` — Added `Platforms []string` to `InsightsPageData`, made `insightsFilterBar` accept and loop over platforms
- `internal/handlers/home.go` — Populate `Platforms` from `h.App.Registry.Platforms()`
- `internal/handlers/insights_page.go` — Populate `Platforms` from `h.App.Registry.Platforms()`
