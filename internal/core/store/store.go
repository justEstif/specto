// Package store implements the repository layer that sits between the core
// domain logic and the database. It handles credential encryption, model
// conversion, and transactional boundaries.
//
// All store implementations wrap the sqlc-generated database.Queries and
// convert between core domain types (internal/core) and database models
// (internal/database).
package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/justestif/specto/internal/core"
)

// MediaItemStore manages media item persistence.
type MediaItemStore interface {
	// Create inserts a media item, performing an upsert on
	// (user_id, platform, external_id) to handle deduplication.
	Create(ctx context.Context, userID uuid.UUID, item core.MediaItem) (uuid.UUID, error)

	// Get retrieves a single media item by ID for a user.
	Get(ctx context.Context, userID, itemID uuid.UUID) (*core.MediaItem, error)

	// List returns media items for a user within the given time range.
	List(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int32) ([]core.MediaItem, error)

	// UpdateEnrichmentStatus sets the enrichment status on a media item.
	UpdateEnrichmentStatus(ctx context.Context, itemID uuid.UUID, status string) error

	// ListPendingEnrichment returns items that still need enrichment.
	ListPendingEnrichment(ctx context.Context, limit int32) ([]core.MediaItem, error)
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
	GetCredentials(ctx context.Context, userID uuid.UUID, plugin string) (*core.Credentials, error)

	// UpsertCredentials encrypts and stores credentials for a plugin.
	UpsertCredentials(ctx context.Context, userID uuid.UUID, plugin string, authType core.AuthType, creds core.Credentials, expiresAt *time.Time) error

	// DeleteCredentials removes credentials for a plugin.
	DeleteCredentials(ctx context.Context, userID uuid.UUID, plugin string) error
}

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
