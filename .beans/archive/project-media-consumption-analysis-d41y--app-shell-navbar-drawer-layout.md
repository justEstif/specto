---
# project-media-consumption-analysis-d41y
title: App shell (navbar, drawer, layout)
status: completed
type: feature
priority: high
created_at: 2026-03-13T15:22:47Z
updated_at: 2026-03-13T15:31:49Z
parent: project-media-consumption-analysis-nn0q
---

Build the authenticated and unauthenticated app shell: sticky navbar with backdrop-blur, DaisyUI drawer for mobile hamburger menu, avatar dropdown (Settings/Share/Sign out), page-loader bar. Uses vt-navbar for view transition persistence. Refs: docs/ui-design.md §App Shell, components/layout.templ, components/navbar.templ.

## Tasks
- [x] Authenticated navbar: logo, Timeline link, Plugins link, avatar dropdown
- [x] Unauthenticated navbar: logo + Sign in button
- [x] Mobile drawer (DaisyUI drawer component, hamburger < sm)
- [x] Avatar dropdown menu (Settings, Share, Sign out links)
- [x] Wire navbar to show correct state based on session
- [x] Ensure vt-navbar region persists across view transitions
- [x] 44px minimum touch targets on all interactive elements

## Summary of Changes

Built the auth-aware app shell with:
- **Navbar**: Two variants — authenticated (logo, Timeline, Plugins links, avatar dropdown with Settings/Share/Sign out) and unauthenticated (logo + Sign in button)
- **Mobile drawer**: DaisyUI drawer component with hamburger toggle visible below sm breakpoint, full slide-out menu with all nav links
- **Avatar dropdown**: Shows user initial in a circle, dropdown with display name, Settings, Share profile, Sign out
- **OptionalAuth middleware**: New middleware that loads user into context if session exists but doesn't reject anonymous visitors
- **Layout**: Restructured to wrap authenticated pages in DaisyUI drawer, pass user to navbar, added Google Fonts (Playfair Display, DM Sans, JetBrains Mono)
- **Home page**: Updated to pass user through, added 3-column feature card grid
- **44px touch targets**: All interactive navbar elements meet minimum touch target size

### Files modified
- `components/navbar.templ` — Full rewrite with auth-aware navbar + drawer sidebar
- `components/layout.templ` — Drawer wrapper, user param, Google Fonts
- `components/home.templ` — User param passthrough, feature grid
- `internal/handlers/home.go` — Extract user from context
- `internal/middleware/auth.go` — Added OptionalAuth middleware
- `cmd/web/main.go` — Wired OptionalAuth on HTML routes
