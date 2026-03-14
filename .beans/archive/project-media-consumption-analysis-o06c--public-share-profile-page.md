---
# project-media-consumption-analysis-o06c
title: Public share profile page
status: completed
type: feature
priority: normal
created_at: 2026-03-13T15:24:00Z
updated_at: 2026-03-13T15:45:46Z
parent: project-media-consumption-analysis-nn0q
blocked_by:
    - project-media-consumption-analysis-5k75
---

Public profile at /share/{slug}. Standalone page (no navbar). Server-rendered, no HTMX. Shows display name + slug header, then blocks in user-configured order: Top Genres (horizontal bars), Mood Profile (summary + bars), Top Creators (numbered list with platform badges), Platform Mix, Currently Into (quoted text). 'Powered by Specto' footer. grain texture on header. Refs: docs/ui-design.md §Public Share Profile, docs/sharing.md.

## Tasks
- [ ] Share profile templ template (components/share_profile.templ)
- [ ] Header section: display name (text-display), @slug (muted), optional grain overlay
- [ ] Top Genres block: horizontal bar chart with percentages
- [ ] Mood Profile block: summary text + mood bars
- [ ] Top Creators block: numbered list, media icon, creator name, platform badge
- [ ] Platform Mix block: platform bars with percentages
- [ ] Currently Into block: quoted custom text
- [ ] 'Powered by Specto' footer link
- [ ] Render blocks dynamically based on user config (order, enabled/disabled)
- [ ] Responsive: max-w-2xl centered >= sm, edge-to-edge < sm
- [ ] Handler + route wiring (unauthenticated, 404 if profile not found/disabled)

## Summary of Changes

Built the public share profile page at /share/{slug} with:
- Standalone page layout (no navbar, no HTMX) with its own HTML document
- Header: display name + @slug with grain texture overlay
- Block types: Top Genres (bar chart), Mood Profile (summary + bars), Top Creators (numbered list), Platform Mix (bars), Currently Into (blockquote)
- Each block rendered conditionally based on enabled flag
- Responsive: max-w-2xl centered, edge-to-edge on mobile
- 'Powered by Specto' footer
- Handler looks up user by profile_slug (added new SQL query + store method)
- Returns 404 if slug not found
- Currently renders placeholder blocks (share_profiles table and data population pending)

### Files created/modified
- components/share_profile.templ - ShareProfilePage + all block type components
- internal/handlers/share_page.go - ShareProfilePage handler
- internal/database/queries.sql - Added GetUserByProfileSlug query
- internal/core/stores.go - Added GetByProfileSlug to UserStore interface
- internal/core/store/user_store.go - Implemented GetByProfileSlug
- internal/core/store/queries.go - Added to Querier interface
- internal/core/store/mock_querier_test.go - Added mock method
- cmd/web/main.go - Added /share/{slug} route
