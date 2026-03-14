package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/justestif/specto/internal/app"
	"github.com/justestif/specto/internal/auth"
	"github.com/justestif/specto/internal/core"
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

// pluginContext holds the common context extracted by requirePlugin.
type pluginContext struct {
	User   *core.UserInfo
	Name   string
	Plugin core.SourcePlugin
}

// requirePlugin extracts the authenticated user and looks up the plugin
// from the URL. It writes an error response and returns false if either
// step fails, so callers can early-return on !ok.
func (h *Handler) requirePlugin(w http.ResponseWriter, r *http.Request) (pluginContext, bool) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return pluginContext{}, false
	}

	name := chi.URLParam(r, "plugin")
	p := h.App.Registry.Get(name)
	if p == nil {
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("Plugin %q not found", name))
		return pluginContext{}, false
	}

	return pluginContext{User: user, Name: name, Plugin: p}, true
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
