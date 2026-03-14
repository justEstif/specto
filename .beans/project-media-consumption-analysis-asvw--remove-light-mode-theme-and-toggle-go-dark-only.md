---
# project-media-consumption-analysis-asvw
title: Remove light mode theme and toggle — go dark-only
status: completed
type: task
priority: normal
created_at: 2026-03-14T23:33:05Z
updated_at: 2026-03-14T23:34:26Z
---

Remove the specto-light theme, the theme toggle in settings/appearance, and related CSS. Commit to dark-only aesthetic.

## Summary of Changes

Removed the light mode theme toggle and committed to a dark-only aesthetic:

- **styles/input.css**: Removed the entire `specto-light` theme definition, its scrollbar/selection styles, and the unused `dark:` custom variant
- **components/settings.templ**: Removed the `SettingsAppearance` component (theme toggle UI) and the Appearance tab from the settings tab bar
- **internal/handlers/settings_page.go**: Removed the `appearance` tab from the partial routing map

The app now has only the `specto-dark` theme with no theme switching UI.
