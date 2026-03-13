---
# project-media-consumption-analysis-ra9i
title: Unified file upload option for manual imports
status: completed
type: feature
priority: normal
created_at: 2026-03-13T17:57:10Z
updated_at: 2026-03-13T18:37:54Z
---

## Problem

Currently file-based imports (CSV, JSON exports from platforms) may be listed as separate options. It would be cleaner to have a single "Upload" option where the user selects a file and specifies the platform/source.

## Requirements

- [ ] Single upload entry point in the available integrations section
- [ ] User selects the platform (e.g., Spotify, YouTube, Last.fm, etc.)
- [ ] User uploads the export file
- [ ] Add tooltip or FAQ section explaining accepted file types per platform
- [ ] Validate file type matches selected platform

## Open Questions

- What file formats are accepted per platform? (CSV, JSON, etc.)
- Should we auto-detect the platform from the file contents?

## Summary of Changes

Replaced individual file-import plugin cards in the Available section with a unified `FileImportCard` component in `components/plugins.templ`:

- Single "Import from file" card with upload icon
- Platform dropdown selector (populates from all available file-import plugins)
- File input with .csv/.json/.txt accept
- Collapsible "Accepted file formats" FAQ section showing per-platform format hints (Spotify extended streaming history JSON, YouTube Takeout watch-history.json)
- Dynamic form action — `hx-post` URL updates based on selected platform via small JS helper
- Upload button disabled until platform is selected
- OAuth plugins still render as individual cards above the upload card
- SVG logos now use `w-full h-full` to scale to their container size
