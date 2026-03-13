---
# project-media-consumption-analysis-86sq
title: Build sharing page end-to-end
status: todo
type: feature
created_at: 2026-03-13T21:20:34Z
updated_at: 2026-03-13T21:20:34Z
---

## Todo

- [ ] Create `share_profiles` table migration (blocks JSONB, excluded_platforms TEXT[], excluded_tags TEXT[])
- [ ] Add `private BOOLEAN DEFAULT false` column to `media_items`
- [ ] Write sqlc queries for share profile CRUD + block data aggregation
- [ ] Create `ShareProfileStore` interface + pgx implementation
- [ ] Implement API endpoints: GET/PUT `/api/v1/share-profile`, GET `/api/v1/share-profile/preview`, POST `/api/v1/items/{id}/privacy`
- [ ] Wire real data into `ShareProfilePage` handler (replace hardcoded empty blocks)
- [ ] Upgrade excluded platforms input to searchable multi-select component
- [ ] Upgrade excluded tags input to searchable multi-select component
- [ ] Add per-block platform filter multi-selects (in wireframe but missing from template)
- [ ] Wire Save & publish button to API
- [ ] Wire Sortable.js for block drag reorder
- [ ] Register share-profile API routes in main.go
