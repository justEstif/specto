---
# project-media-consumption-analysis-dm3m
title: Merge Routines heatmap into Attention page
status: completed
type: task
priority: normal
created_at: 2026-03-14T22:01:19Z
updated_at: 2026-03-14T22:03:13Z
---

Combine the separate /routines page into the /attention page as a new section. Remove /routines route and nav link.

## Summary of Changes\n\n- Merged heatmap into Attention page as "Weekly Rhythm" section\n- Added HeatmapCells field to AttentionData struct\n- Attention handler now fetches heatmap data alongside existing queries\n- Removed /routines page route, partial, and nav link\n- Simplified routines.templ to only heatmap rendering helpers (no page/filter components)\n- Simplified routines.go to only the JSON API handler\n- JSON API remains at /api/v1/insights/heatmap
