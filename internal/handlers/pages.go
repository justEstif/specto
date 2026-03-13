package handlers

import (
	"errors"
	"net/http"

	"github.com/gorilla/csrf"

	"github.com/justestif/specto/components"
	"github.com/justestif/specto/internal/auth"
)

// LoginPage renders GET /login. If the user is already authenticated
// (detected via OptionalAuth middleware), redirect to the dashboard.
func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.UserFromContext(r.Context()); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	token := csrf.Token(r)
	components.LoginPage(token, "").Render(r.Context(), w)
}

// LoginSubmit handles POST /login from the HTML form. On success it sets
// the session and redirects to /. On failure it re-renders the login card
// with an error message (HTMX swaps just the card via outerHTML).
func (h *Handler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" || password == "" {
		token := csrf.Token(r)
		components.LoginCard(token, "Email and password are required").Render(r.Context(), w)
		return
	}

	user, err := h.App.Auth.Login(r.Context(), email, password)
	if err != nil {
		token := csrf.Token(r)
		if errors.Is(err, auth.ErrInvalidCredentials) {
			components.LoginCard(token, "Invalid email or password").Render(r.Context(), w)
			return
		}
		components.LoginCard(token, "Something went wrong. Please try again.").Render(r.Context(), w)
		return
	}

	if err := h.App.Auth.Sessions.SetUserSession(w, r, user.ID); err != nil {
		token := csrf.Token(r)
		components.LoginCard(token, "Failed to create session").Render(r.Context(), w)
		return
	}

	// On HTMX request, use HX-Redirect to do a full navigation to /
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// RegisterPage renders GET /register. Redirects if already logged in.
func (h *Handler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.UserFromContext(r.Context()); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	token := csrf.Token(r)
	components.RegisterPage(token, "").Render(r.Context(), w)
}

// RegisterSubmit handles POST /register from the HTML form. Validates
// input, creates the account, sets the session, and redirects to /.
func (h *Handler) RegisterSubmit(w http.ResponseWriter, r *http.Request) {
	displayName := r.FormValue("display_name")
	email := r.FormValue("email")
	password := r.FormValue("password")
	passwordConfirm := r.FormValue("password_confirm")

	token := csrf.Token(r)

	if displayName == "" || email == "" || password == "" {
		components.RegisterCard(token, "All fields are required").Render(r.Context(), w)
		return
	}

	if len(password) < 8 {
		components.RegisterCard(token, "Password must be at least 8 characters").Render(r.Context(), w)
		return
	}

	if password != passwordConfirm {
		components.RegisterCard(token, "Passwords do not match").Render(r.Context(), w)
		return
	}

	user, err := h.App.Auth.Register(r.Context(), email, displayName, password)
	if err != nil {
		if errors.Is(err, auth.ErrEmailTaken) {
			components.RegisterCard(token, "Email already registered").Render(r.Context(), w)
			return
		}
		components.RegisterCard(token, "Something went wrong. Please try again.").Render(r.Context(), w)
		return
	}

	if err := h.App.Auth.Sessions.SetUserSession(w, r, user.ID); err != nil {
		components.RegisterCard(token, "Account created but failed to sign in. Please try logging in.").Render(r.Context(), w)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// LogoutSubmit handles POST /logout from the HTML form (navbar sign-out).
// Clears the session and redirects to the landing page.
func (h *Handler) LogoutSubmit(w http.ResponseWriter, r *http.Request) {
	h.App.Auth.Sessions.ClearSession(w, r)
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
