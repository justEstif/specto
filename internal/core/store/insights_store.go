package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
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

func (s *PgInsightsStore) PlatformBreakdown(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]core.PlatformBreakdownEntry, error) {
	rows, err := s.q.PlatformBreakdown(ctx, database.PlatformBreakdownParams{
		UserID:       uuidToPgx(userID),
		ConsumedAt:   timestamptz(from),
		ConsumedAt_2: timestamptz(to),
	})
	if err != nil {
		return nil, fmt.Errorf("querying platform breakdown: %w", err)
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

func (s *PgInsightsStore) TagDistribution(ctx context.Context, userID uuid.UUID, from, to time.Time, limit int32) ([]core.TagDistributionEntry, error) {
	rows, err := s.q.TagDistribution(ctx, database.TagDistributionParams{
		UserID:       uuidToPgx(userID),
		ConsumedAt:   timestamptz(from),
		ConsumedAt_2: timestamptz(to),
		Limit:        limit,
	})
	if err != nil {
		return nil, fmt.Errorf("querying tag distribution: %w", err)
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

func (s *PgInsightsStore) ListMediaItems(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int32) ([]core.MediaItem, error) {
	rows, err := s.q.ListMediaItems(ctx, database.ListMediaItemsParams{
		UserID:       uuidToPgx(userID),
		ConsumedAt:   timestamptz(from),
		ConsumedAt_2: timestamptz(to),
		Limit:        limit,
		Offset:       offset,
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
