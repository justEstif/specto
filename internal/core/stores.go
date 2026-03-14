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

	// GetByExternalID retrieves a media item by its platform-specific external ID.
	GetByExternalID(ctx context.Context, userID uuid.UUID, platform, externalID string) (*MediaItem, uuid.UUID, error)

	// List returns media items for a user within the given time range.
	List(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int32) ([]MediaItem, error)

	// ListFiltered returns media items for a user within the given time range,
	// with optional filtering by platform, media type, and title/creator search.
	// Pass nil for any filter to skip it.
	ListFiltered(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int32, platform, mediaType, search *string) ([]MediaItem, error)

	// UpdateEnrichmentStatus sets the enrichment status on a media item.
	UpdateEnrichmentStatus(ctx context.Context, itemID uuid.UUID, status string) error

	// UpdateEnrichmentStatusWithRetries sets the enrichment status and retry
	// count on a media item. Used by the enrichment worker.
	UpdateEnrichmentStatusWithRetries(ctx context.Context, itemID uuid.UUID, status string, retries int32) error

	// ListPendingEnrichment returns items that still need enrichment.
	ListPendingEnrichment(ctx context.Context, limit int32) ([]MediaItem, error)

	// ClaimPendingItems atomically selects and locks pending items for
	// enrichment processing, skipping items already locked by another
	// worker instance. Uses FOR UPDATE SKIP LOCKED.
	ClaimPendingItems(ctx context.Context, limit int32, maxRetries int32) ([]EnrichmentItem, error)

	// ResetEnrichment resets all enriched/failed items for a user back to
	// pending with zero retries. Returns the number of items reset.
	// Used for re-enrichment when tags or prompts change.
	ResetEnrichment(ctx context.Context, userID uuid.UUID) (int64, error)

	// ResetEnrichmentByID resets a single item's enrichment status to pending.
	ResetEnrichmentByID(ctx context.Context, itemID, userID uuid.UUID) error

	// EnrichmentStats returns counts of items in each enrichment status
	// for a user.
	EnrichmentStats(ctx context.Context, userID uuid.UUID) (*EnrichmentStatusCounts, error)

	// DeleteByPlatform removes all media items for a user on a given platform.
	// Returns the number of items deleted.
	DeleteByPlatform(ctx context.Context, userID uuid.UUID, platform string) (int64, error)

	// OnThisDay returns media items consumed on this day (month+day) in
	// previous years. Items are ordered by consumed_at descending.
	OnThisDay(ctx context.Context, userID uuid.UUID, limit int32) ([]OnThisDayItem, error)
}

// OnThisDayItem wraps a MediaItem with the year it was consumed.
type OnThisDayItem struct {
	Year int
	Item MediaItem
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

	// DeleteByPlugin removes all sync log entries for a user's plugin.
	DeleteByPlugin(ctx context.Context, userID uuid.UUID, plugin string) error
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

	// GetByProfileSlug retrieves a user by their public profile slug.
	GetByProfileSlug(ctx context.Context, slug string) (*UserInfo, error)

	// GetByAuth retrieves a user by auth provider and subject.
	GetByAuth(ctx context.Context, provider, subject string) (*UserInfo, error)

	// Create creates a new user with OAuth credentials.
	Create(ctx context.Context, email, displayName string, avatarURL *string, provider, subject string) (*UserInfo, error)

	// CreateWithPassword creates a new user with email/password auth.
	CreateWithPassword(ctx context.Context, email, displayName, passwordHash string) (*UserInfo, error)

	// UpdateProfile updates a user's display name, avatar, and profile slug.
	UpdateProfile(ctx context.Context, id uuid.UUID, displayName string, avatarURL, profileSlug *string) (*UserInfo, error)

	// MarkOnboarded sets the user's onboarded flag to true.
	MarkOnboarded(ctx context.Context, id uuid.UUID) error
}

// InsightsFilter holds optional filters for insights queries.
type InsightsFilter struct {
	Platform  *string
	MediaType *string
}

// InsightsStore provides pre-aggregated analytics data.
type InsightsStore interface {
	// PlatformBreakdown returns item counts and total duration grouped by
	// platform and media type for the given date range.
	PlatformBreakdown(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]PlatformBreakdownEntry, error)

	// PlatformBreakdownFiltered returns platform breakdown with optional
	// platform and media type filters applied.
	PlatformBreakdownFiltered(ctx context.Context, userID uuid.UUID, from, to time.Time, filter InsightsFilter) ([]PlatformBreakdownEntry, error)

	// TagDistribution returns tag usage counts for the given date range,
	// limited to tags with confidence >= minConfidence (or authoritative tags
	// where confidence is NULL). Results are ordered by count descending.
	TagDistribution(ctx context.Context, userID uuid.UUID, from, to time.Time, limit int32) ([]TagDistributionEntry, error)

	// TagDistributionFiltered returns tag distribution with optional
	// platform and media type filters applied.
	TagDistributionFiltered(ctx context.Context, userID uuid.UUID, from, to time.Time, limit int32, filter InsightsFilter) ([]TagDistributionEntry, error)

	// ListMediaItems returns media items for a user within a date range
	// (used internally by InsightsService for timeline aggregation).
	ListMediaItems(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int32) ([]MediaItem, error)

	// ListMediaItemsFiltered returns media items with optional filters
	// (used internally by InsightsService for filtered timeline aggregation).
	ListMediaItemsFiltered(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int32, filter InsightsFilter) ([]MediaItem, error)

	// TagDistributionByCategory returns tag distribution filtered to a
	// specific category (genre/topic/mood/format).
	TagDistributionByCategory(ctx context.Context, userID uuid.UUID, from, to time.Time, limit int32, category string, filter InsightsFilter) ([]TagDistributionEntry, error)

	// AttentionByType returns consumption counts and time breakdowns
	// grouped by media type, optionally filtered by platform.
	AttentionByType(ctx context.Context, userID uuid.UUID, from, to time.Time, platform *string) ([]AttentionByTypeEntry, error)

	// ConsumptionHeatmap returns consumption counts grouped by day-of-week
	// (0=Sun..6=Sat) and hour-of-day (0..23) for a rhythm heatmap.
	ConsumptionHeatmap(ctx context.Context, userID uuid.UUID, from, to time.Time, filter InsightsFilter) ([]HeatmapCell, error)
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

// EnrichmentStatusCounts holds counts of items in each enrichment status.
type EnrichmentStatusCounts struct {
	Pending   int64
	Enriching int64
	Enriched  int64
	Failed    int64
}

// EnrichmentItem wraps a MediaItem with its database ID and retry count,
// used by the enrichment worker to track per-item state.
type EnrichmentItem struct {
	ID      uuid.UUID
	UserID  uuid.UUID
	Item    MediaItem
	Retries int32
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
	Onboarded    bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// --- Share Profile types ---

// ShareProfile is the domain representation of a user's public profile configuration.
type ShareProfile struct {
	ID                uuid.UUID
	UserID            uuid.UUID
	Blocks            []ShareBlock
	ExcludedPlatforms []string
	ExcludedTags      []string
	Published         bool
	Slug              *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ShareBlock represents a single configurable block on the public share profile.
type ShareBlock struct {
	Type      string   `json:"type"` // "top_genres", "mood_profile", "top_creators", "platform_mix", "currently_into"
	Enabled   bool     `json:"enabled"`
	TimeRange string   `json:"time_range,omitempty"` // "7d", "30d", "90d", "all"
	Count     int      `json:"count,omitempty"`      // for top-N blocks
	Platforms []string `json:"platforms,omitempty"`  // platform filter for this block
	ItemIDs   []string `json:"item_ids,omitempty"`   // for recent_favorites
	Text      string   `json:"text,omitempty"`       // for currently_into
}

// PublicShareProfile is the public-facing resolved profile with user info.
type PublicShareProfile struct {
	DisplayName string
	AvatarURL   *string
	Slug        string
	Profile     ShareProfile
}

// ShareProfileStore manages share profile persistence.
type ShareProfileStore interface {
	// Get retrieves the share profile for a user.
	Get(ctx context.Context, userID uuid.UUID) (*ShareProfile, error)

	// GetBySlug retrieves a published share profile by its public slug,
	// including user display info. Returns nil if not found or not published.
	GetBySlug(ctx context.Context, slug string) (*PublicShareProfile, error)

	// Upsert creates or updates a share profile.
	Upsert(ctx context.Context, userID uuid.UUID, profile ShareProfile) (*ShareProfile, error)

	// SetItemPrivacy sets the private flag on a media item.
	SetItemPrivacy(ctx context.Context, userID, itemID uuid.UUID, private bool) error

	// GetPublicTagDistribution returns tag distribution for public items,
	// respecting platform and tag exclusions.
	GetPublicTagDistribution(ctx context.Context, userID uuid.UUID, from, to time.Time, limit int32, excludedPlatforms, excludedTags []string, categoryFilter *string) ([]TagDistributionEntry, error)

	// GetPublicTopCreators returns top creators for public items.
	GetPublicTopCreators(ctx context.Context, userID uuid.UUID, from, to time.Time, limit int32, excludedPlatforms []string) ([]TopCreatorEntry, error)

	// GetPublicPlatformMix returns platform consumption mix for public items.
	GetPublicPlatformMix(ctx context.Context, userID uuid.UUID, from, to time.Time, excludedPlatforms []string) ([]PlatformMixEntry, error)
}

// TopCreatorEntry represents a creator with consumption count.
type TopCreatorEntry struct {
	Creator   string
	Platform  string
	MediaType string
	Count     int64
}

// PlatformMixEntry represents a platform's share of consumption.
type PlatformMixEntry struct {
	Platform string
	Count    int64
}

// AttentionByTypeEntry represents consumption stats grouped by media type,
// including both content duration and actual time spent.
type AttentionByTypeEntry struct {
	MediaType        string
	Count            int64
	TotalTimeSpent   int64 // seconds of actual engagement
	TotalDurationSec int64 // seconds of content duration
}

// HeatmapCell represents a single cell in the day-of-week × hour-of-day
// consumption heatmap.
type HeatmapCell struct {
	DayOfWeek int // 0=Sun, 1=Mon, ..., 6=Sat
	HourOfDay int // 0..23
	Count     int64
}
