package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/justestif/specto/internal/core"
	"github.com/justestif/specto/internal/database"
)

// EraStore implements core.EraStore using sqlc-generated queries.
type EraStore struct {
	q Querier
}

// NewEraStore creates a new EraStore.
func NewEraStore(q Querier) *EraStore {
	return &EraStore{q: q}
}

// Compile-time assertion.
var _ core.EraStore = (*EraStore)(nil)

func (s *EraStore) Create(ctx context.Context, era core.Era) (*core.Era, error) {
	row, err := s.q.CreateEra(ctx, database.CreateEraParams{
		UserID:          uuidToPgx(era.UserID),
		MediaType:       textPtr(era.MediaType),
		SuggestedTitle:  textPtr(era.SuggestedTitle),
		StartedAt:       timestamptz(era.StartedAt),
		EndedAt:         timestamptzPtr(era.EndedAt),
		ItemCount:       era.ItemCount,
		Distinctiveness: era.Distinctiveness,
		Status:          era.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("creating era: %w", err)
	}
	return eraFromDB(row), nil
}

func (s *EraStore) List(ctx context.Context, userID uuid.UUID, mediaType *string) ([]core.Era, error) {
	rows, err := s.q.ListEras(ctx, database.ListErasParams{
		UserID:    uuidToPgx(userID),
		MediaType: textPtr(mediaType),
	})
	if err != nil {
		return nil, fmt.Errorf("listing eras: %w", err)
	}

	eras := make([]core.Era, len(rows))
	for i, row := range rows {
		eras[i] = *eraFromDB(row)
	}
	return eras, nil
}

func (s *EraStore) Get(ctx context.Context, userID, eraID uuid.UUID) (*core.Era, error) {
	row, err := s.q.GetEra(ctx, database.GetEraParams{
		ID:     uuidToPgx(eraID),
		UserID: uuidToPgx(userID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting era: %w", err)
	}
	return eraFromDB(row), nil
}

func (s *EraStore) UpdateTitle(ctx context.Context, eraID uuid.UUID, title string) (*core.Era, error) {
	row, err := s.q.UpdateEraTitle(ctx, database.UpdateEraTitleParams{
		ID:    uuidToPgx(eraID),
		Title: pgText(title),
	})
	if err != nil {
		return nil, fmt.Errorf("updating era title: %w", err)
	}
	return eraFromDB(row), nil
}

func (s *EraStore) UpdateSuggestedTitle(ctx context.Context, eraID uuid.UUID, title string) (*core.Era, error) {
	row, err := s.q.UpdateEraSuggestedTitle(ctx, database.UpdateEraSuggestedTitleParams{
		ID:             uuidToPgx(eraID),
		SuggestedTitle: pgText(title),
	})
	if err != nil {
		return nil, fmt.Errorf("updating era suggested title: %w", err)
	}
	return eraFromDB(row), nil
}

func (s *EraStore) Dismiss(ctx context.Context, userID, eraID uuid.UUID) error {
	err := s.q.DismissEra(ctx, database.DismissEraParams{
		ID:     uuidToPgx(eraID),
		UserID: uuidToPgx(userID),
	})
	if err != nil {
		return fmt.Errorf("dismissing era: %w", err)
	}
	return nil
}

func (s *EraStore) DeleteSuggested(ctx context.Context, userID uuid.UUID, mediaType string) error {
	err := s.q.DeleteErasByUserAndType(ctx, database.DeleteErasByUserAndTypeParams{
		UserID:    uuidToPgx(userID),
		MediaType: pgText(mediaType),
	})
	if err != nil {
		return fmt.Errorf("deleting suggested eras: %w", err)
	}
	return nil
}

func (s *EraStore) UpsertTag(ctx context.Context, eraID, tagID uuid.UUID, weight float32) error {
	err := s.q.UpsertEraTag(ctx, database.UpsertEraTagParams{
		EraID:  uuidToPgx(eraID),
		TagID:  uuidToPgx(tagID),
		Weight: weight,
	})
	if err != nil {
		return fmt.Errorf("upserting era tag: %w", err)
	}
	return nil
}

func (s *EraStore) GetTags(ctx context.Context, eraID uuid.UUID) ([]core.EraTag, error) {
	rows, err := s.q.GetEraTags(ctx, uuidToPgx(eraID))
	if err != nil {
		return nil, fmt.Errorf("getting era tags: %w", err)
	}

	tags := make([]core.EraTag, len(rows))
	for i, row := range rows {
		tags[i] = core.EraTag{
			TagID:    pgxToUUID(row.TagID),
			TagName:  row.TagName,
			Category: row.TagCategory.String,
			Weight:   row.Weight,
		}
	}
	return tags, nil
}

func (s *EraStore) TagVectorByWindow(ctx context.Context, userID uuid.UUID, from, to time.Time, mediaType string) ([]core.WindowTagEntry, error) {
	rows, err := s.q.TagVectorByWindow(ctx, database.TagVectorByWindowParams{
		UserID:       uuidToPgx(userID),
		ConsumedAt:   timestamptz(from),
		ConsumedAt_2: timestamptz(to),
		Type:         mediaType,
	})
	if err != nil {
		return nil, fmt.Errorf("fetching tag vectors: %w", err)
	}

	entries := make([]core.WindowTagEntry, len(rows))
	for i, row := range rows {
		entries[i] = core.WindowTagEntry{
			WindowStart: row.WindowStart.Time,
			TagID:       pgxToUUID(row.TagID),
			TagName:     row.TagName,
			Category:    row.TagCategory.String,
			TagCount:    row.TagCount,
			WindowTotal: row.WindowTotal,
		}
	}
	return entries, nil
}

func (s *EraStore) CountItemsInRange(ctx context.Context, userID uuid.UUID, mediaType string, from, to time.Time) (int64, error) {
	count, err := s.q.CountItemsInRange(ctx, database.CountItemsInRangeParams{
		UserID:       uuidToPgx(userID),
		Type:         mediaType,
		ConsumedAt:   timestamptz(from),
		ConsumedAt_2: timestamptz(to),
	})
	if err != nil {
		return 0, fmt.Errorf("counting items in range: %w", err)
	}
	return count, nil
}

// --- conversion helpers ---

func eraFromDB(row database.Era) *core.Era {
	var endedAt *time.Time
	if row.EndedAt.Valid {
		t := row.EndedAt.Time
		endedAt = &t
	}

	return &core.Era{
		ID:              pgxToUUID(row.ID),
		UserID:          pgxToUUID(row.UserID),
		MediaType:       ptrFromText(row.MediaType),
		Title:           ptrFromText(row.Title),
		SuggestedTitle:  ptrFromText(row.SuggestedTitle),
		StartedAt:       row.StartedAt.Time,
		EndedAt:         endedAt,
		ItemCount:       row.ItemCount,
		Distinctiveness: row.Distinctiveness,
		Status:          row.Status,
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}
}
