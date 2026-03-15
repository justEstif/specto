---
# project-media-consumption-analysis-cknh
title: Fix eras timeline grid overflow with many eras
status: completed
type: bug
priority: normal
created_at: 2026-03-15T01:59:07Z
updated_at: 2026-03-15T02:08:16Z
---

The eras lane uses flex-1 with no minimum width, so segments get crushed when there are many eras. Make the lane horizontally scrollable with a min-width per segment.

## Summary of Changes\n\n- Made era lane container horizontally scrollable with `overflow-x-auto`\n- Added `min-w-48` (12rem) to each era segment so they don't collapse\n- Added `erasLaneMinWidth()` helper that sets a min-width on the flex container when there are more than 4 eras, triggering horizontal scroll\n- For 4 or fewer eras, layout remains unchanged (segments flex to fill)

\n- Refactored to use DaisyUI `carousel carousel-start` instead of manual overflow-x-auto + min-width calculation\n- Each era segment is now a `carousel-item` with `flex-1 min-w-48`\n- Removed `erasLaneMinWidth()` helper (no longer needed)

\n\n## Next: Add carousel nav buttons\n- [x] Add prev/next buttons flanking the carousel\n- [x] Add scrollEras templ script for smooth horizontal scrolling\n- [x] Only show buttons when eras overflow (buttons are always present, no-op at edges)

\n\n## Final approach: responsive grid\nReplaced carousel + nav buttons with a simple responsive CSS grid:\n- 1 col on mobile, 2 on sm, 3 on lg, 4 on xl\n- Removed the horizontal spine line (doesn't work with wrapping grid)\n- Removed flex-1/min-w-0 from era segments (grid handles sizing)\n- No JS needed
