package store

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/justestif/specto/internal/database"
)

func TestTagStore_GetOrCreate(t *testing.T) {
	tagID := uuid.New()

	mock := &mockQuerier{
		getOrCreateTagFn: func(_ context.Context, arg database.GetOrCreateTagParams) (database.Tag, error) {
			if arg.Name != "rock" {
				t.Errorf("Name: want 'rock', got %q", arg.Name)
			}
			if !arg.Category.Valid || arg.Category.String != "genre" {
				t.Errorf("Category: want 'genre', got %+v", arg.Category)
			}
			return database.Tag{
				ID:       uuidToPgx(tagID),
				Name:     "rock",
				Category: pgtype.Text{String: "genre", Valid: true},
			}, nil
		},
	}

	store := NewTagStore(mock)
	id, err := store.GetOrCreate(context.Background(), "rock")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != tagID {
		t.Fatalf("expected tag ID %v, got %v", tagID, id)
	}
}

func TestTagStore_GetOrCreate_InvalidTag(t *testing.T) {
	mock := &mockQuerier{}
	store := NewTagStore(mock)

	_, err := store.GetOrCreate(context.Background(), "vaporwave")
	if err == nil {
		t.Fatal("expected error for invalid tag")
	}
}

func TestTagStore_ResolveTag_DirectMatch(t *testing.T) {
	tagID := uuid.New()

	mock := &mockQuerier{
		getOrCreateTagFn: func(_ context.Context, arg database.GetOrCreateTagParams) (database.Tag, error) {
			return database.Tag{
				ID:       uuidToPgx(tagID),
				Name:     arg.Name,
				Category: arg.Category,
			}, nil
		},
	}

	store := NewTagStore(mock)
	id, name, err := store.ResolveTag(context.Background(), "electronic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != tagID {
		t.Errorf("ID: want %v, got %v", tagID, id)
	}
	if name != "electronic" {
		t.Errorf("Name: want 'electronic', got %q", name)
	}
}

func TestTagStore_ResolveTag_NormalizesCase(t *testing.T) {
	tagID := uuid.New()

	mock := &mockQuerier{
		getOrCreateTagFn: func(_ context.Context, arg database.GetOrCreateTagParams) (database.Tag, error) {
			if arg.Name != "rock" {
				t.Errorf("expected normalized 'rock', got %q", arg.Name)
			}
			return database.Tag{ID: uuidToPgx(tagID), Name: "rock"}, nil
		},
	}

	store := NewTagStore(mock)
	// "Rock" should normalize to "rock" which is in the fixed set
	// Note: "Rock" != "rock" so IsValidTag("Rock") is false, falls to alias
	// But " rock " with whitespace should normalize to "rock"
	id, name, err := store.ResolveTag(context.Background(), " rock ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != tagID {
		t.Errorf("ID mismatch")
	}
	if name != "rock" {
		t.Errorf("Name: want 'rock', got %q", name)
	}
}

func TestTagStore_ResolveTag_AliasLookup(t *testing.T) {
	tagID := uuid.New()

	mock := &mockQuerier{
		getTagByAliasFn: func(_ context.Context, alias string) (database.Tag, error) {
			if alias != "hip hop" {
				t.Errorf("Alias: want 'hip hop', got %q", alias)
			}
			return database.Tag{
				ID:       uuidToPgx(tagID),
				Name:     "hip-hop",
				Category: pgtype.Text{String: "genre", Valid: true},
			}, nil
		},
	}

	store := NewTagStore(mock)
	id, name, err := store.ResolveTag(context.Background(), "Hip Hop")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != tagID {
		t.Errorf("ID: want %v, got %v", tagID, id)
	}
	if name != "hip-hop" {
		t.Errorf("Name: want 'hip-hop', got %q", name)
	}
}

func TestTagStore_ResolveTag_Unknown(t *testing.T) {
	mock := &mockQuerier{
		getTagByAliasFn: func(_ context.Context, alias string) (database.Tag, error) {
			return database.Tag{}, fmt.Errorf("no rows")
		},
	}

	store := NewTagStore(mock)
	_, _, err := store.ResolveTag(context.Background(), "completely-unknown-tag")
	if err == nil {
		t.Fatal("expected error for unknown tag")
	}
}

func TestTagStore_AddMediaItemTag(t *testing.T) {
	itemID := uuid.New()
	tagID := uuid.New()
	conf := float32(0.85)

	mock := &mockQuerier{
		addMediaItemTagFn: func(_ context.Context, arg database.AddMediaItemTagParams) error {
			if pgxToUUID(arg.MediaItemID) != itemID {
				t.Errorf("MediaItemID mismatch")
			}
			if pgxToUUID(arg.TagID) != tagID {
				t.Errorf("TagID mismatch")
			}
			if arg.Source != "llm" {
				t.Errorf("Source: want 'llm', got %q", arg.Source)
			}
			if !arg.Confidence.Valid || arg.Confidence.Float32 != 0.85 {
				t.Errorf("Confidence: want 0.85, got %+v", arg.Confidence)
			}
			return nil
		},
	}

	store := NewTagStore(mock)
	err := store.AddMediaItemTag(context.Background(), itemID, tagID, "llm", &conf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTagStore_AddMediaItemTag_NilConfidence(t *testing.T) {
	mock := &mockQuerier{
		addMediaItemTagFn: func(_ context.Context, arg database.AddMediaItemTagParams) error {
			if arg.Confidence.Valid {
				t.Error("Confidence: expected invalid for nil (authoritative tag)")
			}
			return nil
		},
	}

	store := NewTagStore(mock)
	err := store.AddMediaItemTag(context.Background(), uuid.New(), uuid.New(), "plugin", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTagStore_ListMediaItemTags(t *testing.T) {
	itemID := uuid.New()
	conf := float32(0.8)

	mock := &mockQuerier{
		listMediaItemTagsFn: func(_ context.Context, mid pgtype.UUID) ([]database.ListMediaItemTagsRow, error) {
			return []database.ListMediaItemTagsRow{
				{
					Name:       "rock",
					Category:   pgtype.Text{String: "genre", Valid: true},
					Source:     "plugin",
					Confidence: pgtype.Float4{},
				},
				{
					Name:       "melancholic",
					Category:   pgtype.Text{String: "mood", Valid: true},
					Source:     "llm",
					Confidence: pgtype.Float4{Float32: conf, Valid: true},
				},
			}, nil
		},
	}

	store := NewTagStore(mock)
	tags, err := store.ListMediaItemTags(context.Background(), itemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(tags))
	}

	if tags[0].Name != "rock" || tags[0].Category != "genre" || tags[0].Source != "plugin" {
		t.Errorf("first tag: got %+v", tags[0])
	}
	if tags[0].Confidence != nil {
		t.Errorf("first tag confidence: want nil (authoritative), got %v", tags[0].Confidence)
	}

	if tags[1].Name != "melancholic" || tags[1].Source != "llm" {
		t.Errorf("second tag: got %+v", tags[1])
	}
	if tags[1].Confidence == nil || *tags[1].Confidence != 0.8 {
		t.Errorf("second tag confidence: want 0.8, got %v", tags[1].Confidence)
	}
}
