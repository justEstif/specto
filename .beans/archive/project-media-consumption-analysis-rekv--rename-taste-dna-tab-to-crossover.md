---
# project-media-consumption-analysis-rekv
title: Rename Taste DNA tab to Crossover
status: completed
type: task
priority: normal
created_at: 2026-03-14T23:18:10Z
updated_at: 2026-03-14T23:19:58Z
---

Rename taste-dna to crossover across routes, components, and handlers.

## Summary of Changes\n\nRenamed user-facing "Taste DNA" to "Crossover":\n- Tab label: Taste DNA → Crossover\n- URL: /insights/taste-dna → /insights/crossover\n- API endpoint: /api/v1/insights/taste-dna → /api/v1/insights/crossover\n- Templ file: taste_dna.templ → crossover.templ, all component names updated\n- Internal Go types (TasteDNAEntry, CrossPlatformTasteDNA) kept as-is since they describe the backend behavior accurately
