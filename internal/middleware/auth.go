package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/justestif/specto/internal/auth"
	"github.com/justestif/specto/internal/database"
)

// RequireAuth returns middleware that ensures the user is authenticated.
// The database.Queries instance is injected rather than using a global.
func RequireAuth(db *database.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := auth.GetUserIDFromSession(r)
			if err != nil {
				writeUnauthorized(w, r)
				return
			}

			user, err := auth.GetUserByID(r.Context(), db, userID)
			if err != nil {
				writeUnauthorized(w, r)
				return
			}

			ctx := auth.ContextWithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeUnauthorized(w http.ResponseWriter, r *http.Request) {
	if isAPIRequest(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"code":    "unauthorized",
				"message": "Authentication required",
			},
		})
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func isAPIRequest(r *http.Request) bool {
	return len(r.URL.Path) >= 5 && r.URL.Path[:5] == "/api/"
}
