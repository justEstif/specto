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
	GetMediaItemByExternalID(ctx context.Context, arg database.GetMediaItemByExternalIDParams) (database.MediaItem, error)
	ListMediaItems(ctx context.Context, arg database.ListMediaItemsParams) ([]database.MediaItem, error)
	ListMediaItemsFiltered(ctx context.Context, arg database.ListMediaItemsFilteredParams) ([]database.MediaItem, error)
	UpdateEnrichmentStatus(ctx context.Context, arg database.UpdateEnrichmentStatusParams) error
	UpdateEnrichmentStatusWithRetries(ctx context.Context, arg database.UpdateEnrichmentStatusWithRetriesParams) error
	ListPendingEnrichment(ctx context.Context, limit int32) ([]database.MediaItem, error)
	ClaimPendingItems(ctx context.Context, arg database.ClaimPendingItemsParams) ([]database.MediaItem, error)
	ResetEnrichmentByUser(ctx context.Context, userID pgtype.UUID) (int64, error)
	ResetEnrichmentByID(ctx context.Context, arg database.ResetEnrichmentByIDParams) error
	EnrichmentStats(ctx context.Context, userID pgtype.UUID) (database.EnrichmentStatsRow, error)
	DeleteMediaItemsByPlatform(ctx context.Context, arg database.DeleteMediaItemsByPlatformParams) (int64, error)

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
	DeleteSyncLogsByPlugin(ctx context.Context, arg database.DeleteSyncLogsByPluginParams) error

	// Users
	GetUserByID(ctx context.Context, id pgtype.UUID) (database.User, error)
	GetUserByEmail(ctx context.Context, email string) (database.User, error)
	GetUserByProfileSlug(ctx context.Context, profileSlug pgtype.Text) (database.User, error)
	GetUserByAuth(ctx context.Context, arg database.GetUserByAuthParams) (database.User, error)
	CreateUser(ctx context.Context, arg database.CreateUserParams) (database.User, error)
	CreateUserWithPassword(ctx context.Context, arg database.CreateUserWithPasswordParams) (database.User, error)
	UpdateUserProfile(ctx context.Context, arg database.UpdateUserProfileParams) (database.User, error)
	MarkUserOnboarded(ctx context.Context, id pgtype.UUID) error
	ListUserIDsWithEnrichedItems(ctx context.Context) ([]pgtype.UUID, error)

	// Tags
	GetOrCreateTag(ctx context.Context, arg database.GetOrCreateTagParams) (database.Tag, error)
	GetTagByName(ctx context.Context, name string) (database.Tag, error)
	GetTagByAlias(ctx context.Context, alias string) (database.Tag, error)
	AddMediaItemTag(ctx context.Context, arg database.AddMediaItemTagParams) error
	ListMediaItemTags(ctx context.Context, mediaItemID pgtype.UUID) ([]database.ListMediaItemTagsRow, error)

	// On This Day
	OnThisDay(ctx context.Context, arg database.OnThisDayParams) ([]database.MediaItem, error)

	// Insights / analytics
	PlatformBreakdown(ctx context.Context, arg database.PlatformBreakdownParams) ([]database.PlatformBreakdownRow, error)
	PlatformBreakdownFiltered(ctx context.Context, arg database.PlatformBreakdownFilteredParams) ([]database.PlatformBreakdownFilteredRow, error)
	TagDistribution(ctx context.Context, arg database.TagDistributionParams) ([]database.TagDistributionRow, error)
	TagDistributionFiltered(ctx context.Context, arg database.TagDistributionFilteredParams) ([]database.TagDistributionFilteredRow, error)
	TagDistributionByCategory(ctx context.Context, arg database.TagDistributionByCategoryParams) ([]database.TagDistributionByCategoryRow, error)
	AttentionByType(ctx context.Context, arg database.AttentionByTypeParams) ([]database.AttentionByTypeRow, error)
	ConsumptionHeatmap(ctx context.Context, arg database.ConsumptionHeatmapParams) ([]database.ConsumptionHeatmapRow, error)
	CrossPlatformCrossover(ctx context.Context, arg database.CrossPlatformCrossoverParams) ([]database.CrossPlatformCrossoverRow, error)
	TopicTimeSeries(ctx context.Context, arg database.TopicTimeSeriesParams) ([]database.TopicTimeSeriesRow, error)
	TopicSpikes(ctx context.Context, arg database.TopicSpikesParams) ([]database.TopicSpikesRow, error)

	// Eras
	TagVectorByWindow(ctx context.Context, arg database.TagVectorByWindowParams) ([]database.TagVectorByWindowRow, error)
	CreateEra(ctx context.Context, arg database.CreateEraParams) (database.Era, error)
	UpsertEraTag(ctx context.Context, arg database.UpsertEraTagParams) error
	ListEras(ctx context.Context, arg database.ListErasParams) ([]database.Era, error)
	GetEra(ctx context.Context, arg database.GetEraParams) (database.Era, error)
	GetEraTags(ctx context.Context, eraID pgtype.UUID) ([]database.GetEraTagsRow, error)
	UpdateEraTitle(ctx context.Context, arg database.UpdateEraTitleParams) (database.Era, error)
	UpdateEraSuggestedTitle(ctx context.Context, arg database.UpdateEraSuggestedTitleParams) (database.Era, error)
	DismissEra(ctx context.Context, arg database.DismissEraParams) error
	DeleteErasByUserAndType(ctx context.Context, arg database.DeleteErasByUserAndTypeParams) error
	CountItemsInRange(ctx context.Context, arg database.CountItemsInRangeParams) (int64, error)

	// Share profiles
	GetShareProfile(ctx context.Context, userID pgtype.UUID) (database.ShareProfile, error)
	GetShareProfileBySlug(ctx context.Context, slug pgtype.Text) (database.GetShareProfileBySlugRow, error)
	UpsertShareProfile(ctx context.Context, arg database.UpsertShareProfileParams) (database.ShareProfile, error)
	SetItemPrivacy(ctx context.Context, arg database.SetItemPrivacyParams) (database.SetItemPrivacyRow, error)
	GetPublicItems(ctx context.Context, arg database.GetPublicItemsParams) ([]database.MediaItem, error)
	GetPublicTagDistribution(ctx context.Context, arg database.GetPublicTagDistributionParams) ([]database.GetPublicTagDistributionRow, error)
	GetPublicTopCreators(ctx context.Context, arg database.GetPublicTopCreatorsParams) ([]database.GetPublicTopCreatorsRow, error)
	GetPublicPlatformMix(ctx context.Context, arg database.GetPublicPlatformMixParams) ([]database.GetPublicPlatformMixRow, error)
}

// Compile-time assertion that database.Queries satisfies Querier.
var _ Querier = (*database.Queries)(nil)
