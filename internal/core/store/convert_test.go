package store

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/justestif/specto/internal/core"
	"github.com/justestif/specto/internal/database"
)

func TestUUIDToPgx(t *testing.T) {
	id := uuid.New()
	pgxID := uuidToPgx(id)

	if !pgxID.Valid {
		t.Fatal("expected Valid=true")
	}
	if pgxID.Bytes != [16]byte(id) {
		t.Fatalf("expected bytes to match: got %v, want %v", pgxID.Bytes, id)
	}
}

func TestPgxToUUID(t *testing.T) {
	id := uuid.New()
	pgxID := pgtype.UUID{Bytes: id, Valid: true}
	got := pgxToUUID(pgxID)
	if got != id {
		t.Fatalf("expected %v, got %v", id, got)
	}
}

func TestPgxToUUIDInvalid(t *testing.T) {
	pgxID := pgtype.UUID{}
	got := pgxToUUID(pgxID)
	if got != uuid.Nil {
		t.Fatalf("expected uuid.Nil, got %v", got)
	}
}

func TestUUIDRoundTrip(t *testing.T) {
	id := uuid.New()
	roundTripped := pgxToUUID(uuidToPgx(id))
	if roundTripped != id {
		t.Fatalf("round-trip failed: got %v, want %v", roundTripped, id)
	}
}

func TestTextPtr(t *testing.T) {
	s := "hello"
	txt := textPtr(&s)
	if !txt.Valid || txt.String != "hello" {
		t.Fatalf("expected valid text 'hello', got %+v", txt)
	}

	txt = textPtr(nil)
	if txt.Valid {
		t.Fatal("expected invalid text for nil input")
	}
}

func TestPtrFromText(t *testing.T) {
	txt := pgtype.Text{String: "world", Valid: true}
	s := ptrFromText(txt)
	if s == nil || *s != "world" {
		t.Fatalf("expected 'world', got %v", s)
	}

	txt = pgtype.Text{}
	s = ptrFromText(txt)
	if s != nil {
		t.Fatalf("expected nil, got %v", s)
	}
}

func TestTimestamptz(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Microsecond)
	ts := timestamptz(now)
	if !ts.Valid || !ts.Time.Equal(now) {
		t.Fatalf("expected valid timestamp %v, got %+v", now, ts)
	}
}

func TestTimestamptzPtr(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Microsecond)
	ts := timestamptzPtr(&now)
	if !ts.Valid || !ts.Time.Equal(now) {
		t.Fatalf("expected valid timestamp, got %+v", ts)
	}

	ts = timestamptzPtr(nil)
	if ts.Valid {
		t.Fatal("expected invalid timestamp for nil")
	}
}

func TestPtrFromTimestamptz(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Microsecond)
	ts := pgtype.Timestamptz{Time: now, Valid: true}
	got := ptrFromTimestamptz(ts)
	if got == nil || !got.Equal(now) {
		t.Fatalf("expected %v, got %v", now, got)
	}

	got = ptrFromTimestamptz(pgtype.Timestamptz{})
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestInt4(t *testing.T) {
	n := int4(42)
	if !n.Valid || n.Int32 != 42 {
		t.Fatalf("expected valid int4(42), got %+v", n)
	}
}

func TestInt4Val(t *testing.T) {
	if got := int4Val(pgtype.Int4{Int32: 7, Valid: true}); got != 7 {
		t.Fatalf("expected 7, got %d", got)
	}
	if got := int4Val(pgtype.Int4{}); got != 0 {
		t.Fatalf("expected 0 for invalid, got %d", got)
	}
}

func TestDurationToInterval(t *testing.T) {
	d := 5*time.Minute + 30*time.Second
	iv := durationToInterval(&d)
	if !iv.Valid {
		t.Fatal("expected valid interval")
	}
	if iv.Microseconds != d.Microseconds() {
		t.Fatalf("expected %d microseconds, got %d", d.Microseconds(), iv.Microseconds)
	}

	iv = durationToInterval(nil)
	if iv.Valid {
		t.Fatal("expected invalid interval for nil duration")
	}
}

func TestIntervalToDuration(t *testing.T) {
	us := int64(330 * time.Second / time.Microsecond)
	iv := pgtype.Interval{Microseconds: us, Valid: true}
	d := intervalToDuration(iv)
	if d == nil {
		t.Fatal("expected non-nil duration")
	}
	if *d != 330*time.Second {
		t.Fatalf("expected %v, got %v", 330*time.Second, *d)
	}

	d = intervalToDuration(pgtype.Interval{})
	if d != nil {
		t.Fatalf("expected nil for invalid interval, got %v", d)
	}
}

func TestIntervalToDurationWithDays(t *testing.T) {
	iv := pgtype.Interval{Days: 2, Microseconds: 3600000000, Valid: true}
	d := intervalToDuration(iv)
	if d == nil {
		t.Fatal("expected non-nil")
	}
	expected := 2*24*time.Hour + time.Hour
	if *d != expected {
		t.Fatalf("expected %v, got %v", expected, *d)
	}
}

func TestDurationRoundTrip(t *testing.T) {
	original := 2*time.Hour + 15*time.Minute + 30*time.Second
	roundTripped := intervalToDuration(durationToInterval(&original))
	if roundTripped == nil || *roundTripped != original {
		t.Fatalf("round-trip failed: got %v, want %v", roundTripped, original)
	}
}

func TestMediaItemFromDB(t *testing.T) {
	meta := map[string]any{"track_id": "abc123"}
	metaJSON, _ := json.Marshal(meta)

	dur := 3 * time.Minute
	dbItem := database.MediaItem{
		ID:               uuidToPgx(uuid.New()),
		UserID:           uuidToPgx(uuid.New()),
		Platform:         "spotify",
		Type:             "music",
		Title:            "Test Song",
		Creator:          pgtype.Text{String: "Test Artist", Valid: true},
		ConsumedAt:       pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		Duration:         durationToInterval(&dur),
		TimeSpent:        pgtype.Interval{},
		Url:              pgtype.Text{String: "https://spotify.com/track/abc", Valid: true},
		ExternalID:       "abc123",
		EnrichmentStatus: "pending",
		RawMetadata:      metaJSON,
		CreatedAt:        pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		UpdatedAt:        pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}

	item := mediaItemFromDB(dbItem)

	if item.Platform != "spotify" {
		t.Errorf("Platform: want 'spotify', got %q", item.Platform)
	}
	if item.Type != core.MediaMusic {
		t.Errorf("Type: want %q, got %q", core.MediaMusic, item.Type)
	}
	if item.Title != "Test Song" {
		t.Errorf("Title: want 'Test Song', got %q", item.Title)
	}
	if item.Creator != "Test Artist" {
		t.Errorf("Creator: want 'Test Artist', got %q", item.Creator)
	}
	if item.ExternalID != "abc123" {
		t.Errorf("ExternalID: want 'abc123', got %q", item.ExternalID)
	}
	if item.URL != "https://spotify.com/track/abc" {
		t.Errorf("URL: want 'https://spotify.com/track/abc', got %q", item.URL)
	}
	if item.Duration == nil || *item.Duration != 3*time.Minute {
		t.Errorf("Duration: want 3m, got %v", item.Duration)
	}
	if item.TimeSpent != nil {
		t.Errorf("TimeSpent: want nil, got %v", item.TimeSpent)
	}
	if item.RawMetadata == nil || item.RawMetadata["track_id"] != "abc123" {
		t.Errorf("RawMetadata: want {track_id: abc123}, got %v", item.RawMetadata)
	}
}

func TestMediaItemFromDB_NullFields(t *testing.T) {
	dbItem := database.MediaItem{
		Platform:   "netflix",
		Type:       "video",
		Title:      "Test Movie",
		ExternalID: "movie-1",
	}

	item := mediaItemFromDB(dbItem)

	if item.Creator != "" {
		t.Errorf("Creator: want empty, got %q", item.Creator)
	}
	if item.URL != "" {
		t.Errorf("URL: want empty, got %q", item.URL)
	}
	if item.Duration != nil {
		t.Errorf("Duration: want nil, got %v", item.Duration)
	}
	if item.RawMetadata != nil {
		t.Errorf("RawMetadata: want nil, got %v", item.RawMetadata)
	}
	if item.ConsumedAt != (time.Time{}) {
		t.Errorf("ConsumedAt: want zero, got %v", item.ConsumedAt)
	}
}

func TestPluginStateFromDB(t *testing.T) {
	now := time.Now().UTC()
	id := uuid.New()
	userID := uuid.New()

	dbPS := database.PluginState{
		ID:           uuidToPgx(id),
		UserID:       uuidToPgx(userID),
		Plugin:       "spotify",
		Status:       "connected",
		Enabled:      true,
		Cursor:       pgtype.Text{String: "cursor-123", Valid: true},
		LastSyncedAt: pgtype.Timestamptz{Time: now, Valid: true},
		ErrorMessage: pgtype.Text{},
		CreatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
	}

	info := pluginStateFromDB(dbPS)

	if info.ID != id {
		t.Errorf("ID: want %v, got %v", id, info.ID)
	}
	if info.Plugin != "spotify" {
		t.Errorf("Plugin: want 'spotify', got %q", info.Plugin)
	}
	if !info.Enabled {
		t.Error("Enabled: want true")
	}
	if info.Cursor == nil || *info.Cursor != "cursor-123" {
		t.Errorf("Cursor: want 'cursor-123', got %v", info.Cursor)
	}
	if info.ErrorMessage != nil {
		t.Errorf("ErrorMessage: want nil, got %v", info.ErrorMessage)
	}
}

func TestSyncLogFromDB(t *testing.T) {
	now := time.Now().UTC()
	id := uuid.New()

	dbSL := database.SyncLog{
		ID:           uuidToPgx(id),
		UserID:       uuidToPgx(uuid.New()),
		Plugin:       "youtube",
		StartedAt:    pgtype.Timestamptz{Time: now, Valid: true},
		CompletedAt:  pgtype.Timestamptz{Time: now.Add(time.Minute), Valid: true},
		ItemsAdded:   pgtype.Int4{Int32: 10, Valid: true},
		ItemsSkipped: pgtype.Int4{Int32: 2, Valid: true},
		ItemsUpdated: pgtype.Int4{Int32: 1, Valid: true},
		Status:       "completed",
		ErrorCode:    pgtype.Text{},
		ErrorMessage: pgtype.Text{},
		DurationMs:   pgtype.Int4{Int32: 60000, Valid: true},
	}

	entry := syncLogFromDB(dbSL)

	if entry.ID != id {
		t.Errorf("ID: want %v, got %v", id, entry.ID)
	}
	if entry.ItemsAdded != 10 {
		t.Errorf("ItemsAdded: want 10, got %d", entry.ItemsAdded)
	}
	if entry.ItemsSkipped != 2 {
		t.Errorf("ItemsSkipped: want 2, got %d", entry.ItemsSkipped)
	}
	if entry.CompletedAt == nil {
		t.Error("CompletedAt: want non-nil")
	}
	if entry.ErrorCode != nil {
		t.Errorf("ErrorCode: want nil, got %v", entry.ErrorCode)
	}
}

func TestUserFromDB(t *testing.T) {
	now := time.Now().UTC()
	id := uuid.New()

	dbUser := database.User{
		ID:           uuidToPgx(id),
		Email:        "test@example.com",
		DisplayName:  "Test User",
		AvatarUrl:    pgtype.Text{String: "https://example.com/avatar.png", Valid: true},
		AuthProvider: "email",
		AuthSubject:  "test@example.com",
		ProfileSlug:  pgtype.Text{String: "test-user", Valid: true},
		PasswordHash: pgtype.Text{String: "$2a$10$hash", Valid: true},
		CreatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
	}

	info := userFromDB(dbUser)

	if info.ID != id {
		t.Errorf("ID: want %v, got %v", id, info.ID)
	}
	if info.Email != "test@example.com" {
		t.Errorf("Email: want 'test@example.com', got %q", info.Email)
	}
	if info.AvatarURL == nil || *info.AvatarURL != "https://example.com/avatar.png" {
		t.Errorf("AvatarURL: want url, got %v", info.AvatarURL)
	}
	if info.ProfileSlug == nil || *info.ProfileSlug != "test-user" {
		t.Errorf("ProfileSlug: want 'test-user', got %v", info.ProfileSlug)
	}
	if info.PasswordHash == nil || *info.PasswordHash != "$2a$10$hash" {
		t.Errorf("PasswordHash: want hash, got %v", info.PasswordHash)
	}
}

func TestUserFromDB_NullOptionals(t *testing.T) {
	dbUser := database.User{
		ID:           uuidToPgx(uuid.New()),
		Email:        "user@example.com",
		DisplayName:  "User",
		AuthProvider: "github",
		AuthSubject:  "12345",
		CreatedAt:    pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		UpdatedAt:    pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}

	info := userFromDB(dbUser)

	if info.AvatarURL != nil {
		t.Errorf("AvatarURL: want nil, got %v", info.AvatarURL)
	}
	if info.ProfileSlug != nil {
		t.Errorf("ProfileSlug: want nil, got %v", info.ProfileSlug)
	}
	if info.PasswordHash != nil {
		t.Errorf("PasswordHash: want nil, got %v", info.PasswordHash)
	}
}
