---
# project-media-consumption-analysis-91nr
title: Post-signup welcome flow
status: in-progress
type: feature
priority: high
created_at: 2026-03-14T22:07:52Z
updated_at: 2026-03-14T22:30:08Z
parent: project-media-consumption-analysis-iq3r
---

Lighter alternative: redirect first-login users to /plugins with a welcome banner, redirect to dashboard after first import.

## Tasks
- [ ] Add onboarded bool to users table (migration + sqlc)
- [ ] Redirect to /plugins on first login (when onboarded=false)
- [ ] Add welcome banner to plugins page for new users
- [ ] Mark onboarded=true after first successful import/sync
- [ ] Build and test
