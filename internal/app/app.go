// Package app wires the core layer together and provides the top-level
// application struct used by HTTP handlers and middleware.
package app

import (
	"log/slog"

	"github.com/justestif/specto/internal/auth"
	"github.com/justestif/specto/internal/core"
	"github.com/justestif/specto/internal/core/store"
	"github.com/justestif/specto/internal/database"
	"github.com/justestif/specto/internal/plugins/anilist"
	"github.com/justestif/specto/internal/plugins/lastfm"
	"github.com/justestif/specto/internal/plugins/tmdb"
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
	LastfmAPIKey  string                       // Last.fm API key for music enrichment (optional)
	TMDBAPIKey    string                       // TMDB API key for movie/TV enrichment (optional)
	LLMEnricher   core.Enricher                // LLM enricher (nil to skip Phase 2)
	Logger        *slog.Logger                 // structured logger; nil falls back to slog.Default()
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

	// Logger is the application-wide structured logger.
	Logger *slog.Logger

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
	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}

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
		0, // use default min sync interval
		log,
	)

	// Build enrichment infrastructure
	// Build provider list. API-key-gated providers are conditional;
	// providers with no auth (like AniList) are always registered.
	// LLM enricher (cfg.LLMEnricher) runs in Phase 2 after API providers.
	var providers []core.EnrichmentProvider
	providers = append(providers, anilist.New())
	if cfg.LastfmAPIKey != "" {
		providers = append(providers, lastfm.New(cfg.LastfmAPIKey))
	}
	if cfg.TMDBAPIKey != "" {
		providers = append(providers, tmdb.New(cfg.TMDBAPIKey))
	}
	coordinator := core.NewEnrichmentCoordinator(providers, cfg.LLMEnricher, log)
	worker := core.NewEnrichmentWorker(
		coordinator,
		mediaItemStore,
		tagStore,
		log,
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
		Logger:        log,
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
