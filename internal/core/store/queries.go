package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/justestif/specto/internal/database"
)

// Querier defines the subset of database.Queries methods used by the store
// layer. This allows store implementations to be tested with mocks.
type Querier interface {
	// Media items
	CreateMediaItem(ctx context.Context, arg database.CreateMediaItemParams) (database.MediaItem, error)
	GetMediaItemByID(ctx context.Context, arg database.GetMediaItemByIDParams) (database.MediaItem, error)
	ListMediaItems(ctx context.Context, arg database.ListMediaItemsParams) ([]database.MediaItem, error)
	UpdateEnrichmentStatus(ctx context.Context, arg database.UpdateEnrichmentStatusParams) error
	ListPendingEnrichment(ctx context.Context, limit int32) ([]database.MediaItem, error)

	// Plugin state
	GetPluginState(ctx context.Context, arg database.GetPluginStateParams) (database.PluginState, error)
	UpsertPluginState(ctx context.Context, arg database.UpsertPluginStateParams) (database.PluginState, error)
	UpdatePluginStateStatus(ctx context.Context, arg database.UpdatePluginStateStatusParams) (database.PluginState, error)
	UpdatePluginStateSynced(ctx context.Context, arg database.UpdatePluginStateSyncedParams) (database.PluginState, error)
	ListPluginStates(ctx context.Context, userID pgtype.UUID) ([]database.PluginState, error)

	// Plugin credentials
	GetPluginCredentials(ctx context.Context, arg database.GetPluginCredentialsParams) (database.PluginCredential, error)
	UpsertPluginCredentials(ctx context.Context, arg database.UpsertPluginCredentialsParams) (database.PluginCredential, error)
	DeletePluginCredentials(ctx context.Context, arg database.DeletePluginCredentialsParams) error

	// Sync log
	CreateSyncLog(ctx context.Context, arg database.CreateSyncLogParams) (database.SyncLog, error)
	CompleteSyncLog(ctx context.Context, arg database.CompleteSyncLogParams) (database.SyncLog, error)
	ListSyncLogs(ctx context.Context, arg database.ListSyncLogsParams) ([]database.SyncLog, error)

	// Users
	GetUserByID(ctx context.Context, id pgtype.UUID) (database.User, error)
	GetUserByEmail(ctx context.Context, email string) (database.User, error)
	GetUserByAuth(ctx context.Context, arg database.GetUserByAuthParams) (database.User, error)
	CreateUser(ctx context.Context, arg database.CreateUserParams) (database.User, error)
	CreateUserWithPassword(ctx context.Context, arg database.CreateUserWithPasswordParams) (database.User, error)
	UpdateUserProfile(ctx context.Context, arg database.UpdateUserProfileParams) (database.User, error)

	// Tags
	GetOrCreateTag(ctx context.Context, arg database.GetOrCreateTagParams) (database.Tag, error)
	GetTagByName(ctx context.Context, name string) (database.Tag, error)
	GetTagByAlias(ctx context.Context, alias string) (database.Tag, error)
	AddMediaItemTag(ctx context.Context, arg database.AddMediaItemTagParams) error
	ListMediaItemTags(ctx context.Context, mediaItemID pgtype.UUID) ([]database.ListMediaItemTagsRow, error)

	// Insights / analytics
	PlatformBreakdown(ctx context.Context, arg database.PlatformBreakdownParams) ([]database.PlatformBreakdownRow, error)
	TagDistribution(ctx context.Context, arg database.TagDistributionParams) ([]database.TagDistributionRow, error)
}

// Compile-time assertion that database.Queries satisfies Querier.
var _ Querier = (*database.Queries)(nil)
