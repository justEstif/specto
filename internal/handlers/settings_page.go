package handlers

import (
	"log"
	"net/http"

	"github.com/gorilla/csrf"

	"github.com/justestif/specto/components"
	"github.com/justestif/specto/internal/auth"
)

// SettingsPage renders GET /settings — the account settings tab.
func (h *Handler) SettingsPage(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := components.SettingsData{
		User:      user,
		ActiveTab: "account",
		CSRFToken: csrf.Token(r),
	}
	components.SettingsPage(data).Render(r.Context(), w)
}

// SettingsAppearancePage renders GET /settings/appearance.
func (h *Handler) SettingsAppearancePage(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := components.SettingsData{
		User:      user,
		ActiveTab: "appearance",
		CSRFToken: csrf.Token(r),
	}
	components.SettingsPage(data).Render(r.Context(), w)
}

// SettingsSharingPage renders GET /settings/sharing.
func (h *Handler) SettingsSharingPage(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := components.SettingsData{
		User:      user,
		ActiveTab: "sharing",
		CSRFToken: csrf.Token(r),
	}
	components.SettingsPage(data).Render(r.Context(), w)
}

// SettingsAccountPartial renders the account tab content for HTMX swap.
func (h *Handler) SettingsAccountPartial(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	data := components.SettingsData{
		User:      user,
		ActiveTab: "account",
		CSRFToken: csrf.Token(r),
	}
	components.SettingsAccount(data).Render(r.Context(), w)
}

// SettingsAppearancePartial renders the appearance tab content for HTMX swap.
func (h *Handler) SettingsAppearancePartial(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	data := components.SettingsData{
		User:      user,
		ActiveTab: "appearance",
		CSRFToken: csrf.Token(r),
	}
	components.SettingsAppearance(data).Render(r.Context(), w)
}

// SettingsSharingPartial renders the sharing tab content for HTMX swap.
func (h *Handler) SettingsSharingPartial(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	data := components.SettingsData{
		User:      user,
		ActiveTab: "sharing",
		CSRFToken: csrf.Token(r),
	}
	components.SettingsSharing(data).Render(r.Context(), w)
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
