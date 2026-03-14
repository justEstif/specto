package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/justestif/specto/internal/app"
	"github.com/justestif/specto/internal/core"
	"github.com/justestif/specto/internal/database"
	"github.com/justestif/specto/internal/enrichment"
	"github.com/justestif/specto/internal/handlers"
	customMiddleware "github.com/justestif/specto/internal/middleware"
	"github.com/justestif/specto/internal/plugins/spotify"
	"github.com/justestif/specto/internal/plugins/youtube"
)

func main() {
	// Initialize database
	if err := database.InitDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Load configuration from environment (single place for all env reads)
	encKey := os.Getenv("ENCRYPTION_KEY")
	if encKey == "" {
		log.Fatal("ENCRYPTION_KEY environment variable not set (must be 64 hex chars for AES-256)")
	}

	sessionSecret := []byte(os.Getenv("SESSION_SECRET"))
	if len(sessionSecret) < 32 {
		log.Fatal("SESSION_SECRET must be at least 32 bytes long")
	}

	// Load optional OAuth client credentials from environment
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}

	oauthClients := make(map[string]app.OAuthClientConfig)
	if id, secret := os.Getenv("SPOTIFY_CLIENT_ID"), os.Getenv("SPOTIFY_CLIENT_SECRET"); id != "" && secret != "" {
		oauthClients["spotify-api"] = app.OAuthClientConfig{
			ClientID:     id,
			ClientSecret: secret,
		}
	}
	if id, secret := os.Getenv("YOUTUBE_CLIENT_ID"), os.Getenv("YOUTUBE_CLIENT_SECRET"); id != "" && secret != "" {
		oauthClients["youtube-api"] = app.OAuthClientConfig{
			ClientID:     id,
			ClientSecret: secret,
		}
	}
	// App-level OAuth login providers (Google, GitHub)
	if id, secret := os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET"); id != "" && secret != "" {
		oauthClients["google"] = app.OAuthClientConfig{
			ClientID:     id,
			ClientSecret: secret,
		}
	}
	if id, secret := os.Getenv("GITHUB_CLIENT_ID"), os.Getenv("GITHUB_CLIENT_SECRET"); id != "" && secret != "" {
		oauthClients["github"] = app.OAuthClientConfig{
			ClientID:     id,
			ClientSecret: secret,
		}
	}

	// Load optional enrichment API keys
	lastfmAPIKey := os.Getenv("LASTFM_API_KEY")
	tmdbAPIKey := os.Getenv("TMDB_API_KEY")

	// Initialize optional LLM enricher (Genkit)
	var llmEnricher core.Enricher
	if llmProvider := os.Getenv("LLM_PROVIDER"); llmProvider != "" {
		enricher, err := enrichment.New(context.Background(), enrichment.Config{
			Provider: llmProvider,
			Model:    os.Getenv("LLM_MODEL"),
			APIKey:   os.Getenv("LLM_API_KEY"),
		}, nil)
		if err != nil {
			log.Fatalf("Failed to initialize LLM enricher: %v", err)
		}
		llmEnricher = enricher
	}

	// Initialize core application layer
	application := app.New(database.DB, app.Config{
		EncryptionKey: encKey,
		SessionSecret: sessionSecret,
		BaseURL:       baseURL,
		OAuthClients:  oauthClients,
		LastfmAPIKey:  lastfmAPIKey,
		TMDBAPIKey:    tmdbAPIKey,
		LLMEnricher:   llmEnricher,
	})

	// Register plugins
	if err := application.Registry.Register(spotify.New()); err != nil {
		log.Fatalf("Failed to register spotify plugin: %v", err)
	}
	if err := application.Registry.Register(spotify.NewAPI()); err != nil {
		log.Fatalf("Failed to register spotify-api plugin: %v", err)
	}
	if err := application.Registry.Register(youtube.NewWithEnrich()); err != nil {
		log.Fatalf("Failed to register youtube plugin: %v", err)
	}
	if err := application.Registry.Register(youtube.NewAPI()); err != nil {
		log.Fatalf("Failed to register youtube-api plugin: %v", err)
	}

	// Wire handlers with dependencies
	h := handlers.New(application)

	r := chi.NewRouter()

	// Standard middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	// CSRF protection - set secure=true in production
	csrfKey := []byte(os.Getenv("CSRF_KEY"))
	if len(csrfKey) != 32 {
		log.Fatal("CSRF_KEY must be exactly 32 bytes long")
	}
	csrfMw := customMiddleware.SetupCSRF(csrfKey, false)

	// Static files (no CSRF needed)
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// HTML routes with CSRF + optional auth (for navbar state)
	r.Group(func(r chi.Router) {
		r.Use(csrfMw)
		r.Use(customMiddleware.OptionalAuth(application.Auth))

		// Public pages (auth-aware for navbar state)
		r.Get("/", h.Home)
		r.Get("/login", h.LoginPage)
		r.Post("/login", h.LoginSubmit)
		r.Get("/register", h.RegisterPage)
		r.Post("/register", h.RegisterSubmit)
		r.Post("/logout", h.LogoutSubmit)

		// OAuth app login (Google, GitHub)
		r.Get("/auth/google/login", h.GoogleLogin)
		r.Get("/auth/google/callback", h.GoogleCallback)
		r.Get("/auth/github/login", h.GithubLogin)
		r.Get("/auth/github/callback", h.GithubCallback)

		// Public share profile (no auth required, standalone page)
		r.Get("/share/{slug}", h.ShareProfilePage)
	})

	// Authenticated HTML pages
	r.Group(func(r chi.Router) {
		r.Use(csrfMw)
		r.Use(customMiddleware.RequireAuth(application.Auth))

		r.Get("/timeline", h.TimelinePage)
		r.Get("/plugins", h.PluginsPage)
		r.Get("/settings", h.SettingsPage)
		r.Get("/settings/{tab}", h.SettingsPage)
		r.Put("/settings/account", h.SettingsAccountUpdate)
	})

	// Authenticated HTML partials (for HTMX swaps)
	r.Group(func(r chi.Router) {
		r.Use(csrfMw)
		r.Use(customMiddleware.RequireAuth(application.Auth))

		r.Get("/partials/dashboard", h.DashboardPartial)
		r.Get("/partials/activity-chart", h.ActivityChartPartial)
		r.Get("/partials/timeline", h.RecentItemsPartial)
		r.Get("/partials/timeline-page", h.TimelinePagePartial)
		r.Get("/partials/settings/{tab}", h.SettingsPartial)
	})

	// JSON API (v1)
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", h.Health)

		// Auth (public)
		r.Post("/auth/register", h.Register)
		r.Post("/auth/login", h.Login)

		// Authenticated routes
		r.Group(func(r chi.Router) {
			r.Use(customMiddleware.RequireAuth(application.Auth))

			// Session
			r.Get("/session", h.Session)
			r.Delete("/session", h.Logout)

			// Plugins
			r.Get("/plugins", h.ListPlugins)
			r.Route("/plugins/{plugin}", func(r chi.Router) {
				r.Get("/", h.GetPlugin)
				r.Post("/connect", h.ConnectPlugin)
				r.Get("/callback", h.OAuthCallback)
				r.Post("/import", h.ImportPlugin)
				r.Delete("/disconnect", h.DisconnectPlugin)
				r.Delete("/data", h.DeletePluginData)
				r.Post("/sync", h.SyncPlugin)
				r.Get("/sync-history", h.SyncHistory)
			})

			// Timeline
			r.Get("/timeline", h.Timeline)

			// Insights
			r.Get("/insights/summary", h.InsightsSummary)
			r.Get("/insights/platform-breakdown", h.InsightsPlatformBreakdown)
			r.Get("/insights/tags", h.InsightsTags)
			r.Get("/insights/timeline", h.InsightsTimeline)

			// Share profile
			r.Route("/share-profile", func(r chi.Router) {
				r.Get("/", h.GetShareProfile)
				r.Put("/", h.UpdateShareProfile)
				r.Get("/preview", h.SharePreview)
			})

			// Item privacy
			r.Post("/items/{id}/privacy", h.ToggleItemPrivate)
		})
	})

	// Log registered plugins (if any)
	for _, name := range application.Registry.List() {
		log.Printf("Registered plugin: %s", name)
	}

	// Start enrichment worker in background
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	go application.Worker.Start(workerCtx)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// Graceful shutdown: listen for SIGINT/SIGTERM
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on http://localhost:%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")

	// Stop enrichment worker
	workerCancel()

	// Graceful HTTP shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
