---
# project-media-consumption-analysis-ugpr
title: Build insights/query service
status: todo
type: task
created_at: 2026-03-12T20:21:23Z
updated_at: 2026-03-12T20:21:23Z
parent: project-media-consumption-analysis-86vz
blocked_by:
    - project-media-consumption-analysis-6o1q
---

Create internal/core/insights.go wrapping the existing sqlc queries (PlatformBreakdown, TagDistribution, ListMediaItems, etc.) with domain-level service functions. Includes: GetSummary(ctx, userID) returning total items, total watch time, top platform, top media type; GetTimeline(ctx, userID, bucket, dateRange) for time-bucketed consumption data; GetPlatformBreakdown(ctx, userID); GetTagDistribution(ctx, userID, category, minConfidence) with confidence threshold filtering. Return domain types, not database models. Include unit tests.
