---
# project-media-consumption-analysis-igzs
title: UX improvements for plugins page
status: completed
type: feature
priority: normal
created_at: 2026-03-15T01:42:00Z
updated_at: 2026-03-15T01:44:52Z
---

## UX Research Findings

### Issues Identified

#### 1. One plugin per row wastes space (High Priority)
The current `space-y-4` vertical stack puts each plugin in a full-width card. With only 4-6 plugins, the page feels sparse and requires unnecessary scrolling. Users cannot compare plugins at a glance.

**Recommendation:** Use a responsive grid — 2 columns on `sm+`, 3 on `lg+`. Plugin cards become compact, scannable tiles. The grid layout lets users see all options simultaneously, reducing cognitive load and scroll distance.

#### 2. File input contrast is poor (High Priority)
`file-input` with `bg-base-100 border-base-300` on a `bg-neutral` card creates a near-invisible boundary. The file input blends into the card background, making it hard to identify as an interactive element.

**Recommendation:** Use `bg-base-200` or add a stronger border like `border-base-content/20`. Also consider a styled drop-zone instead of the native file input for better affordance.

#### 3. Information hierarchy is flat (Medium Priority)
Every plugin card looks identical — same size, same visual weight. Connected plugins with errors, plugins needing a sync, and idle plugins all look the same. Users cannot quickly identify what needs attention.

**Recommendation:** Use visual differentiation: error states get a colored left border or tinted background. Show item counts prominently for connected plugins. Make the status dot larger or use a badge.

#### 4. Empty state for Connected section is generic (Medium Priority)
"No plugins connected yet. Connect one below to start tracking." is functional but misses an opportunity to guide users.

**Recommendation:** Show a visual preview or highlight the most popular plugin to connect. Use a more actionable empty state.

#### 5. FileImportCard export guides are hidden (Low Priority)
The collapse/accordion pattern for export guides is fine, but users may not discover them. The guides are the most useful content for file-import users.

**Recommendation:** Consider showing the guide for the selected platform inline when a platform is chosen from the dropdown, rather than requiring separate accordion interaction.

## Implementation Plan

- [x] Convert plugin listings from vertical stack to responsive grid
- [x] Fix file input contrast — use stronger border/background
- [x] Show export guide inline when platform is selected in FileImportCard
- [x] Improve connected plugin cards with item count prominence
- [x] Refine empty state copy and visual treatment

## Summary of Changes

All five improvements implemented:
1. Plugin cards now use a responsive grid (1/2/3 columns) instead of single-column stack
2. File inputs and selects use `bg-base-300 border-base-content/20` for visible contrast on neutral cards
3. Export guides show inline when a platform is selected (replaced accordion pattern)
4. Connected plugins show item count prominently in primary color; error cards get a tinted border
5. Empty state uses dashed border and two-line copy with clearer guidance
6. Plugin logos get a subtle background container for better visual anchoring
7. Card actions separated with a border-top for clearer hierarchy
