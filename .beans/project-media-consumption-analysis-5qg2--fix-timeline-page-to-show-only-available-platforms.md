---
# project-media-consumption-analysis-5qg2
title: Fix timeline page to show only available platforms
status: completed
type: bug
priority: normal
created_at: 2026-03-13T21:09:36Z
updated_at: 2026-03-13T21:10:57Z
---

The timeline page hardcodes platform filter options including Netflix. Should dynamically populate from registered plugins.

## Summary of Changes

- Added `Platforms()` method to `PluginRegistry` that returns unique platform names derived from registered plugins (strips `-api` suffix for deduplication)
- Added `Platforms []string` field to `TimelinePageData` struct
- Changed `timelineFilters` templ to dynamically render platform options from the passed list instead of hardcoding them
- Added `platformLabel()` helper for human-readable platform names
- Updated `TimelinePage` handler to pass `h.App.Registry.Platforms()` to the template
