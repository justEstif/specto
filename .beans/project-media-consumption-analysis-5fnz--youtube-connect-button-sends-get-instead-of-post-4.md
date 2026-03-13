---
# project-media-consumption-analysis-5fnz
title: YouTube connect button sends GET instead of POST (405 Method Not Allowed)
status: completed
type: bug
priority: high
created_at: 2026-03-13T17:55:19Z
updated_at: 2026-03-13T18:34:46Z
---

## Problem

Same issue as Spotify — clicking the YouTube connect button sends a GET to `/api/v1/plugins/youtube-api/connect` but the route expects POST, returning 405.

## Console Output

```
htmx.min.js:1  GET http://localhost:3000/api/v1/plugins/youtube-api/connect 405 (Method Not Allowed)
Response Status Error Code 405 from /api/v1/plugins/youtube-api/connect
```

## Likely Cause

Same root cause as the Spotify bug — the connect buttons use hx-get instead of hx-post. Probably a shared templ component for plugin connect buttons.

## To Investigate

- [ ] Check if there's a shared connect button component used by all plugins
- [ ] Fix hx-get → hx-post (or match route method)
- [ ] Ensure CSRF token is included
- [ ] Verify YouTube OAuth flow initiates correctly

## Summary of Changes

Fixed by the same change as the Spotify bug — both use the shared `PluginCard` component in `components/plugins.templ`. The OAuth connect button now uses `hx-post` instead of a plain `<a href>`.
