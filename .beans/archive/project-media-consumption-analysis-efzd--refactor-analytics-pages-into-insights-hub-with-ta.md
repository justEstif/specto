---
# project-media-consumption-analysis-efzd
title: Refactor analytics pages into Insights hub with tab navigation
status: completed
type: task
priority: normal
created_at: 2026-03-14T23:12:34Z
updated_at: 2026-03-14T23:16:09Z
---

Consolidate Attention, Taste DNA, and Obsessions under a single /insights route with sub-tabs. Reduces navbar to 3 items (Timeline, Insights, Plugins).\n\n## Plan\n- [x] Create insights hub templ component with tab navigation\n- [x] Update routes: /insights, /insights/attention, /insights/taste-dna, /insights/obsessions\n- [x] Create unified insights handler with tab routing\n- [x] Update navbar to single Insights link\n- [x] Remove old standalone page routes\n- [x] Update HTMX partial targets\n- [x] Regenerate templ and verify build+tests

## Summary of Changes

Consolidated Attention, Taste DNA, and Obsessions into a single Insights hub:
- Navbar reduced from 5 links to 3: Timeline, Insights, Plugins
- /insights defaults to Attention tab, with Taste DNA and Obsessions as sub-tabs
- Shared filter bar (platform/type/range) rendered once per tab, targeting /partials/insights/{tab}
- HTMX tab switching with URL push (/insights, /insights/taste-dna, /insights/obsessions)
- Legacy /attention route 301-redirects to /insights
- Removed standalone page handlers, filter bars, and Layout wrappers from each sub-page
- All tests pass, go vet clean
