---
# project-media-consumption-analysis-yp0q
title: Update UI design wireframes with dashboard filters
status: completed
type: task
priority: normal
created_at: 2026-03-13T21:02:01Z
updated_at: 2026-03-13T21:03:33Z
---

Update the ASCII wireframe for the Dashboard section in docs/ui-design.md to include a filter bar (platform, type, date range). The timeline page wireframe already shows filters — the dashboard should get a similar treatment. Add a filter bar between the heading and the stats row, with dropdowns for platform and media type. The existing activity chart range tabs (7d/30d/90d) should become the global date range filter that affects all sections. Document the HTMX interaction pattern (partial swap of dashboard content on filter change).

## Summary of Changes

Updated the Dashboard wireframe in docs/ui-design.md:
- Added a filter bar (platform dropdown, type dropdown, date range tabs) between heading and stats
- Moved the activity chart range tabs into the global filter bar so all sections respond to the same range
- Added a Filters section documenting the controls and their values
- Updated HTMX interactions to describe the partial swap pattern for filter changes (single swap of #dashboard-content)
- Updated responsive notes: filters stack vertically on mobile
