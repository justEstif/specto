---
# project-media-consumption-analysis-uuzz
title: Create docs/sharing.md
status: completed
type: task
priority: normal
created_at: 2026-03-10T22:14:24Z
updated_at: 2026-03-11T12:42:57Z
---

Privacy model, what's public by default, URL structure, data granularity controls. Addresses open questions from MVP.md.

## Summary of Changes\n\nCreated docs/sharing.md covering:\n- Core principle: default to private, opt-in to sharing\n- Block-based profile composition (top genres, mood profile, top creators, recent favorites, etc.)\n- What is never shared (full history, timestamps, time spent, raw metadata)\n- Three exclusion levels: platform, tag, individual item\n- share_profiles table with JSONB block config\n- Item-level privacy flag on media_items\n- Preview before publishing\n- Routes for settings, preview, enable/disable\n\nAlso resolved last open question in MVP.md.
