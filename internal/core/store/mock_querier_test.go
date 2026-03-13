package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/justestif/specto/internal/database"
)

// mockQuerier is a configurable mock of the Querier interface for unit tests.
// Each method field, when set, overrides the default behavior.
type mockQuerier struct {
	createMediaItemFn          func(ctx context.Context, arg database.CreateMediaItemParams) (database.MediaItem, error)
	getMediaItemByIDFn         func(ctx context.Context, arg database.GetMediaItemByIDParams) (database.MediaItem, error)
	getMediaItemByExternalIDFn func(ctx context.Context, arg database.GetMediaItemByExternalIDParams) (database.MediaItem, error)
	listMediaItemsFn           func(ctx context.Context, arg database.ListMediaItemsParams) ([]database.MediaItem, error)
	updateEnrichmentStatusFn   func(ctx context.Context, arg database.UpdateEnrichmentStatusParams) error
	listPendingEnrichmentFn    func(ctx context.Context, limit int32) ([]database.MediaItem, error)
	getPluginStateFn           func(ctx context.Context, arg database.GetPluginStateParams) (database.PluginState, error)
	upsertPluginStateFn        func(ctx context.Context, arg database.UpsertPluginStateParams) (database.PluginState, error)
	updatePluginStateStatusFn  func(ctx context.Context, arg database.UpdatePluginStateStatusParams) (database.PluginState, error)
	updatePluginStateSyncedFn  func(ctx context.Context, arg database.UpdatePluginStateSyncedParams) (database.PluginState, error)
	listPluginStatesFn         func(ctx context.Context, userID pgtype.UUID) ([]database.PluginState, error)
	getPluginCredentialsFn     func(ctx context.Context, arg database.GetPluginCredentialsParams) (database.PluginCredential, error)
	upsertPluginCredentialsFn  func(ctx context.Context, arg database.UpsertPluginCredentialsParams) (database.PluginCredential, error)
	deletePluginCredentialsFn  func(ctx context.Context, arg database.DeletePluginCredentialsParams) error
	createSyncLogFn            func(ctx context.Context, arg database.CreateSyncLogParams) (database.SyncLog, error)
	completeSyncLogFn          func(ctx context.Context, arg database.CompleteSyncLogParams) (database.SyncLog, error)
	listSyncLogsFn             func(ctx context.Context, arg database.ListSyncLogsParams) ([]database.SyncLog, error)
	getUserByIDFn              func(ctx context.Context, id pgtype.UUID) (database.User, error)
	getUserByEmailFn           func(ctx context.Context, email string) (database.User, error)
	getUserByProfileSlugFn     func(ctx context.Context, profileSlug pgtype.Text) (database.User, error)
	getUserByAuthFn            func(ctx context.Context, arg database.GetUserByAuthParams) (database.User, error)
	createUserFn               func(ctx context.Context, arg database.CreateUserParams) (database.User, error)
	createUserWithPasswordFn   func(ctx context.Context, arg database.CreateUserWithPasswordParams) (database.User, error)
	updateUserProfileFn        func(ctx context.Context, arg database.UpdateUserProfileParams) (database.User, error)
	getOrCreateTagFn           func(ctx context.Context, arg database.GetOrCreateTagParams) (database.Tag, error)
	getTagByNameFn             func(ctx context.Context, name string) (database.Tag, error)
	getTagByAliasFn            func(ctx context.Context, alias string) (database.Tag, error)
	addMediaItemTagFn          func(ctx context.Context, arg database.AddMediaItemTagParams) error
	listMediaItemTagsFn        func(ctx context.Context, mediaItemID pgtype.UUID) ([]database.ListMediaItemTagsRow, error)
	platformBreakdownFn        func(ctx context.Context, arg database.PlatformBreakdownParams) ([]database.PlatformBreakdownRow, error)
	tagDistributionFn          func(ctx context.Context, arg database.TagDistributionParams) ([]database.TagDistributionRow, error)
}

var _ Querier = (*mockQuerier)(nil)

func (m *mockQuerier) CreateMediaItem(ctx context.Context, arg database.CreateMediaItemParams) (database.MediaItem, error) {
	if m.createMediaItemFn != nil {
		return m.createMediaItemFn(ctx, arg)
	}
	return database.MediaItem{}, fmt.Errorf("CreateMediaItem not mocked")
}

func (m *mockQuerier) GetMediaItemByID(ctx context.Context, arg database.GetMediaItemByIDParams) (database.MediaItem, error) {
	if m.getMediaItemByIDFn != nil {
		return m.getMediaItemByIDFn(ctx, arg)
	}
	return database.MediaItem{}, fmt.Errorf("GetMediaItemByID not mocked")
}

func (m *mockQuerier) GetMediaItemByExternalID(ctx context.Context, arg database.GetMediaItemByExternalIDParams) (database.MediaItem, error) {
	if m.getMediaItemByExternalIDFn != nil {
		return m.getMediaItemByExternalIDFn(ctx, arg)
	}
	return database.MediaItem{}, fmt.Errorf("GetMediaItemByExternalID not mocked")
}

func (m *mockQuerier) ListMediaItems(ctx context.Context, arg database.ListMediaItemsParams) ([]database.MediaItem, error) {
	if m.listMediaItemsFn != nil {
		return m.listMediaItemsFn(ctx, arg)
	}
	return nil, fmt.Errorf("ListMediaItems not mocked")
}

func (m *mockQuerier) UpdateEnrichmentStatus(ctx context.Context, arg database.UpdateEnrichmentStatusParams) error {
	if m.updateEnrichmentStatusFn != nil {
		return m.updateEnrichmentStatusFn(ctx, arg)
	}
	return fmt.Errorf("UpdateEnrichmentStatus not mocked")
}

func (m *mockQuerier) ListPendingEnrichment(ctx context.Context, limit int32) ([]database.MediaItem, error) {
	if m.listPendingEnrichmentFn != nil {
		return m.listPendingEnrichmentFn(ctx, limit)
	}
	return nil, fmt.Errorf("ListPendingEnrichment not mocked")
}

func (m *mockQuerier) GetPluginState(ctx context.Context, arg database.GetPluginStateParams) (database.PluginState, error) {
	if m.getPluginStateFn != nil {
		return m.getPluginStateFn(ctx, arg)
	}
	return database.PluginState{}, fmt.Errorf("GetPluginState not mocked")
}

func (m *mockQuerier) UpsertPluginState(ctx context.Context, arg database.UpsertPluginStateParams) (database.PluginState, error) {
	if m.upsertPluginStateFn != nil {
		return m.upsertPluginStateFn(ctx, arg)
	}
	return database.PluginState{}, fmt.Errorf("UpsertPluginState not mocked")
}

func (m *mockQuerier) UpdatePluginStateStatus(ctx context.Context, arg database.UpdatePluginStateStatusParams) (database.PluginState, error) {
	if m.updatePluginStateStatusFn != nil {
		return m.updatePluginStateStatusFn(ctx, arg)
	}
	return database.PluginState{}, fmt.Errorf("UpdatePluginStateStatus not mocked")
}

func (m *mockQuerier) UpdatePluginStateSynced(ctx context.Context, arg database.UpdatePluginStateSyncedParams) (database.PluginState, error) {
	if m.updatePluginStateSyncedFn != nil {
		return m.updatePluginStateSyncedFn(ctx, arg)
	}
	return database.PluginState{}, fmt.Errorf("UpdatePluginStateSynced not mocked")
}

func (m *mockQuerier) ListPluginStates(ctx context.Context, userID pgtype.UUID) ([]database.PluginState, error) {
	if m.listPluginStatesFn != nil {
		return m.listPluginStatesFn(ctx, userID)
	}
	return nil, fmt.Errorf("ListPluginStates not mocked")
}

func (m *mockQuerier) GetPluginCredentials(ctx context.Context, arg database.GetPluginCredentialsParams) (database.PluginCredential, error) {
	if m.getPluginCredentialsFn != nil {
		return m.getPluginCredentialsFn(ctx, arg)
	}
	return database.PluginCredential{}, fmt.Errorf("GetPluginCredentials not mocked")
}

func (m *mockQuerier) UpsertPluginCredentials(ctx context.Context, arg database.UpsertPluginCredentialsParams) (database.PluginCredential, error) {
	if m.upsertPluginCredentialsFn != nil {
		return m.upsertPluginCredentialsFn(ctx, arg)
	}
	return database.PluginCredential{}, fmt.Errorf("UpsertPluginCredentials not mocked")
}

func (m *mockQuerier) DeletePluginCredentials(ctx context.Context, arg database.DeletePluginCredentialsParams) error {
	if m.deletePluginCredentialsFn != nil {
		return m.deletePluginCredentialsFn(ctx, arg)
	}
	return fmt.Errorf("DeletePluginCredentials not mocked")
}

func (m *mockQuerier) CreateSyncLog(ctx context.Context, arg database.CreateSyncLogParams) (database.SyncLog, error) {
	if m.createSyncLogFn != nil {
		return m.createSyncLogFn(ctx, arg)
	}
	return database.SyncLog{}, fmt.Errorf("CreateSyncLog not mocked")
}

func (m *mockQuerier) CompleteSyncLog(ctx context.Context, arg database.CompleteSyncLogParams) (database.SyncLog, error) {
	if m.completeSyncLogFn != nil {
		return m.completeSyncLogFn(ctx, arg)
	}
	return database.SyncLog{}, fmt.Errorf("CompleteSyncLog not mocked")
}

func (m *mockQuerier) ListSyncLogs(ctx context.Context, arg database.ListSyncLogsParams) ([]database.SyncLog, error) {
	if m.listSyncLogsFn != nil {
		return m.listSyncLogsFn(ctx, arg)
	}
	return nil, fmt.Errorf("ListSyncLogs not mocked")
}

func (m *mockQuerier) GetUserByID(ctx context.Context, id pgtype.UUID) (database.User, error) {
	if m.getUserByIDFn != nil {
		return m.getUserByIDFn(ctx, id)
	}
	return database.User{}, fmt.Errorf("GetUserByID not mocked")
}

func (m *mockQuerier) GetUserByProfileSlug(ctx context.Context, profileSlug pgtype.Text) (database.User, error) {
	if m.getUserByProfileSlugFn != nil {
		return m.getUserByProfileSlugFn(ctx, profileSlug)
	}
	return database.User{}, fmt.Errorf("GetUserByProfileSlug not mocked")
}

func (m *mockQuerier) GetUserByEmail(ctx context.Context, email string) (database.User, error) {
	if m.getUserByEmailFn != nil {
		return m.getUserByEmailFn(ctx, email)
	}
	return database.User{}, fmt.Errorf("GetUserByEmail not mocked")
}

func (m *mockQuerier) GetUserByAuth(ctx context.Context, arg database.GetUserByAuthParams) (database.User, error) {
	if m.getUserByAuthFn != nil {
		return m.getUserByAuthFn(ctx, arg)
	}
	return database.User{}, fmt.Errorf("GetUserByAuth not mocked")
}

func (m *mockQuerier) CreateUser(ctx context.Context, arg database.CreateUserParams) (database.User, error) {
	if m.createUserFn != nil {
		return m.createUserFn(ctx, arg)
	}
	return database.User{}, fmt.Errorf("CreateUser not mocked")
}

func (m *mockQuerier) CreateUserWithPassword(ctx context.Context, arg database.CreateUserWithPasswordParams) (database.User, error) {
	if m.createUserWithPasswordFn != nil {
		return m.createUserWithPasswordFn(ctx, arg)
	}
	return database.User{}, fmt.Errorf("CreateUserWithPassword not mocked")
}

func (m *mockQuerier) UpdateUserProfile(ctx context.Context, arg database.UpdateUserProfileParams) (database.User, error) {
	if m.updateUserProfileFn != nil {
		return m.updateUserProfileFn(ctx, arg)
	}
	return database.User{}, fmt.Errorf("UpdateUserProfile not mocked")
}

func (m *mockQuerier) GetOrCreateTag(ctx context.Context, arg database.GetOrCreateTagParams) (database.Tag, error) {
	if m.getOrCreateTagFn != nil {
		return m.getOrCreateTagFn(ctx, arg)
	}
	return database.Tag{}, fmt.Errorf("GetOrCreateTag not mocked")
}

func (m *mockQuerier) GetTagByName(ctx context.Context, name string) (database.Tag, error) {
	if m.getTagByNameFn != nil {
		return m.getTagByNameFn(ctx, name)
	}
	return database.Tag{}, fmt.Errorf("GetTagByName not mocked")
}

func (m *mockQuerier) GetTagByAlias(ctx context.Context, alias string) (database.Tag, error) {
	if m.getTagByAliasFn != nil {
		return m.getTagByAliasFn(ctx, alias)
	}
	return database.Tag{}, fmt.Errorf("GetTagByAlias not mocked")
}

func (m *mockQuerier) AddMediaItemTag(ctx context.Context, arg database.AddMediaItemTagParams) error {
	if m.addMediaItemTagFn != nil {
		return m.addMediaItemTagFn(ctx, arg)
	}
	return fmt.Errorf("AddMediaItemTag not mocked")
}

func (m *mockQuerier) ListMediaItemTags(ctx context.Context, mediaItemID pgtype.UUID) ([]database.ListMediaItemTagsRow, error) {
	if m.listMediaItemTagsFn != nil {
		return m.listMediaItemTagsFn(ctx, mediaItemID)
	}
	return nil, fmt.Errorf("ListMediaItemTags not mocked")
}

func (m *mockQuerier) PlatformBreakdown(ctx context.Context, arg database.PlatformBreakdownParams) ([]database.PlatformBreakdownRow, error) {
	if m.platformBreakdownFn != nil {
		return m.platformBreakdownFn(ctx, arg)
	}
	return nil, fmt.Errorf("PlatformBreakdown not mocked")
}

func (m *mockQuerier) TagDistribution(ctx context.Context, arg database.TagDistributionParams) ([]database.TagDistributionRow, error) {
	if m.tagDistributionFn != nil {
		return m.tagDistributionFn(ctx, arg)
	}
	return nil, fmt.Errorf("TagDistribution not mocked")
}
