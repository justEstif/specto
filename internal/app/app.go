// Package app wires the core layer together and provides the top-level
// application struct used by HTTP handlers and middleware.
package app

import (
	"github.com/justestif/specto/internal/auth"
	"github.com/justestif/specto/internal/core"
	"github.com/justestif/specto/internal/core/store"
	"github.com/justestif/specto/internal/database"
)

// Config holds all configuration values needed by the application.
// Loaded once in main() and passed in — no env reads inside app.
type Config struct {
	EncryptionKey string // 64 hex chars for AES-256 credential encryption
	SessionSecret []byte // at least 32 bytes for session cookie signing
}

// App holds all core dependencies. It is created once at startup and
// shared across handlers and middleware via closures.
type App struct {
	// Core services
	Auth     *auth.Service
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
// The database Queries instance and config are injected — no globals read.
func New(db *database.Queries, cfg Config) *App {
	// Build store layer
	querier := db // *database.Queries satisfies store.Querier
	userStore := store.NewUserStore(querier)
	mediaItemStore := store.NewMediaItemStore(querier)
	pluginStateStore := store.NewPluginStateStore(querier, cfg.EncryptionKey)
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

	// Build auth service
	sessions := auth.NewSessionManager(cfg.SessionSecret)
	authSvc := auth.NewService(userStore, sessions)

	return &App{
		Auth:         authSvc,
		Registry:     registry,
		Syncer:       syncer,
		Insights:     insights,
		Users:        userStore,
		MediaItems:   mediaItemStore,
		PluginStates: pluginStateStore,
		SyncLogs:     syncLogStore,
		Tags:         tagStore,
	}
}
