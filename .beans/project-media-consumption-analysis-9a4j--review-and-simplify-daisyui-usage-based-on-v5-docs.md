---
# project-media-consumption-analysis-9a4j
title: Review and simplify DaisyUI usage based on v5 docs
status: completed
type: task
priority: normal
created_at: 2026-03-13T20:17:17Z
updated_at: 2026-03-13T20:20:59Z
---

Audit templ components for custom Tailwind that could be replaced with DaisyUI 5 component classes, and identify DaisyUI v4 class names that don't exist in v5.


## Todo
- [x] Remove `-bordered` suffixes from input/select/file-input
- [x] Replace `form-control` and `label-text` with v5 equivalents
- [x] Replace custom progress bars with `<progress class="progress">`
- [x] Use `navbar-start`/`navbar-end` in navbar
- [x] Use `list`/`list-row` for tags list (timeline rows too complex for list-row auto-grow)
- [x] Wrap stat cards in `stats` container — skipped: current responsive grid (1→2→4 cols) is better than stats horizontal/vertical
- [x] Regenerate templ files


## Summary of Changes
- Removed `input-bordered` from 5 files (login, register, settings, timeline, plugins) — not a valid DaisyUI 5 class
- Removed `select-bordered` from 3 files (timeline, settings, plugins) — not a valid DaisyUI 5 class
- Removed `file-input-bordered` from plugins.templ — not a valid DaisyUI 5 class
- Replaced `form-control` + `label-text` with `fieldset` + `label` in plugins.templ (v5 equivalents)
- Replaced 3 hand-built progress bars (dashboard + share_profile) with `<progress class="progress progress-*">`
- Replaced `flex-1`/`flex-none` with `navbar-start`/`navbar-end` in navbar.templ
- Replaced custom `divide-y` tag list with `list`/`list-row` component in dashboard tags
- Build verified passing
