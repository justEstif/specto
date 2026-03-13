package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/justestif/specto/components"
)

// ShareProfilePage renders GET /share/{slug} — the public share profile.
// This is a standalone page with no navbar, fully server-rendered, no HTMX.
// Returns 404 if the user doesn't exist or has no profile slug.
func (h *Handler) ShareProfilePage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	user, err := h.App.Users.GetByProfileSlug(r.Context(), slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// TODO: Load share_profiles config from DB once the table exists.
	// For now, render a basic profile with placeholder blocks.
	data := components.ShareProfileData{
		DisplayName: user.DisplayName,
		Slug:        slug,
		Blocks: []components.ShareBlock{
			{
				Type:    "top_genres",
				Enabled: true,
				Bars:    nil, // No data yet — will show "No data yet."
			},
			{
				Type:    "mood_profile",
				Enabled: true,
				Summary: "",
				Bars:    nil,
			},
			{
				Type:    "top_creators",
				Enabled: true,
				Entries: nil,
			},
			{
				Type:    "currently_into",
				Enabled: true,
				Text:    "",
			},
		},
	}

	components.ShareProfilePage(data).Render(r.Context(), w)
}
