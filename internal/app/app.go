// Package app wires the core layer together and provides the top-level
// application struct used by HTTP handlers and middleware.
package app

import (
	"github.com/justestif/specto/internal/auth"
	"github.com/justestif/specto/internal/core"
	"github.com/justestif/specto/internal/core/store"
	"github.com/justestif/specto/internal/database"
)

// OAuthClientConfig holds the client credentials for an OAuth provider.
type OAuthClientConfig struct {
	ClientID     string
	ClientSecret string
}

// Config holds all configuration values needed by the application.
// Loaded once in main() and passed in — no env reads inside app.
type Config struct {
	EncryptionKey string                       // 64 hex chars for AES-256 credential encryption
	SessionSecret []byte                       // at least 32 bytes for session cookie signing
	BaseURL       string                       // server's public base URL (e.g., "http://localhost:3000")
	OAuthClients  map[string]OAuthClientConfig // keyed by plugin name
}

// App holds all core dependencies. It is created once at startup and
// shared across handlers and middleware via closures.
type App struct {
	// Core services
	Auth        *auth.Service
	OAuth       *auth.OAuthService
	Registry    *core.PluginRegistry
	Syncer      *core.SyncService
	Insights    *core.InsightsService
	Coordinator *core.EnrichmentCoordinator
	Worker      *core.EnrichmentWorker

	// Core stores (exposed for handlers that need direct store access)
	Users         core.UserStore
	MediaItems    core.MediaItemStore
	PluginStates  core.PluginStateStore
	SyncLogs      core.SyncLogStore
	Tags          core.TagStore
	ShareProfiles core.ShareProfileStore
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
	shareProfileStore := store.NewShareProfileStore(querier)

	// Build core services
	registry := core.NewPluginRegistry()

	syncer := core.NewSyncService(
		registry,
		pluginStateStore,
		mediaItemStore,
		syncLogStore,
		0,   // use default min sync interval
		nil, // use default logger
	)

	// Build enrichment infrastructure
	// API providers are empty for now — individual provider beans will register them.
	// LLM enricher is nil for now — the Genkit LLM enricher bean will set it.
	coordinator := core.NewEnrichmentCoordinator(nil, nil, nil)
	worker := core.NewEnrichmentWorker(
		coordinator,
		mediaItemStore,
		tagStore,
		nil,                           // use default logger
		core.EnrichmentWorkerConfig{}, // use defaults
	)

	// Wire token refresher after OAuth service is created (set below)

	insights := core.NewInsightsService(insightsStore)

	// Build auth service
	sessions := auth.NewSessionManager(cfg.SessionSecret)
	authSvc := auth.NewService(userStore, sessions)

	// Build OAuth service
	oauthClients := make(map[string]auth.OAuthClientCredentials, len(cfg.OAuthClients))
	for name, oc := range cfg.OAuthClients {
		oauthClients[name] = auth.OAuthClientCredentials{
			ClientID:     oc.ClientID,
			ClientSecret: oc.ClientSecret,
		}
	}
	oauthSvc := auth.NewOAuthService(cfg.BaseURL, oauthClients, nil)

	// Wire token refresher so SyncService auto-refreshes expired OAuth tokens
	syncer.TokenRefresher = oauthSvc

	return &App{
		Auth:          authSvc,
		OAuth:         oauthSvc,
		Registry:      registry,
		Syncer:        syncer,
		Insights:      insights,
		Coordinator:   coordinator,
		Worker:        worker,
		Users:         userStore,
		MediaItems:    mediaItemStore,
		PluginStates:  pluginStateStore,
		SyncLogs:      syncLogStore,
		Tags:          tagStore,
		ShareProfiles: shareProfileStore,
	}
}
