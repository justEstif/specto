---
# project-media-consumption-analysis-00xx
title: Plugins page
status: completed
type: feature
priority: normal
created_at: 2026-03-13T15:23:31Z
updated_at: 2026-03-13T15:39:45Z
parent: project-media-consumption-analysis-nn0q
blocked_by:
    - project-media-consumption-analysis-d41y
---

Plugin management at /plugins. Two sections: Connected (with sync/disconnect actions) and Available (with connect/upload). Plugin cards show platform icon, auth type, connection status dot, item count, last sync time. Refs: docs/ui-design.md §Plugins, docs/api.md (plugin endpoints).

## Tasks
- [ ] Plugins page templ template (components/plugins.templ)
- [ ] Plugin card component: icon, name, auth type, status dot, item count, last sync
- [ ] Connected section: cards with 'Sync now' + 'Disconnect' buttons
- [ ] Available section: cards with 'Connect' link (OAuth) or file upload input
- [ ] 'Sync now': hx-post /api/v1/plugins/{plugin}/sync, swaps card with updated data
- [ ] 'Disconnect': hx-delete /api/v1/plugins/{plugin}/disconnect with hx-confirm
- [ ] File upload: hx-post with hx-encoding=multipart/form-data, progress indicator
- [ ] Partial: /partials/plugin-card/{plugin} — returns single card
- [ ] Responsive: single column, edge-to-edge cards on mobile
- [ ] Handler + route wiring

## Summary of Changes

Built the plugins management page at /plugins with:
- Two sections: Connected and Available, separated based on plugin state
- Plugin cards: platform icon, display name, auth type label, connection status dot, last sync time, error messages
- Connected plugin actions: Sync now (hx-post), Disconnect (hx-delete with confirm dialog)
- Available OAuth plugins: Connect link
- File import plugins: file upload form with hx-encoding=multipart/form-data
- CSRF token passed for all mutating actions
- Responsive: single column, 44px touch targets

### Files created
- components/plugins.templ - PluginsPage, PluginCard, helper functions
- internal/handlers/plugins_page.go - PluginsPage handler, plugin view builder
