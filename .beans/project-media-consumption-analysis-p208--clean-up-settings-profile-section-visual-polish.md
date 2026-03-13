---
# project-media-consumption-analysis-p208
title: Clean up settings profile section - visual polish
status: completed
type: task
priority: normal
created_at: 2026-03-13T21:20:54Z
updated_at: 2026-03-13T21:21:59Z
---

Improve spacing, layout, and typography of the Profile fieldset on the Account settings tab

## Summary of Changes\n\nVisually polished the Profile section on the Account settings tab:\n\n- **Form layout**: Switched to a responsive 2-column grid (label | input) on desktop, stacking on mobile, for a cleaner aligned look\n- **Email field**: Replaced opacity-dimmed disabled input with a styled read-only display using bg-base-200/60 and a helper text explaining it can't be changed\n- **Profile slug**: Added contextual helper text (format hint when empty, share URL preview when set) with mono font for the URL\n- **Connected accounts**: Added badge-styled status indicators (badge-success/badge-ghost) instead of plain text + dot, with dividers between rows\n- **Proper for/id links**: Added id attributes to inputs matching label for attributes for accessibility
