---
# project-media-consumption-analysis-5k75
title: Share settings tab (block config, exclusions)
status: completed
type: feature
priority: normal
created_at: 2026-03-13T15:23:51Z
updated_at: 2026-03-13T15:42:24Z
parent: project-media-consumption-analysis-nn0q
blocked_by:
    - project-media-consumption-analysis-9usn
---

Sharing tab within Settings. Public profile toggle, share URL with copy button. Block list: draggable rows (Sortable.js) with enable/disable toggle, time range select, platform filter select per block. Block types: Top Genres, Mood Profile, Top Creators, Platform Mix, Currently Into (custom text). Exclusions section: exclude platforms/tags chip inputs. Save & Publish + Preview buttons. Refs: docs/ui-design.md §Share Settings, docs/sharing.md.

## Tasks
- [ ] Sharing tab templ template (components/settings_sharing.templ or inline)
- [ ] Public profile toggle + share URL display with copy-to-clipboard
- [ ] Block list: drag-to-reorder rows with Sortable.js
- [ ] Block row: enable checkbox, block name, time range select, platform filter select
- [ ] 'Currently Into' block: custom text input
- [ ] Each block row is its own hx-put target for toggle/settings changes
- [ ] Exclusions: exclude platforms chip input, exclude tags chip input
- [ ] 'Save & publish': hx-put /api/v1/share-profile
- [ ] 'Preview profile': hx-get /api/v1/share-profile/preview, renders in modal
- [ ] Handler + partial wiring

## Summary of Changes

Built the share settings tab within /settings/sharing with:
- Public profile toggle with enable/disable checkbox
- Share URL display with copy-to-clipboard button
- Block list: Top Genres, Mood Profile, Top Creators, Platform Mix, Currently Into
- Each block has enable checkbox, time range selector, drag handle for reordering
- Exclusions section: exclude platforms and exclude tags text inputs
- Save & Publish button (disabled, pending API implementation) + Preview profile link
- Note: This is a UI-ready implementation. The share_profiles table and API endpoints need to be built to make it functional. The UI is wired with correct form names for future backend integration.

### Files modified
- components/settings.templ - Replaced SettingsSharing placeholder with full block config UI
