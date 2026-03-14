---
# project-media-consumption-analysis-91nr
title: Post-signup welcome flow
status: completed
type: feature
priority: high
created_at: 2026-03-14T22:07:52Z
updated_at: 2026-03-14T22:36:54Z
parent: project-media-consumption-analysis-iq3r
---

Lighter alternative: redirect first-login users to /plugins with a welcome banner, redirect to dashboard after first import.

## Tasks
- [ ] Add onboarded bool to users table (migration + sqlc)
- [ ] Redirect to /plugins on first login (when onboarded=false)
- [ ] Add welcome banner to plugins page for new users
- [ ] Mark onboarded=true after first successful import/sync
- [ ] Build and test

## Summary of Changes

Lightweight onboarding: new users are redirected from / to /plugins on first login. A DaisyUI alert banner welcomes them and tells them to connect a source. After first OAuth connect or file import, the user is marked as onboarded and the banner disappears. No new routes, no wizard — just a redirect and a banner.
