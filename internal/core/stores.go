package core

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// --- Store interfaces ---
// These interfaces define the contracts that the store layer implements.
// They live in the core package to avoid import cycles: the store package
// imports core for domain types, and the syncer (also in core) depends
// on these interfaces without importing the store package.

// MediaItemStore manages media item persistence.
type MediaItemStore interface {
	// Create inserts a media item, performing an upsert on
	// (user_id, platform, external_id) to handle deduplication.
	Create(ctx context.Context, userID uuid.UUID, item MediaItem) (uuid.UUID, error)

	// Get retrieves a single media item by ID for a user.
	Get(ctx context.Context, userID, itemID uuid.UUID) (*MediaItem, error)

	// List returns media items for a user within the given time range.
	List(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int32) ([]MediaItem, error)

	// UpdateEnrichmentStatus sets the enrichment status on a media item.
	UpdateEnrichmentStatus(ctx context.Context, itemID uuid.UUID, status string) error

	// ListPendingEnrichment returns items that still need enrichment.
	ListPendingEnrichment(ctx context.Context, limit int32) ([]MediaItem, error)
}

// PluginStateStore manages plugin connection state and credentials.
type PluginStateStore interface {
	// GetState retrieves the state of a plugin for a user.
	GetState(ctx context.Context, userID uuid.UUID, plugin string) (*PluginStateInfo, error)

	// UpsertState creates or updates the state of a plugin for a user.
	UpsertState(ctx context.Context, userID uuid.UUID, plugin, status string, enabled bool) (*PluginStateInfo, error)

	// UpdateStatus updates the status and error message for a plugin.
	UpdateStatus(ctx context.Context, userID uuid.UUID, plugin, status string, errMsg *string) (*PluginStateInfo, error)

	// UpdateSynced marks a plugin as successfully synced with a new cursor.
	UpdateSynced(ctx context.Context, userID uuid.UUID, plugin string, cursor *string) (*PluginStateInfo, error)

	// ListStates returns all plugin states for a user.
	ListStates(ctx context.Context, userID uuid.UUID) ([]PluginStateInfo, error)

	// GetCredentials retrieves and decrypts credentials for a plugin.
	GetCredentials(ctx context.Context, userID uuid.UUID, plugin string) (*Credentials, error)

	// UpsertCredentials encrypts and stores credentials for a plugin.
	UpsertCredentials(ctx context.Context, userID uuid.UUID, plugin string, authType AuthType, creds Credentials, expiresAt *time.Time) error

	// DeleteCredentials removes credentials for a plugin.
	DeleteCredentials(ctx context.Context, userID uuid.UUID, plugin string) error
}

// SyncLogStore manages sync log entries.
type SyncLogStore interface {
	// Begin creates a new sync log entry with status "running".
	Begin(ctx context.Context, userID uuid.UUID, plugin string) (uuid.UUID, error)

	// Complete marks a sync log as completed with result counts.
	Complete(ctx context.Context, logID uuid.UUID, result SyncLogResult) error

	// Fail marks a sync log as failed with an error.
	Fail(ctx context.Context, logID uuid.UUID, result SyncLogResult) error

	// List returns recent sync logs for a plugin.
	List(ctx context.Context, userID uuid.UUID, plugin string, limit int32) ([]SyncLogEntry, error)
}

// TagStore manages tag persistence, alias resolution, and media item tagging.
type TagStore interface {
	// ResolveTag resolves a tag string to a canonical tag UUID.
	// It first checks if the tag exists by name in the fixed set,
	// then falls back to alias lookup. Returns the tag UUID and the
	// canonical tag name, or an error if the tag is unknown.
	ResolveTag(ctx context.Context, tag string) (uuid.UUID, string, error)

	// GetOrCreate ensures a tag exists in the database and returns its UUID.
	// The tag must be in the fixed tag set. The category is looked up automatically.
	GetOrCreate(ctx context.Context, tag string) (uuid.UUID, error)

	// AddMediaItemTag associates a tag with a media item.
	// source is "plugin" or "llm". confidence is nil for authoritative (plugin) tags.
	AddMediaItemTag(ctx context.Context, itemID, tagID uuid.UUID, source string, confidence *float32) error

	// ListMediaItemTags returns all tags for a media item.
	ListMediaItemTags(ctx context.Context, itemID uuid.UUID) ([]MediaItemTagInfo, error)
}

// UserStore manages user persistence.
type UserStore interface {
	// GetByID retrieves a user by their ID.
	GetByID(ctx context.Context, id uuid.UUID) (*UserInfo, error)

	// GetByEmail retrieves a user by email.
	GetByEmail(ctx context.Context, email string) (*UserInfo, error)

	// GetByAuth retrieves a user by auth provider and subject.
	GetByAuth(ctx context.Context, provider, subject string) (*UserInfo, error)

	// Create creates a new user with OAuth credentials.
	Create(ctx context.Context, email, displayName string, avatarURL *string, provider, subject string) (*UserInfo, error)

	// CreateWithPassword creates a new user with email/password auth.
	CreateWithPassword(ctx context.Context, email, displayName, passwordHash string) (*UserInfo, error)

	// UpdateProfile updates a user's display name, avatar, and profile slug.
	UpdateProfile(ctx context.Context, id uuid.UUID, displayName string, avatarURL, profileSlug *string) (*UserInfo, error)
}

// InsightsStore provides pre-aggregated analytics data.
type InsightsStore interface {
	// PlatformBreakdown returns item counts and total duration grouped by
	// platform and media type for the given date range.
	PlatformBreakdown(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]PlatformBreakdownEntry, error)

	// TagDistribution returns tag usage counts for the given date range,
	// limited to tags with confidence >= minConfidence (or authoritative tags
	// where confidence is NULL). Results are ordered by count descending.
	TagDistribution(ctx context.Context, userID uuid.UUID, from, to time.Time, limit int32) ([]TagDistributionEntry, error)

	// ListMediaItems returns media items for a user within a date range
	// (used internally by InsightsService for timeline aggregation).
	ListMediaItems(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int32) ([]MediaItem, error)
}

// --- Domain types used by store interfaces ---

// PlatformBreakdownEntry represents consumption stats for a single
// platform + media type combination.
type PlatformBreakdownEntry struct {
	Platform         string
	MediaType        string
	Count            int64
	TotalDurationSec int64
}

// TagDistributionEntry represents how often a tag appears across
// media items within a date range.
type TagDistributionEntry struct {
	Name     string
	Category string
	Count    int64
}

// Summary is the top-level insights overview for a user.
type Summary struct {
	TotalItems       int64
	TotalDurationSec int64
	TopPlatform      string
	TopMediaType     string
}

// TimelineEntry represents aggregated consumption data for a single
// time bucket (day, week, or month).
type TimelineEntry struct {
	Bucket           time.Time
	Count            int64
	TotalDurationSec int64
}

// TimeBucket defines the granularity for timeline aggregation.
type TimeBucket string

const (
	BucketDay   TimeBucket = "day"
	BucketWeek  TimeBucket = "week"
	BucketMonth TimeBucket = "month"
)

// PluginStateInfo is the domain representation of a plugin's state.
type PluginStateInfo struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Plugin       string
	Status       string
	Enabled      bool
	Cursor       *string
	LastSyncedAt *time.Time
	ErrorMessage *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// SyncLogResult holds the outcome counters for a completed or failed sync.
type SyncLogResult struct {
	ItemsAdded   int32
	ItemsSkipped int32
	ItemsUpdated int32
	ErrorCode    *string
	ErrorMessage *string
	DurationMs   int32
}

// SyncLogEntry is the domain representation of a sync log row.
type SyncLogEntry struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Plugin       string
	StartedAt    time.Time
	CompletedAt  *time.Time
	ItemsAdded   int32
	ItemsSkipped int32
	ItemsUpdated int32
	Status       string
	ErrorCode    *string
	ErrorMessage *string
	DurationMs   int32
}

// MediaItemTagInfo is the domain representation of a tag attached to a media item.
type MediaItemTagInfo struct {
	Name       string
	Category   string
	Source     string
	Confidence *float32
}

// UserInfo is the domain representation of a user.
type UserInfo struct {
	ID           uuid.UUID
	Email        string
	DisplayName  string
	AvatarURL    *string
	AuthProvider string
	AuthSubject  string
	ProfileSlug  *string
	PasswordHash *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
