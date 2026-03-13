package handlers

import (
	"log"
	"net/http"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/csrf"

	"github.com/justestif/specto/components"
	"github.com/justestif/specto/internal/auth"
)

// settingsTabPartials maps tab names to their partial templ components.
var settingsTabPartials = map[string]func(components.SettingsData) templ.Component{
	"account":    components.SettingsAccount,
	"appearance": components.SettingsAppearance,
	"sharing":    components.SettingsSharing,
}

// validSettingsTab returns the tab name if valid, or "account" as default.
func validSettingsTab(tab string) string {
	if _, ok := settingsTabPartials[tab]; ok {
		return tab
	}
	return "account"
}

// SettingsPage renders GET /settings and GET /settings/{tab}.
// The tab is extracted from the URL; defaults to "account".
func (h *Handler) SettingsPage(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	tab := validSettingsTab(chi.URLParam(r, "tab"))
	data := components.SettingsData{
		User:      user,
		ActiveTab: tab,
		CSRFToken: csrf.Token(r),
	}
	components.SettingsPage(data).Render(r.Context(), w)
}

// SettingsPartial renders GET /partials/settings/{tab} — returns just
// the tab content for HTMX swap.
func (h *Handler) SettingsPartial(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	tab := validSettingsTab(chi.URLParam(r, "tab"))
	data := components.SettingsData{
		User:      user,
		ActiveTab: tab,
		CSRFToken: csrf.Token(r),
	}
	settingsTabPartials[tab](data).Render(r.Context(), w)
}

// SettingsAccountUpdate handles PUT /settings/account — saves profile changes.
func (h *Handler) SettingsAccountUpdate(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	displayName := r.FormValue("display_name")
	profileSlug := r.FormValue("profile_slug")

	if displayName == "" {
		data := components.SettingsData{
			User:      user,
			ActiveTab: "account",
			CSRFToken: csrf.Token(r),
			Message:   "Display name is required",
		}
		components.SettingsAccount(data).Render(r.Context(), w)
		return
	}

	var slugPtr *string
	if profileSlug != "" {
		slugPtr = &profileSlug
	}

	updatedUser, err := h.App.Users.UpdateProfile(r.Context(), user.ID, displayName, user.AvatarURL, slugPtr)
	if err != nil {
		log.Printf("settings: update profile error: %v", err)
		data := components.SettingsData{
			User:      user,
			ActiveTab: "account",
			CSRFToken: csrf.Token(r),
			Message:   "Failed to save changes. Please try again.",
		}
		components.SettingsAccount(data).Render(r.Context(), w)
		return
	}

	data := components.SettingsData{
		User:      updatedUser,
		ActiveTab: "account",
		CSRFToken: csrf.Token(r),
		Message:   "Profile updated successfully",
	}
	components.SettingsAccount(data).Render(r.Context(), w)
}
