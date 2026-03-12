package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/justestif/specto/internal/database"
)

// Handler holds dependencies for HTTP handlers. All route handlers
// are methods on this type, eliminating the need for global state.
type Handler struct {
	DB *database.Queries
}

// New creates a Handler with the given dependencies.
func New(db *database.Queries) *Handler {
	return &Handler{DB: db}
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
