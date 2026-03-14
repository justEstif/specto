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

// PgShareProfileStore implements core.ShareProfileStore using sqlc-generated queries.
type PgShareProfileStore struct {
	q Querier
}

// NewShareProfileStore creates a new ShareProfileStore backed by PostgreSQL.
func NewShareProfileStore(q Querier) *PgShareProfileStore {
	return &PgShareProfileStore{q: q}
}

var _ core.ShareProfileStore = (*PgShareProfileStore)(nil)

func (s *PgShareProfileStore) Get(ctx context.Context, userID uuid.UUID) (*core.ShareProfile, error) {
	row, err := s.q.GetShareProfile(ctx, uuidToPgx(userID))
	if err != nil {
		return nil, fmt.Errorf("getting share profile: %w", err)
	}

	profile, err := shareProfileFromDB(row)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func (s *PgShareProfileStore) GetBySlug(ctx context.Context, slug string) (*core.PublicShareProfile, error) {
	row, err := s.q.GetShareProfileBySlug(ctx, pgText(slug))
	if err != nil {
		return nil, fmt.Errorf("getting share profile by slug: %w", err)
	}

	var blocks []core.ShareBlock
	if len(row.Blocks) > 0 {
		if err := json.Unmarshal(row.Blocks, &blocks); err != nil {
			return nil, fmt.Errorf("unmarshaling blocks: %w", err)
		}
	}

	return &core.PublicShareProfile{
		DisplayName: row.DisplayName,
		AvatarURL:   ptrFromText(row.AvatarUrl),
		Slug:        ptrValFromText(row.Slug),
		Profile: core.ShareProfile{
			ID:                pgxToUUID(row.ID),
			UserID:            pgxToUUID(row.UserID),
			Blocks:            blocks,
			ExcludedPlatforms: row.ExcludedPlatforms,
			ExcludedTags:      row.ExcludedTags,
			Published:         row.Published,
			Slug:              ptrFromText(row.Slug),
			CreatedAt:         row.CreatedAt.Time,
			UpdatedAt:         row.UpdatedAt.Time,
		},
	}, nil
}

func (s *PgShareProfileStore) Upsert(ctx context.Context, userID uuid.UUID, profile core.ShareProfile) (*core.ShareProfile, error) {
	blocksJSON, err := json.Marshal(profile.Blocks)
	if err != nil {
		return nil, fmt.Errorf("marshaling blocks: %w", err)
	}

	// Ensure non-nil slices for PostgreSQL arrays.
	excludedPlatforms := profile.ExcludedPlatforms
	if excludedPlatforms == nil {
		excludedPlatforms = []string{}
	}
	excludedTags := profile.ExcludedTags
	if excludedTags == nil {
		excludedTags = []string{}
	}

	row, err := s.q.UpsertShareProfile(ctx, database.UpsertShareProfileParams{
		UserID:            uuidToPgx(userID),
		Blocks:            blocksJSON,
		ExcludedPlatforms: excludedPlatforms,
		ExcludedTags:      excludedTags,
		Published:         profile.Published,
		Slug:              textPtr(profile.Slug),
	})
	if err != nil {
		return nil, fmt.Errorf("upserting share profile: %w", err)
	}

	result, err := shareProfileFromDB(row)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *PgShareProfileStore) SetItemPrivacy(ctx context.Context, userID, itemID uuid.UUID, private bool) error {
	_, err := s.q.SetItemPrivacy(ctx, database.SetItemPrivacyParams{
		ID:      uuidToPgx(itemID),
		UserID:  uuidToPgx(userID),
		Private: private,
	})
	if err != nil {
		return fmt.Errorf("setting item privacy: %w", err)
	}
	return nil
}

func (s *PgShareProfileStore) GetPublicTagDistribution(ctx context.Context, userID uuid.UUID, from, to time.Time, limit int32, excludedPlatforms, excludedTags []string, categoryFilter *string) ([]core.TagDistributionEntry, error) {
	if excludedPlatforms == nil {
		excludedPlatforms = []string{}
	}
	if excludedTags == nil {
		excludedTags = []string{}
	}

	rows, err := s.q.GetPublicTagDistribution(ctx, database.GetPublicTagDistributionParams{
		UserID:         uuidToPgx(userID),
		ConsumedAt:     timestamptz(from),
		ConsumedAt_2:   timestamptz(to),
		Limit:          limit,
		Column5:        excludedPlatforms,
		Column6:        excludedTags,
		CategoryFilter: textPtr(categoryFilter),
	})
	if err != nil {
		return nil, fmt.Errorf("querying public tag distribution: %w", err)
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

func (s *PgShareProfileStore) GetPublicTopCreators(ctx context.Context, userID uuid.UUID, from, to time.Time, limit int32, excludedPlatforms []string) ([]core.TopCreatorEntry, error) {
	if excludedPlatforms == nil {
		excludedPlatforms = []string{}
	}

	rows, err := s.q.GetPublicTopCreators(ctx, database.GetPublicTopCreatorsParams{
		UserID:       uuidToPgx(userID),
		ConsumedAt:   timestamptz(from),
		ConsumedAt_2: timestamptz(to),
		Limit:        limit,
		Column5:      excludedPlatforms,
	})
	if err != nil {
		return nil, fmt.Errorf("querying public top creators: %w", err)
	}

	entries := make([]core.TopCreatorEntry, len(rows))
	for i, row := range rows {
		entries[i] = core.TopCreatorEntry{
			Creator:   ptrValFromText(row.Creator),
			Platform:  row.Platform,
			MediaType: row.Type,
			Count:     row.Count,
		}
	}
	return entries, nil
}

func (s *PgShareProfileStore) GetPublicPlatformMix(ctx context.Context, userID uuid.UUID, from, to time.Time, excludedPlatforms []string) ([]core.PlatformMixEntry, error) {
	if excludedPlatforms == nil {
		excludedPlatforms = []string{}
	}

	rows, err := s.q.GetPublicPlatformMix(ctx, database.GetPublicPlatformMixParams{
		UserID:       uuidToPgx(userID),
		ConsumedAt:   timestamptz(from),
		ConsumedAt_2: timestamptz(to),
		Column4:      excludedPlatforms,
	})
	if err != nil {
		return nil, fmt.Errorf("querying public platform mix: %w", err)
	}

	entries := make([]core.PlatformMixEntry, len(rows))
	for i, row := range rows {
		entries[i] = core.PlatformMixEntry{
			Platform: row.Platform,
			Count:    row.Count,
		}
	}
	return entries, nil
}

// --- Helpers ---

// pgText creates a pgtype.Text from a string.
func pgText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

// shareProfileFromDB converts a database.ShareProfile to a core.ShareProfile.
func shareProfileFromDB(sp database.ShareProfile) (*core.ShareProfile, error) {
	var blocks []core.ShareBlock
	if len(sp.Blocks) > 0 {
		if err := json.Unmarshal(sp.Blocks, &blocks); err != nil {
			return nil, fmt.Errorf("unmarshaling blocks: %w", err)
		}
	}

	return &core.ShareProfile{
		ID:                pgxToUUID(sp.ID),
		UserID:            pgxToUUID(sp.UserID),
		Blocks:            blocks,
		ExcludedPlatforms: sp.ExcludedPlatforms,
		ExcludedTags:      sp.ExcludedTags,
		Published:         sp.Published,
		Slug:              ptrFromText(sp.Slug),
		CreatedAt:         sp.CreatedAt.Time,
		UpdatedAt:         sp.UpdatedAt.Time,
	}, nil
}
