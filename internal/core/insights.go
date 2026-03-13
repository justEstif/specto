package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// InsightsService provides aggregated analytics over a user's media
// consumption data. It delegates storage queries to an InsightsStore and
// applies domain logic (e.g. timeline bucketing, summary derivation).
type InsightsService struct {
	store InsightsStore
}

// NewInsightsService creates an InsightsService.
func NewInsightsService(store InsightsStore) *InsightsService {
	return &InsightsService{store: store}
}

// GetSummary returns a high-level overview: total items, total duration,
// top platform, and top media type for the given date range.
func (s *InsightsService) GetSummary(ctx context.Context, userID uuid.UUID, from, to time.Time) (*Summary, error) {
	return s.GetSummaryFiltered(ctx, userID, from, to, InsightsFilter{})
}

// GetSummaryFiltered returns a high-level overview with optional platform
// and media type filters applied.
func (s *InsightsService) GetSummaryFiltered(ctx context.Context, userID uuid.UUID, from, to time.Time, filter InsightsFilter) (*Summary, error) {
	if err := validateDateRange(from, to); err != nil {
		return nil, err
	}

	breakdown, err := s.store.PlatformBreakdownFiltered(ctx, userID, from, to, filter)
	if err != nil {
		return nil, fmt.Errorf("getting summary: %w", err)
	}

	summary := &Summary{}
	platformCounts := make(map[string]int64)
	typeCounts := make(map[string]int64)

	for _, entry := range breakdown {
		summary.TotalItems += entry.Count
		summary.TotalDurationSec += entry.TotalDurationSec
		platformCounts[entry.Platform] += entry.Count
		typeCounts[entry.MediaType] += entry.Count
	}

	summary.TopPlatform = maxKey(platformCounts)
	summary.TopMediaType = maxKey(typeCounts)

	return summary, nil
}

// GetTimeline returns time-bucketed consumption data. Each entry
// represents one bucket (day/week/month) with the count and total
// duration of items consumed in that bucket.
func (s *InsightsService) GetTimeline(ctx context.Context, userID uuid.UUID, bucket TimeBucket, from, to time.Time) ([]TimelineEntry, error) {
	return s.GetTimelineFiltered(ctx, userID, bucket, from, to, InsightsFilter{})
}

// GetTimelineFiltered returns time-bucketed consumption data with optional
// platform and media type filters applied.
func (s *InsightsService) GetTimelineFiltered(ctx context.Context, userID uuid.UUID, bucket TimeBucket, from, to time.Time, filter InsightsFilter) ([]TimelineEntry, error) {
	if err := validateDateRange(from, to); err != nil {
		return nil, err
	}
	if err := validateBucket(bucket); err != nil {
		return nil, err
	}

	// Fetch all items in the range. We use a high limit and paginate
	// through all results to build the timeline in-memory. For MVP scale
	// this is fine — a single user won't have millions of items.
	hasFilter := filter.Platform != nil || filter.MediaType != nil
	const pageSize int32 = 500
	var allItems []MediaItem
	for offset := int32(0); ; offset += pageSize {
		var page []MediaItem
		var err error
		if hasFilter {
			page, err = s.store.ListMediaItemsFiltered(ctx, userID, from, to, pageSize, offset, filter)
		} else {
			page, err = s.store.ListMediaItems(ctx, userID, from, to, pageSize, offset)
		}
		if err != nil {
			return nil, fmt.Errorf("fetching items for timeline: %w", err)
		}
		allItems = append(allItems, page...)
		if int32(len(page)) < pageSize {
			break
		}
	}

	// Build a map from bucket key to aggregated entry.
	bucketMap := make(map[time.Time]*TimelineEntry)
	for _, item := range allItems {
		key := truncateToBucket(item.ConsumedAt, bucket)
		entry, ok := bucketMap[key]
		if !ok {
			entry = &TimelineEntry{Bucket: key}
			bucketMap[key] = entry
		}
		entry.Count++
		if item.Duration != nil {
			entry.TotalDurationSec += int64(item.Duration.Seconds())
		}
	}

	// Generate a contiguous sequence of buckets from 'from' to 'to',
	// filling in zeros for empty buckets.
	timeline := generateBucketSequence(from, to, bucket, bucketMap)
	return timeline, nil
}

// GetPlatformBreakdown returns consumption stats grouped by platform
// and media type.
func (s *InsightsService) GetPlatformBreakdown(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]PlatformBreakdownEntry, error) {
	return s.GetPlatformBreakdownFiltered(ctx, userID, from, to, InsightsFilter{})
}

// GetPlatformBreakdownFiltered returns consumption stats with optional filters.
func (s *InsightsService) GetPlatformBreakdownFiltered(ctx context.Context, userID uuid.UUID, from, to time.Time, filter InsightsFilter) ([]PlatformBreakdownEntry, error) {
	if err := validateDateRange(from, to); err != nil {
		return nil, err
	}

	entries, err := s.store.PlatformBreakdownFiltered(ctx, userID, from, to, filter)
	if err != nil {
		return nil, fmt.Errorf("getting platform breakdown: %w", err)
	}

	return entries, nil
}

// GetTagDistribution returns tag usage counts within a date range.
// Only tags with confidence >= the hardcoded threshold (0.7) or
// authoritative (NULL confidence) tags are included. Results are
// ordered by count descending.
func (s *InsightsService) GetTagDistribution(ctx context.Context, userID uuid.UUID, from, to time.Time, limit int32) ([]TagDistributionEntry, error) {
	return s.GetTagDistributionFiltered(ctx, userID, from, to, limit, InsightsFilter{})
}

// GetTagDistributionFiltered returns tag distribution with optional filters.
func (s *InsightsService) GetTagDistributionFiltered(ctx context.Context, userID uuid.UUID, from, to time.Time, limit int32, filter InsightsFilter) ([]TagDistributionEntry, error) {
	if err := validateDateRange(from, to); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50 // sensible default
	}

	entries, err := s.store.TagDistributionFiltered(ctx, userID, from, to, limit, filter)
	if err != nil {
		return nil, fmt.Errorf("getting tag distribution: %w", err)
	}

	return entries, nil
}

// --- helpers ---

func validateDateRange(from, to time.Time) error {
	if from.IsZero() {
		return errors.New("insights: from time must not be zero")
	}
	if to.IsZero() {
		return errors.New("insights: to time must not be zero")
	}
	if to.Before(from) {
		return errors.New("insights: to must be after from")
	}
	return nil
}

func validateBucket(b TimeBucket) error {
	switch b {
	case BucketDay, BucketWeek, BucketMonth:
		return nil
	default:
		return fmt.Errorf("insights: invalid time bucket %q (must be day, week, or month)", b)
	}
}

// maxKey returns the key with the highest value in the map, or "" if empty.
func maxKey(m map[string]int64) string {
	var maxK string
	var maxV int64
	for k, v := range m {
		if v > maxV || (v == maxV && k < maxK) {
			maxK = k
			maxV = v
		}
	}
	return maxK
}

// truncateToBucket truncates a timestamp to the start of its bucket.
func truncateToBucket(t time.Time, bucket TimeBucket) time.Time {
	t = t.UTC()
	switch bucket {
	case BucketDay:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	case BucketWeek:
		// ISO week: Monday is the first day of the week.
		weekday := t.Weekday()
		if weekday == time.Sunday {
			weekday = 7
		}
		d := t.AddDate(0, 0, -int(weekday-time.Monday))
		return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	case BucketMonth:
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	default:
		return t
	}
}

// generateBucketSequence creates a contiguous series of TimelineEntries
// from 'from' to 'to', filling in data from the bucketMap and inserting
// zero-value entries for empty buckets.
func generateBucketSequence(from, to time.Time, bucket TimeBucket, data map[time.Time]*TimelineEntry) []TimelineEntry {
	start := truncateToBucket(from, bucket)
	end := truncateToBucket(to, bucket)

	var result []TimelineEntry
	for current := start; !current.After(end); current = advanceBucket(current, bucket) {
		if entry, ok := data[current]; ok {
			result = append(result, *entry)
		} else {
			result = append(result, TimelineEntry{Bucket: current})
		}
	}
	return result
}

// advanceBucket moves a time forward by one bucket period.
func advanceBucket(t time.Time, bucket TimeBucket) time.Time {
	switch bucket {
	case BucketDay:
		return t.AddDate(0, 0, 1)
	case BucketWeek:
		return t.AddDate(0, 0, 7)
	case BucketMonth:
		return t.AddDate(0, 1, 0)
	default:
		return t.AddDate(0, 0, 1)
	}
}
