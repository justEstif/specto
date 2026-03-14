package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/justestif/specto/internal/app"
	"github.com/justestif/specto/internal/logger"
)

// Handler holds dependencies for HTTP handlers. All route handlers
// are methods on this type, eliminating the need for global state.
type Handler struct {
	App *app.App
	log *slog.Logger
}

// New creates a Handler with the given application dependencies.
func New(application *app.App) *Handler {
	return &Handler{
		App: application,
		log: application.Logger,
	}
}

// addContext adds business context to the request's wide event.
// Handlers call this to enrich the per-request log event with
// domain-specific fields like user_id, plugin name, etc.
func addContext(r *http.Request, key string, value any) {
	if we := logger.FromContext(r.Context()); we != nil {
		we.Set(key, value)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
