package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/justestif/specto/internal/app"
)

// Handler holds dependencies for HTTP handlers. All route handlers
// are methods on this type, eliminating the need for global state.
type Handler struct {
	App *app.App
}

// New creates a Handler with the given application dependencies.
func New(application *app.App) *Handler {
	return &Handler{App: application}
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
