---
# project-media-consumption-analysis-y2c3
title: Goodreads plugin (CSV file import)
status: completed
type: feature
priority: normal
created_at: 2026-03-15T01:25:00Z
updated_at: 2026-03-15T01:35:14Z
parent: project-media-consumption-analysis-geho
blocked_by:
    - project-media-consumption-analysis-7fan
---

Implement Goodreads SourcePlugin parsing library export CSV. Rich data: title, author, ISBN, rating, dates, shelves, page count, review. Strip ="" ISBN wrapper. Map Exclusive Shelf to status. Use Goodreads Book Id as ExternalID.

Depends on MediaBook type being added first.

## Tasks
- [ ] Create internal/plugins/goodreads/ package
- [ ] Implement Plugin struct with SourcePlugin interface
- [ ] Parse goodreads_library_export.csv (31 columns)
- [ ] Strip ="" wrapper from ISBN/ISBN13 fields
- [ ] Map Exclusive Shelf to status (read/currently-reading/to-read)
- [ ] Handle My Rating = 0 as unrated (not zero stars)
- [ ] Store rich metadata: page count, publisher, binding, read count, review
- [ ] Use Book Id as ExternalID for dedup
- [ ] Implement Enrich() as no-op (Open Library enrichment deferred)
- [ ] Write comprehensive tests
- [ ] Register plugin in cmd/web/main.go
- [ ] Add import guide modal for Goodreads

## Summary of Changes\n\nImplemented in project-media-consumption-analysis-mqgb. Plugin created at internal/plugins/goodreads/ with full CSV parsing and test coverage.

## Summary of Changes\nImplemented Goodreads CSV import plugin with ISBN stripping, shelf-to-status mapping, rating handling, date fallback. 17 tests passing.
