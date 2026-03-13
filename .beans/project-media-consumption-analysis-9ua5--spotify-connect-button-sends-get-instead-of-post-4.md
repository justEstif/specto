---
# project-media-consumption-analysis-9ua5
title: Spotify connect button sends GET instead of POST (405 Method Not Allowed)
status: completed
type: bug
priority: high
created_at: 2026-03-13T17:55:06Z
updated_at: 2026-03-13T18:34:46Z
---

## Problem

Clicking the Spotify connect/integrate button sends a GET to `/api/v1/plugins/spotify-api/connect` but the route expects POST, returning 405 Method Not Allowed.

## Console Output

```
htmx.min.js:1  GET http://localhost:3000/api/v1/plugins/spotify-api/connect 405 (Method Not Allowed)
Response Status Error Code 405 from /api/v1/plugins/spotify-api/connect
```

## Likely Cause

The button/link uses `hx-get` (or a plain `<a>` tag) instead of `hx-post` for the Spotify connect action.

## To Investigate

- [ ] Find the templ component rendering the Spotify connect button
- [ ] Change from hx-get/href to hx-post (or vice versa, match the route method)
- [ ] Ensure CSRF token is included if switching to POST
- [ ] Verify Spotify OAuth flow initiates correctly

## Summary of Changes

Changed the OAuth connect button in `components/plugins.templ` from a plain `<a href>` (GET) to a `<button hx-post>` with CSRF header and client-side redirect handling. The button now POSTs to the connect endpoint, receives the OAuth redirect URL in JSON, and navigates to it via JavaScript.
