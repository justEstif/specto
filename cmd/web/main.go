package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/justestif/specto/internal/app"
	"github.com/justestif/specto/internal/database"
	"github.com/justestif/specto/internal/handlers"
	customMiddleware "github.com/justestif/specto/internal/middleware"
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

	// Initialize core application layer
	application := app.New(database.DB, app.Config{
		EncryptionKey: encKey,
		SessionSecret: sessionSecret,
	})

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

	// Protected HTML routes with CSRF
	r.Group(func(r chi.Router) {
		r.Use(csrfMw)
		r.Get("/", h.Home)
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
				r.Post("/import", h.ImportPlugin)
				r.Delete("/disconnect", h.DisconnectPlugin)
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
		})
	})

	// Log registered plugins (if any)
	for _, name := range application.Registry.List() {
		log.Printf("Registered plugin: %s", name)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Server starting on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
