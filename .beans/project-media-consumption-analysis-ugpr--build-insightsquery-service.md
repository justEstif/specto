---
# project-media-consumption-analysis-ugpr
title: Build insights/query service
status: completed
type: task
priority: normal
created_at: 2026-03-12T20:21:23Z
updated_at: 2026-03-12T21:17:24Z
parent: project-media-consumption-analysis-86vz
blocked_by:
    - project-media-consumption-analysis-6o1q
---

Create internal/core/insights.go wrapping the existing sqlc queries (PlatformBreakdown, TagDistribution, ListMediaItems, etc.) with domain-level service functions. Includes: GetSummary(ctx, userID) returning total items, total watch time, top platform, top media type; GetTimeline(ctx, userID, bucket, dateRange) for time-bucketed consumption data; GetPlatformBreakdown(ctx, userID); GetTagDistribution(ctx, userID, category, minConfidence) with confidence threshold filtering. Return domain types, not database models. Include unit tests.


## Todo

- [x] Survey existing sqlc queries for insights capabilities
- [x] Define domain result types (Summary, TimelineEntry, PlatformBreakdown, TagDistribution)
- [x] Implement InsightsService with dependency injection
- [x] Implement GetSummary (total items, total time, top platform, top media type)
- [x] Implement GetTimeline (time-bucketed consumption data)
- [x] Implement GetPlatformBreakdown
- [x] Implement GetTagDistribution with confidence threshold filtering
- [x] Write comprehensive unit tests
- [x] Run tests and verify compilation


## Summary of Changes

Implemented the insights/query service layer:

- **Domain types** in `core/stores.go`: `Summary`, `TimelineEntry`, `PlatformBreakdownEntry`, `TagDistributionEntry`, `TimeBucket` (day/week/month)
- **InsightsStore interface** in `core/stores.go`: `PlatformBreakdown`, `TagDistribution`, `ListMediaItems`
- **InsightsService** in `core/insights.go`: `GetSummary`, `GetTimeline`, `GetPlatformBreakdown`, `GetTagDistribution` with date range validation, automatic pagination, contiguous bucket generation (fills empty time buckets with zeros), and default limit handling
- **PgInsightsStore** in `core/store/insights_store.go`: PostgreSQL implementation using sqlc-generated queries
- **Querier interface** updated with `PlatformBreakdown` and `TagDistribution` methods
- **mockQuerier** updated with corresponding mock functions
- **26 unit tests** across `core/insights_test.go` (20) and `core/store/insights_store_test.go` (4) + 2 helper tests
