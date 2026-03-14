---
# project-media-consumption-analysis-vcah
title: Add platform logos to plugin connect buttons
status: completed
type: feature
priority: normal
created_at: 2026-03-13T17:57:04Z
updated_at: 2026-03-13T18:36:00Z
---

## Problem

The available integrations section shows plain text for Spotify and YouTube connect buttons. It would look better with recognizable platform logos.

## Requirements

- [ ] Add Spotify logo/icon to the Spotify connect button
- [ ] Add YouTube logo/icon to the YouTube connect button
- [ ] Ensure logos look good in both light and dark themes
- [ ] Keep consistent sizing and alignment

## Summary of Changes

Replaced emoji icons with proper SVG brand logos in `components/plugins.templ`:
- Spotify: Official logo in brand green (#1DB954)
- YouTube: Official play button logo in brand red (#FF0000)
- Fallback: 🔌 emoji for unknown plugins

Changed `pluginIcon()` Go function to `pluginLogo()` templ component for inline SVG rendering. Logo container is 32x32px with flex centering.
