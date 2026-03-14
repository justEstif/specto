---
# project-media-consumption-analysis-9usn
title: Settings pages (account, appearance, sharing)
status: completed
type: feature
priority: normal
created_at: 2026-03-13T15:23:40Z
updated_at: 2026-03-13T15:41:20Z
parent: project-media-consumption-analysis-nn0q
blocked_by:
    - project-media-consumption-analysis-d41y
---

Settings at /settings with three tabs: Account, Appearance, Sharing. Tabs swap content via hx-get with hx-push-url. Account: profile form (display name, email disabled, slug), connected OAuth accounts, danger zone (delete). Appearance: dark/light theme toggle via DaisyUI theme-controller. Sharing: see separate bean. Refs: docs/ui-design.md §Settings, docs/api.md.

## Tasks
- [ ] Settings page templ template (components/settings.templ)
- [ ] Tab navigation: Account, Appearance, Sharing — hx-get /settings/{tab} with hx-push-url
- [ ] Account tab: profile fieldset (display name, email readonly, profile slug with preview URL)
- [ ] Account tab: 'Save changes' — hx-put with inline validation
- [ ] Account tab: connected accounts fieldset (Google/GitHub status + link/unlink)
- [ ] Account tab: danger zone fieldset with 'Delete account' btn-error + confirmation modal
- [ ] Appearance tab: dark/light theme radio toggle (DaisyUI theme-controller, client-side only)
- [ ] Partials: /partials/settings/account, /partials/settings/appearance, /partials/settings/sharing
- [ ] Handler + route wiring

## Summary of Changes

Built the settings pages at /settings with three tabs:
- **Account tab**: Profile form (display name, email read-only, profile slug with URL preview), connected OAuth accounts (Google/GitHub status), danger zone with delete account modal
- **Appearance tab**: Dark/light theme toggle using DaisyUI theme-controller radio inputs (pure client-side)
- **Sharing tab**: Placeholder for the separate share settings bean
- Tab switching via HTMX partials with hx-push-url for bookmarkable URLs
- Profile save via PUT /settings/account with inline success/error messages
- All tabs work as both full-page loads and HTMX partial swaps

### Files created
- components/settings.templ - SettingsPage, tabs, Account/Appearance/Sharing tab content
- internal/handlers/settings_page.go - Page + partial handlers for all three tabs, account update
