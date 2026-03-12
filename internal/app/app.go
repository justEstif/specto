// Package app wires the core layer together and provides the top-level
// application struct used by HTTP handlers and middleware.
package app

import (
	"fmt"
	"os"

	"github.com/justestif/specto/internal/core"
	"github.com/justestif/specto/internal/core/store"
	"github.com/justestif/specto/internal/database"
)

// App holds all core dependencies. It is created once at startup and
// shared across handlers and middleware via closures.
type App struct {
	// DB is the sqlc Queries instance for direct database access
	// (used by auth until it's refactored to use core stores).
	DB *database.Queries

	// Core services
	Registry *core.PluginRegistry
	Syncer   *core.SyncService
	Insights *core.InsightsService

	// Core stores (exposed for handlers that need direct store access)
	Users        core.UserStore
	MediaItems   core.MediaItemStore
	PluginStates core.PluginStateStore
	SyncLogs     core.SyncLogStore
	Tags         core.TagStore
}

// New initializes all core components and returns a fully wired App.
// It reads ENCRYPTION_KEY from the environment for credential encryption.
// The database must already be initialized (database.DB and database.Pool set).
func New(db *database.Queries) (*App, error) {
	// Encryption key for plugin credentials
	encKey := os.Getenv("ENCRYPTION_KEY")
	if encKey == "" {
		return nil, fmt.Errorf("ENCRYPTION_KEY environment variable not set (must be 64 hex chars for AES-256)")
	}

	// Build store layer
	querier := db // *database.Queries satisfies store.Querier
	userStore := store.NewUserStore(querier)
	mediaItemStore := store.NewMediaItemStore(querier)
	pluginStateStore := store.NewPluginStateStore(querier, encKey)
	syncLogStore := store.NewSyncLogStore(querier)
	tagStore := store.NewTagStore(querier)
	insightsStore := store.NewInsightsStore(querier)

	// Build core services
	registry := core.NewPluginRegistry()
	enricher := &core.NoOpEnricher{}

	syncer := core.NewSyncService(
		registry,
		pluginStateStore,
		mediaItemStore,
		syncLogStore,
		tagStore,
		enricher,
		0,   // use default min sync interval
		nil, // use default logger
	)

	insights := core.NewInsightsService(insightsStore)

	return &App{
		DB:           db,
		Registry:     registry,
		Syncer:       syncer,
		Insights:     insights,
		Users:        userStore,
		MediaItems:   mediaItemStore,
		PluginStates: pluginStateStore,
		SyncLogs:     syncLogStore,
		Tags:         tagStore,
	}, nil
}
