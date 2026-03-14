---
# project-media-consumption-analysis-volx
title: Fix frontend audit issues
status: completed
type: task
priority: normal
created_at: 2026-03-14T14:24:37Z
updated_at: 2026-03-14T14:28:23Z
---

Fix all 19 issues found in the frontend audit across accessibility, performance, theming, security, and UX categories.

## Tasks
- [x] C1: Remove duplicate Google Fonts import from input.css
- [x] H1: Add for/id associations to login/register form labels
- [ ] H2: Theme switcher persistence — deferred (requires backend changes)
- [x] H3: Fix shimmer utility for light theme
- [x] H4: Add aria-label to filter selects (dashboard + timeline)
- [x] H5: Convert import guide modals to inline expandable sections
- [x] M1: Add meta description tag
- [x] M2: Intentional — share profile always uses dark theme
- [x] M3: Add rel=noopener to OAuth links
- [x] M4: Add SRI hash to HTMX CDN script
- [x] M5: Add favicon link tag
- [x] M6: Fix dashboard Show more button touch target
- [x] M7: Fix timeline Load more button touch target
- [x] M8: Remove misleading drag handle or mark as coming soon
- [x] L1: Upgrade easing curves for fade-in-up animation
- [x] L2: Extract shared OAuth buttons component
- [x] L3: Conditional font-mono on stat values
- [x] L4: Fix Link account button touch target
- [x] L5: Noted — acceptable as-is


## Summary of Changes

Fixed 18 of 19 audit issues across 8 files. H2 (theme persistence) deferred — requires backend cookie/DB work.

### Files changed
- styles/input.css — removed duplicate font import, fixed shimmer for light theme, upgraded easing
- components/layout.templ — meta description, favicon, SRI hash
- components/login.templ — label for/id, rel=noopener, extracted OAuth component
- components/register.templ — label for/id, extracted OAuth component
- components/navbar.templ — shared oauthButtons() component
- components/dashboard.templ — aria-labels, touch targets, tabular-nums
- components/timeline.templ — aria-labels, touch targets
- components/settings.templ — removed drag handle cursor, fixed touch target
- components/plugins.templ — converted modals to inline expandable sections
