---
# project-media-consumption-analysis-76pe
title: Visual polish for Sharing settings tab
status: completed
type: task
priority: normal
created_at: 2026-03-13T21:23:24Z
updated_at: 2026-03-13T21:24:23Z
---

Improve spacing, layout, and typography of the Sharing tab on settings page

## Summary of Changes\n\nVisually polished the Sharing settings tab:\n\n- **Public profile section**: Added descriptive subtitle, better toggle layout, and styled the share URL in a bg-base-200/60 pill with mono font. Missing slug warning now has a status-warning indicator.\n- **Blocks section**: Wrapped in a proper fieldset (was a raw div) for visual consistency. Replaced checkbox toggles with toggle-sm switches. Drag handle changed from '=' to braille dots (⠿). Enabled rows get a subtle bg highlight, disabled rows dim with opacity. Time range labels now read '30 days' instead of '30d'.\n- **Exclusions section**: Switched to responsive 2-column grid layout (label | input) matching the Account tab pattern. Added for/id attributes for accessibility.\n- **Actions bar**: Added items-center alignment for consistent button baseline.
