---
# project-media-consumption-analysis-vbep
title: Clean up timeline overview design
status: completed
type: task
priority: normal
created_at: 2026-03-14T23:46:34Z
updated_at: 2026-03-14T23:48:26Z
---

Redesign the timeline overview scorecard to match Specto's dark cinematic aesthetic. Apply UX research insights and frontend design skill guidelines.

## Summary of Changes

Redesigned the timeline overview scorecard from generic AI-slop stat cards to an editorial, narrative-driven layout:

### What changed
- **Removed** the 4-column identical stat card grid (the classic AI 'big number + small label' pattern)
- **Replaced** with a single lead stat using display typography — item count + platform count read as a natural phrase ('1,247 items across 3 platforms')
- **Replaced** flat badge lists with proportional horizontal bars for genres — visual weight communicates dominance at a glance
- **Moods and topics** now use a compact slash-separated text format instead of badges — less visual noise, secondary emphasis
- **Trending section** uses a left border accent (border-primary) instead of floating badges — reads like an editorial callout
- **Empty state** uses display typography instead of a boxed message — feels intentional, not error-like
- **Removed** the redundant subtitle paragraph that restated what 'Overview' already implies
- **Added** staggered entrance animations (animate-fade-in-up with delays) for cinematic reveal
- All changes follow the existing design system: oklch colors, DaisyUI semantic classes, Outfit display font, warm dark palette
