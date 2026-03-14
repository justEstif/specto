---
# project-media-consumption-analysis-zmzq
title: Navbar layout broken — profile avatar wraps to second line
status: completed
type: bug
priority: high
created_at: 2026-03-13T18:31:12Z
updated_at: 2026-03-13T18:35:13Z
---

## Problem

The navbar has a layout issue where the profile avatar (circle with user initial) wraps to a second row, appearing below the nav links (Timeline, Plugins) instead of inline with them.

## Screenshot

See: /home/estifanos/Pictures/Screenshots/Screenshot from 2026-03-13 14-30-38.png

- "Specto" logo on the left
- "Timeline" and "Plugins" links on the right
- Profile avatar ("E" circle) drops to a new line below the nav links

## Expected

All navbar items should be on a single line: logo left, nav links + profile avatar right, vertically centered.

## To Investigate

- [ ] Check navbar templ component for flex/layout classes
- [ ] Ensure the container uses flex-wrap: nowrap or items are properly sized
- [ ] Verify responsive breakpoints aren't causing premature wrapping
- [ ] Test fix across different viewport widths

## Summary of Changes

Added `flex items-center` to the right-side navbar container (`flex-none` div) in `components/navbar.templ`. The div had `flex-none gap-1` which set flex-shrink behavior but didn't make it a flex container, causing the avatar to wrap to a new line. Now the nav links and avatar are properly inline.
