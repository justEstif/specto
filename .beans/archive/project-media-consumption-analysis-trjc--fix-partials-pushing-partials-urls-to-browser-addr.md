---
# project-media-consumption-analysis-trjc
title: Fix partials pushing /partials/ URLs to browser address bar
status: completed
type: bug
priority: normal
created_at: 2026-03-14T21:49:19Z
updated_at: 2026-03-14T21:50:41Z
---

Dashboard and attention page filters use hx-push-url="true" which pushes /partials/... URLs. Should push the page URL instead (e.g. /dashboard, /attention).

## Summary of Changes\n\nChanged `hx-push-url="true"` to `hx-push-url="false"` on dashboard and attention page filter controls (selects + range tabs) in `components/dashboard.templ` and `components/attention.templ`. This prevents `/partials/...` URLs from being pushed to the browser address bar. The timeline page already had no `hx-push-url` so no change needed there.
