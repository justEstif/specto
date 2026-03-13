package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/justestif/specto/internal/auth"
)

// OptionalAuth returns middleware that loads the user into context if a valid
// session exists, but does not reject unauthenticated requests. Use this on
// pages that render differently for logged-in vs anonymous visitors (e.g. the
// landing page navbar).
func OptionalAuth(authSvc *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := authSvc.Sessions.GetUserIDFromSession(r)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			user, err := authSvc.GetUserByID(r.Context(), userID)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := auth.ContextWithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth returns middleware that ensures the user is authenticated.
// It uses the auth.Service for session and user lookup.
func RequireAuth(authSvc *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := authSvc.Sessions.GetUserIDFromSession(r)
			if err != nil {
				writeUnauthorized(w, r)
				return
			}

			user, err := authSvc.GetUserByID(r.Context(), userID)
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
