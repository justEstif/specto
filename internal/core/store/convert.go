package store

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/justestif/specto/internal/core"
	"github.com/justestif/specto/internal/database"
)

// --- UUID conversion helpers ---

// uuidToPgx converts a google/uuid.UUID to a pgtype.UUID.
func uuidToPgx(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

// pgxToUUID converts a pgtype.UUID to a google/uuid.UUID.
// Returns uuid.Nil if the pgtype.UUID is not valid.
func pgxToUUID(id pgtype.UUID) uuid.UUID {
	if !id.Valid {
		return uuid.Nil
	}
	return uuid.UUID(id.Bytes)
}

// --- pgtype helper constructors ---

// textPtr creates a pgtype.Text from a *string.
func textPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// ptrFromText converts a pgtype.Text to a *string.
func ptrFromText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}

// timestamptz creates a pgtype.Timestamptz from a time.Time.
func timestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// timestamptzPtr creates a pgtype.Timestamptz from a *time.Time.
func timestamptzPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

// ptrFromTimestamptz converts a pgtype.Timestamptz to a *time.Time.
func ptrFromTimestamptz(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	ts := t.Time
	return &ts
}

// int4 creates a pgtype.Int4 from an int32.
func int4(n int32) pgtype.Int4 {
	return pgtype.Int4{Int32: n, Valid: true}
}

// int4Val returns the int32 value from a pgtype.Int4, defaulting to 0.
func int4Val(n pgtype.Int4) int32 {
	if !n.Valid {
		return 0
	}
	return n.Int32
}

// durationToInterval converts a *time.Duration to a pgtype.Interval.
func durationToInterval(d *time.Duration) pgtype.Interval {
	if d == nil {
		return pgtype.Interval{}
	}
	return pgtype.Interval{
		Microseconds: d.Microseconds(),
		Valid:        true,
	}
}

// intervalToDuration converts a pgtype.Interval to a *time.Duration.
func intervalToDuration(i pgtype.Interval) *time.Duration {
	if !i.Valid {
		return nil
	}
	// Interval stores days and months separately; for media durations
	// we only use microseconds (no calendar math needed).
	d := time.Duration(i.Microseconds) * time.Microsecond
	// Add days as 24-hour periods (interval days field).
	d += time.Duration(i.Days) * 24 * time.Hour
	// Months are not expected for media durations, but handle gracefully.
	d += time.Duration(i.Months) * 30 * 24 * time.Hour
	return &d
}

// --- Domain <-> Database model conversion ---

// mediaItemFromDB converts a database.MediaItem to a core.MediaItem.
func mediaItemFromDB(m database.MediaItem) core.MediaItem {
	item := core.MediaItem{
		Platform:   m.Platform,
		Type:       core.MediaType(m.Type),
		Title:      m.Title,
		Creator:    ptrValFromText(m.Creator),
		ExternalID: m.ExternalID,
		URL:        ptrValFromText(m.Url),
		Duration:   intervalToDuration(m.Duration),
		TimeSpent:  intervalToDuration(m.TimeSpent),
	}

	if m.ConsumedAt.Valid {
		item.ConsumedAt = m.ConsumedAt.Time
	}

	if m.RawMetadata != nil {
		_ = json.Unmarshal(m.RawMetadata, &item.RawMetadata)
	}

	return item
}

// ptrValFromText extracts the string value from a pgtype.Text, returning ""
// if not valid.
func ptrValFromText(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

// pluginStateFromDB converts a database.PluginState to a core.PluginStateInfo.
func pluginStateFromDB(ps database.PluginState) core.PluginStateInfo {
	return core.PluginStateInfo{
		ID:           pgxToUUID(ps.ID),
		UserID:       pgxToUUID(ps.UserID),
		Plugin:       ps.Plugin,
		Status:       ps.Status,
		Enabled:      ps.Enabled,
		Cursor:       ptrFromText(ps.Cursor),
		LastSyncedAt: ptrFromTimestamptz(ps.LastSyncedAt),
		ErrorMessage: ptrFromText(ps.ErrorMessage),
		CreatedAt:    ps.CreatedAt.Time,
		UpdatedAt:    ps.UpdatedAt.Time,
	}
}

// syncLogFromDB converts a database.SyncLog to a core.SyncLogEntry.
func syncLogFromDB(sl database.SyncLog) core.SyncLogEntry {
	return core.SyncLogEntry{
		ID:           pgxToUUID(sl.ID),
		UserID:       pgxToUUID(sl.UserID),
		Plugin:       sl.Plugin,
		StartedAt:    sl.StartedAt.Time,
		CompletedAt:  ptrFromTimestamptz(sl.CompletedAt),
		ItemsAdded:   int4Val(sl.ItemsAdded),
		ItemsSkipped: int4Val(sl.ItemsSkipped),
		ItemsUpdated: int4Val(sl.ItemsUpdated),
		Status:       sl.Status,
		ErrorCode:    ptrFromText(sl.ErrorCode),
		ErrorMessage: ptrFromText(sl.ErrorMessage),
		DurationMs:   int4Val(sl.DurationMs),
	}
}

// userFromDB converts a database.User to a core.UserInfo.
func userFromDB(u database.User) core.UserInfo {
	return core.UserInfo{
		ID:           pgxToUUID(u.ID),
		Email:        u.Email,
		DisplayName:  u.DisplayName,
		AvatarURL:    ptrFromText(u.AvatarUrl),
		AuthProvider: u.AuthProvider,
		AuthSubject:  u.AuthSubject,
		ProfileSlug:  ptrFromText(u.ProfileSlug),
		PasswordHash: ptrFromText(u.PasswordHash),
		CreatedAt:    u.CreatedAt.Time,
		UpdatedAt:    u.UpdatedAt.Time,
	}
}
