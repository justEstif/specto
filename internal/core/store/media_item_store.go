package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/justestif/specto/internal/core"
	"github.com/justestif/specto/internal/database"
)

// PgMediaItemStore implements MediaItemStore using sqlc-generated queries.
type PgMediaItemStore struct {
	q Querier
}

// NewMediaItemStore creates a new MediaItemStore backed by PostgreSQL.
func NewMediaItemStore(q Querier) *PgMediaItemStore {
	return &PgMediaItemStore{q: q}
}

var _ core.MediaItemStore = (*PgMediaItemStore)(nil)

func (s *PgMediaItemStore) Create(ctx context.Context, userID uuid.UUID, item core.MediaItem) (uuid.UUID, error) {
	var rawMeta []byte
	if item.RawMetadata != nil {
		var err error
		rawMeta, err = json.Marshal(item.RawMetadata)
		if err != nil {
			return uuid.Nil, fmt.Errorf("marshaling raw metadata: %w", err)
		}
	}

	creator := pgtype.Text{}
	if item.Creator != "" {
		creator = pgtype.Text{String: item.Creator, Valid: true}
	}

	url := pgtype.Text{}
	if item.URL != "" {
		url = pgtype.Text{String: item.URL, Valid: true}
	}

	row, err := s.q.CreateMediaItem(ctx, database.CreateMediaItemParams{
		UserID:      uuidToPgx(userID),
		Platform:    item.Platform,
		Type:        string(item.Type),
		Title:       item.Title,
		Creator:     creator,
		ConsumedAt:  timestamptz(item.ConsumedAt),
		Duration:    durationToInterval(item.Duration),
		TimeSpent:   durationToInterval(item.TimeSpent),
		Url:         url,
		ExternalID:  item.ExternalID,
		RawMetadata: rawMeta,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("creating media item: %w", err)
	}

	return pgxToUUID(row.ID), nil
}

func (s *PgMediaItemStore) GetByExternalID(ctx context.Context, userID uuid.UUID, platform, externalID string) (*core.MediaItem, uuid.UUID, error) {
	row, err := s.q.GetMediaItemByExternalID(ctx, database.GetMediaItemByExternalIDParams{
		UserID:     uuidToPgx(userID),
		Platform:   platform,
		ExternalID: externalID,
	})
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("getting media item by external ID: %w", err)
	}

	item := mediaItemFromDB(row)
	return &item, pgxToUUID(row.ID), nil
}

func (s *PgMediaItemStore) Get(ctx context.Context, userID, itemID uuid.UUID) (*core.MediaItem, error) {
	row, err := s.q.GetMediaItemByID(ctx, database.GetMediaItemByIDParams{
		ID:     uuidToPgx(itemID),
		UserID: uuidToPgx(userID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting media item: %w", err)
	}

	item := mediaItemFromDB(row)
	return &item, nil
}

func (s *PgMediaItemStore) List(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int32) ([]core.MediaItem, error) {
	rows, err := s.q.ListMediaItems(ctx, database.ListMediaItemsParams{
		UserID:       uuidToPgx(userID),
		ConsumedAt:   timestamptz(from),
		ConsumedAt_2: timestamptz(to),
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		return nil, fmt.Errorf("listing media items: %w", err)
	}

	items := make([]core.MediaItem, len(rows))
	for i, row := range rows {
		items[i] = mediaItemFromDB(row)
	}
	return items, nil
}

func (s *PgMediaItemStore) ListFiltered(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int32, platform, mediaType, search *string) ([]core.MediaItem, error) {
	rows, err := s.q.ListMediaItemsFiltered(ctx, database.ListMediaItemsFilteredParams{
		UserID:       uuidToPgx(userID),
		ConsumedAt:   timestamptz(from),
		ConsumedAt_2: timestamptz(to),
		Limit:        limit,
		Offset:       offset,
		Platform:     textPtr(platform),
		MediaType:    textPtr(mediaType),
		Search:       textPtr(search),
	})
	if err != nil {
		return nil, fmt.Errorf("listing filtered media items: %w", err)
	}

	items := make([]core.MediaItem, len(rows))
	for i, row := range rows {
		items[i] = mediaItemFromDB(row)
	}
	return items, nil
}

func (s *PgMediaItemStore) UpdateEnrichmentStatus(ctx context.Context, itemID uuid.UUID, status string) error {
	err := s.q.UpdateEnrichmentStatus(ctx, database.UpdateEnrichmentStatusParams{
		ID:               uuidToPgx(itemID),
		EnrichmentStatus: status,
	})
	if err != nil {
		return fmt.Errorf("updating enrichment status: %w", err)
	}
	return nil
}

func (s *PgMediaItemStore) ListPendingEnrichment(ctx context.Context, limit int32) ([]core.MediaItem, error) {
	rows, err := s.q.ListPendingEnrichment(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("listing pending enrichment: %w", err)
	}

	items := make([]core.MediaItem, len(rows))
	for i, row := range rows {
		items[i] = mediaItemFromDB(row)
	}
	return items, nil
}

func (s *PgMediaItemStore) UpdateEnrichmentStatusWithRetries(ctx context.Context, itemID uuid.UUID, status string, retries int32) error {
	err := s.q.UpdateEnrichmentStatusWithRetries(ctx, database.UpdateEnrichmentStatusWithRetriesParams{
		ID:                uuidToPgx(itemID),
		EnrichmentStatus:  status,
		EnrichmentRetries: retries,
	})
	if err != nil {
		return fmt.Errorf("updating enrichment status with retries: %w", err)
	}
	return nil
}

func (s *PgMediaItemStore) ClaimPendingItems(ctx context.Context, limit int32, maxRetries int32) ([]core.EnrichmentItem, error) {
	rows, err := s.q.ClaimPendingItems(ctx, database.ClaimPendingItemsParams{
		Limit:             limit,
		EnrichmentRetries: maxRetries,
	})
	if err != nil {
		return nil, fmt.Errorf("claiming pending items: %w", err)
	}

	items := make([]core.EnrichmentItem, len(rows))
	for i, row := range rows {
		items[i] = core.EnrichmentItem{
			ID:      pgxToUUID(row.ID),
			UserID:  pgxToUUID(row.UserID),
			Item:    mediaItemFromDB(row),
			Retries: row.EnrichmentRetries,
		}
	}
	return items, nil
}

func (s *PgMediaItemStore) DeleteByPlatform(ctx context.Context, userID uuid.UUID, platform string) (int64, error) {
	count, err := s.q.DeleteMediaItemsByPlatform(ctx, database.DeleteMediaItemsByPlatformParams{
		UserID:   uuidToPgx(userID),
		Platform: platform,
	})
	if err != nil {
		return 0, fmt.Errorf("deleting media items for platform %s: %w", platform, err)
	}
	return count, nil
}
