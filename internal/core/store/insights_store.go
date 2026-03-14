package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/justestif/specto/internal/core"
	"github.com/justestif/specto/internal/database"
)

// PgInsightsStore implements core.InsightsStore using sqlc-generated queries.
type PgInsightsStore struct {
	q Querier
}

// NewInsightsStore creates a new InsightsStore backed by PostgreSQL.
func NewInsightsStore(q Querier) *PgInsightsStore {
	return &PgInsightsStore{q: q}
}

var _ core.InsightsStore = (*PgInsightsStore)(nil)

func (s *PgInsightsStore) PlatformBreakdown(ctx context.Context, userID uuid.UUID, from, to time.Time, filter core.InsightsFilter) ([]core.PlatformBreakdownEntry, error) {
	rows, err := s.q.PlatformBreakdownFiltered(ctx, database.PlatformBreakdownFilteredParams{
		UserID:       uuidToPgx(userID),
		ConsumedAt:   timestamptz(from),
		ConsumedAt_2: timestamptz(to),
		Platform:     textPtr(filter.Platform),
		MediaType:    textPtr(filter.MediaType),
	})
	if err != nil {
		return nil, fmt.Errorf("querying filtered platform breakdown: %w", err)
	}

	entries := make([]core.PlatformBreakdownEntry, len(rows))
	for i, row := range rows {
		entries[i] = core.PlatformBreakdownEntry{
			Platform:         row.Platform,
			MediaType:        row.Type,
			Count:            row.Count,
			TotalDurationSec: row.TotalDurationSec,
		}
	}
	return entries, nil
}

func (s *PgInsightsStore) TagDistribution(ctx context.Context, userID uuid.UUID, from, to time.Time, limit int32, filter core.InsightsFilter) ([]core.TagDistributionEntry, error) {
	rows, err := s.q.TagDistributionFiltered(ctx, database.TagDistributionFilteredParams{
		UserID:       uuidToPgx(userID),
		ConsumedAt:   timestamptz(from),
		ConsumedAt_2: timestamptz(to),
		Limit:        limit,
		Platform:     textPtr(filter.Platform),
		MediaType:    textPtr(filter.MediaType),
	})
	if err != nil {
		return nil, fmt.Errorf("querying filtered tag distribution: %w", err)
	}

	entries := make([]core.TagDistributionEntry, len(rows))
	for i, row := range rows {
		cat := ""
		if row.Category.Valid {
			cat = row.Category.String
		}
		entries[i] = core.TagDistributionEntry{
			Name:     row.Name,
			Category: cat,
			Count:    row.Count,
		}
	}
	return entries, nil
}

func (s *PgInsightsStore) ListMediaItems(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int32, filter core.InsightsFilter) ([]core.MediaItem, error) {
	// Always use the filtered query — passing nil filters is a no-op.
	rows, err := s.q.ListMediaItemsFiltered(ctx, database.ListMediaItemsFilteredParams{
		UserID:       uuidToPgx(userID),
		ConsumedAt:   timestamptz(from),
		ConsumedAt_2: timestamptz(to),
		Limit:        limit,
		Offset:       offset,
		Platform:     textPtr(filter.Platform),
		MediaType:    textPtr(filter.MediaType),
		Search:       textPtr((*string)(nil)),
	})
	if err != nil {
		return nil, fmt.Errorf("listing media items for insights: %w", err)
	}

	items := make([]core.MediaItem, len(rows))
	for i, row := range rows {
		items[i] = mediaItemFromDB(row)
	}
	return items, nil
}

func (s *PgInsightsStore) TagDistributionByCategory(ctx context.Context, userID uuid.UUID, from, to time.Time, limit int32, category string, filter core.InsightsFilter) ([]core.TagDistributionEntry, error) {
	rows, err := s.q.TagDistributionByCategory(ctx, database.TagDistributionByCategoryParams{
		UserID:       uuidToPgx(userID),
		ConsumedAt:   timestamptz(from),
		ConsumedAt_2: timestamptz(to),
		Limit:        limit,
		Platform:     textPtr(filter.Platform),
		MediaType:    textPtr(filter.MediaType),
		Category:     pgtype.Text{String: category, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("querying tag distribution by category %s: %w", category, err)
	}

	entries := make([]core.TagDistributionEntry, len(rows))
	for i, row := range rows {
		cat := ""
		if row.Category.Valid {
			cat = row.Category.String
		}
		entries[i] = core.TagDistributionEntry{
			Name:     row.Name,
			Category: cat,
			Count:    row.Count,
		}
	}
	return entries, nil
}

func (s *PgInsightsStore) AttentionByType(ctx context.Context, userID uuid.UUID, from, to time.Time, platform *string) ([]core.AttentionByTypeEntry, error) {
	rows, err := s.q.AttentionByType(ctx, database.AttentionByTypeParams{
		UserID:       uuidToPgx(userID),
		ConsumedAt:   timestamptz(from),
		ConsumedAt_2: timestamptz(to),
		Platform:     textPtr(platform),
	})
	if err != nil {
		return nil, fmt.Errorf("querying attention by type: %w", err)
	}

	entries := make([]core.AttentionByTypeEntry, len(rows))
	for i, row := range rows {
		entries[i] = core.AttentionByTypeEntry{
			MediaType:        row.Type,
			Count:            row.Count,
			TotalTimeSpent:   row.TotalTimeSpentSec,
			TotalDurationSec: row.TotalDurationSec,
		}
	}
	return entries, nil
}

func (s *PgInsightsStore) ConsumptionHeatmap(ctx context.Context, userID uuid.UUID, from, to time.Time, filter core.InsightsFilter) ([]core.HeatmapCell, error) {
	rows, err := s.q.ConsumptionHeatmap(ctx, database.ConsumptionHeatmapParams{
		UserID:       uuidToPgx(userID),
		ConsumedAt:   timestamptz(from),
		ConsumedAt_2: timestamptz(to),
		Platform:     textPtr(filter.Platform),
		MediaType:    textPtr(filter.MediaType),
	})
	if err != nil {
		return nil, fmt.Errorf("querying consumption heatmap: %w", err)
	}

	cells := make([]core.HeatmapCell, len(rows))
	for i, row := range rows {
		cells[i] = core.HeatmapCell{
			DayOfWeek: int(row.DayOfWeek),
			HourOfDay: int(row.HourOfDay),
			Count:     row.Count,
		}
	}
	return cells, nil
}

func (s *PgInsightsStore) Crossover(ctx context.Context, userID uuid.UUID, from, to time.Time, limit int32, category *string, filter core.InsightsFilter) ([]core.CrossoverEntry, error) {
	rows, err := s.q.CrossPlatformCrossover(ctx, database.CrossPlatformCrossoverParams{
		UserID:       uuidToPgx(userID),
		ConsumedAt:   timestamptz(from),
		ConsumedAt_2: timestamptz(to),
		Limit:        limit,
		Platform:     textPtr(filter.Platform),
		MediaType:    textPtr(filter.MediaType),
		Category:     textPtr(category),
	})
	if err != nil {
		return nil, fmt.Errorf("querying crossover: %w", err)
	}

	entries := make([]core.CrossoverEntry, len(rows))
	for i, row := range rows {
		cat := ""
		if row.Category.Valid {
			cat = row.Category.String
		}
		// Convert platforms from interface{} (pgx returns []string for text[])
		var platforms []string
		if ps, ok := row.Platforms.([]interface{}); ok {
			for _, p := range ps {
				if s, ok := p.(string); ok {
					platforms = append(platforms, s)
				}
			}
		}
		entries[i] = core.CrossoverEntry{
			Name:          row.Name,
			Category:      cat,
			PlatformCount: row.PlatformCount,
			ItemCount:     row.ItemCount,
			Platforms:     platforms,
		}
	}
	return entries, nil
}

func (s *PgInsightsStore) TopicTimeSeries(ctx context.Context, userID uuid.UUID, from, to time.Time, tagName, category *string, filter core.InsightsFilter) ([]core.TopicTimeSeriesEntry, error) {
	rows, err := s.q.TopicTimeSeries(ctx, database.TopicTimeSeriesParams{
		UserID:       uuidToPgx(userID),
		ConsumedAt:   timestamptz(from),
		ConsumedAt_2: timestamptz(to),
		Platform:     textPtr(filter.Platform),
		MediaType:    textPtr(filter.MediaType),
		Category:     textPtr(category),
		TagName:      textPtr(tagName),
	})
	if err != nil {
		return nil, fmt.Errorf("querying topic time series: %w", err)
	}

	entries := make([]core.TopicTimeSeriesEntry, len(rows))
	for i, row := range rows {
		var t time.Time
		if row.WeekStart.Valid {
			t = row.WeekStart.Time
		}
		entries[i] = core.TopicTimeSeriesEntry{
			WeekStart: t,
			TagName:   row.TagName,
			Count:     row.Count,
		}
	}
	return entries, nil
}

func (s *PgInsightsStore) TopicSpikes(ctx context.Context, userID uuid.UUID, from, to time.Time, recentStart time.Time, limit int32, filter core.InsightsFilter) ([]core.TopicSpikeEntry, error) {
	rows, err := s.q.TopicSpikes(ctx, database.TopicSpikesParams{
		UserID:       uuidToPgx(userID),
		ConsumedAt:   timestamptz(from),
		ConsumedAt_2: timestamptz(to),
		Limit:        limit,
		RecentStart:  timestamptz(recentStart),
		Platform:     textPtr(filter.Platform),
		MediaType:    textPtr(filter.MediaType),
	})
	if err != nil {
		return nil, fmt.Errorf("querying topic spikes: %w", err)
	}

	entries := make([]core.TopicSpikeEntry, len(rows))
	for i, row := range rows {
		cat := ""
		if row.Category.Valid {
			cat = row.Category.String
		}
		entries[i] = core.TopicSpikeEntry{
			Name:          row.Name,
			Category:      cat,
			RecentCount:   row.RecentCount,
			TotalCount:    row.TotalCount,
			PlatformCount: row.PlatformCount,
		}
	}
	return entries, nil
}
