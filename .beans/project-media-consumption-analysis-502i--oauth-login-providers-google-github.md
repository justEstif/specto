---
# project-media-consumption-analysis-502i
title: OAuth login providers (Google, GitHub)
status: completed
type: feature
priority: normal
created_at: 2026-03-12T21:39:14Z
updated_at: 2026-03-14T03:49:00Z
parent: project-media-consumption-analysis-bja8
blocked_by:
    - project-media-consumption-analysis-eo0f
---

Add OAuth login as an alternative to email+password. Implement Google and GitHub as initial providers. Uses the same session system — OAuth is just another way to create/authenticate a user. Safe to add after core features since it only touches the server/auth layer, not the domain pipeline.
