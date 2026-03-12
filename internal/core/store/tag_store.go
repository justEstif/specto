package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/justestif/specto/internal/core"
	"github.com/justestif/specto/internal/database"
)

// PgTagStore implements core.TagStore using sqlc-generated queries.
type PgTagStore struct {
	q Querier
}

// NewTagStore creates a new TagStore backed by PostgreSQL.
func NewTagStore(q Querier) *PgTagStore {
	return &PgTagStore{q: q}
}

var _ core.TagStore = (*PgTagStore)(nil)

func (s *PgTagStore) ResolveTag(ctx context.Context, tag string) (uuid.UUID, string, error) {
	normalized := strings.ToLower(strings.TrimSpace(tag))

	// First, check if it's a valid tag in the fixed set.
	if core.IsValidTag(normalized) {
		id, err := s.GetOrCreate(ctx, normalized)
		if err != nil {
			return uuid.Nil, "", err
		}
		return id, normalized, nil
	}

	// Fall back to alias lookup.
	dbTag, err := s.q.GetTagByAlias(ctx, normalized)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("resolving tag %q: not in fixed set and no alias found: %w", tag, err)
	}

	return pgxToUUID(dbTag.ID), dbTag.Name, nil
}

func (s *PgTagStore) GetOrCreate(ctx context.Context, tag string) (uuid.UUID, error) {
	if !core.IsValidTag(tag) {
		return uuid.Nil, fmt.Errorf("tag %q is not in the fixed tag set", tag)
	}

	category := core.TagCategoryOf(tag)
	dbTag, err := s.q.GetOrCreateTag(ctx, database.GetOrCreateTagParams{
		Name:     tag,
		Category: pgtype.Text{String: string(category), Valid: true},
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("get-or-creating tag %q: %w", tag, err)
	}

	return pgxToUUID(dbTag.ID), nil
}

func (s *PgTagStore) AddMediaItemTag(ctx context.Context, itemID, tagID uuid.UUID, source string, confidence *float32) error {
	conf := pgtype.Float4{}
	if confidence != nil {
		conf = pgtype.Float4{Float32: *confidence, Valid: true}
	}

	err := s.q.AddMediaItemTag(ctx, database.AddMediaItemTagParams{
		MediaItemID: uuidToPgx(itemID),
		TagID:       uuidToPgx(tagID),
		Source:      source,
		Confidence:  conf,
	})
	if err != nil {
		return fmt.Errorf("adding media item tag: %w", err)
	}
	return nil
}

func (s *PgTagStore) ListMediaItemTags(ctx context.Context, itemID uuid.UUID) ([]core.MediaItemTagInfo, error) {
	rows, err := s.q.ListMediaItemTags(ctx, uuidToPgx(itemID))
	if err != nil {
		return nil, fmt.Errorf("listing media item tags: %w", err)
	}

	tags := make([]core.MediaItemTagInfo, len(rows))
	for i, row := range rows {
		var conf *float32
		if row.Confidence.Valid {
			c := row.Confidence.Float32
			conf = &c
		}
		tags[i] = core.MediaItemTagInfo{
			Name:       row.Name,
			Category:   ptrValFromText(row.Category),
			Source:     row.Source,
			Confidence: conf,
		}
	}
	return tags, nil
}
