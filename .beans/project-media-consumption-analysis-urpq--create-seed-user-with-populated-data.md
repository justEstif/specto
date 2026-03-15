---
# project-media-consumption-analysis-urpq
title: Create seed user with populated data
status: completed
type: task
priority: normal
created_at: 2026-03-15T00:44:30Z
updated_at: 2026-03-15T01:04:15Z
---

Create a Go seed script and mise task that inserts a test user (test@email.com / password123) with realistic media items, tags, and enrichment data so we don't need LLM tokens or real APIs for testing.

## Summary of Changes

- Created cmd/seed/main.go — standalone Go command that seeds the database with a test user and realistic media data
- Added mise run seed task to mise.toml
- Seed is idempotent (deletes and re-creates on re-run)



### Update: Expanded to 2 years
- 184 items total (101 music, 58 video, 25 podcast)
- Spans ~722 days with distinct taste phases per media type
- 890 tag assignments across 58 unique tags
- Music: 4 eras (indie/dreamy → hip-hop/intense → electronic/energetic → r&b/melancholic)
- Video: 3 eras (science/math → programming/design → philosophy/contemplative)  
- Podcast: 2 eras (tech/business → culture/philosophy)
