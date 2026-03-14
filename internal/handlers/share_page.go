package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/justestif/specto/components"
	"github.com/justestif/specto/internal/auth"
	"github.com/justestif/specto/internal/core"
)

// ShareProfilePage renders GET /share/{slug} — the public share profile.
// This is a standalone page with no navbar, fully server-rendered, no HTMX.
// Returns 404 if the user doesn't exist, has no profile slug, or profile is not published.
func (h *Handler) ShareProfilePage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	// GetBySlug only returns published profiles.
	pub, err := h.App.ShareProfiles.GetBySlug(r.Context(), slug)
	if err != nil {
		// Fall back to user lookup for backward compatibility (profile not yet created
		// or not yet published).
		user, userErr := h.App.Users.GetByProfileSlug(r.Context(), slug)
		if userErr != nil {
			http.NotFound(w, r)
			return
		}

		// Check if the current viewer is the profile owner — if so, render
		// a preview with their blocks even if unpublished.
		if viewer, ok := auth.UserFromContext(r.Context()); ok && viewer.ID == user.ID {
			profile, profErr := h.App.ShareProfiles.Get(r.Context(), user.ID)
			if profErr == nil && len(profile.Blocks) > 0 {
				pub = &core.PublicShareProfile{
					DisplayName: user.DisplayName,
					AvatarURL:   user.AvatarURL,
					Slug:        slug,
					Profile:     *profile,
				}
				// Fall through to the normal render path below.
			} else {
				data := components.ShareProfileData{
					DisplayName: user.DisplayName,
					AvatarURL:   user.AvatarURL,
					Slug:        slug,
					Blocks:      nil,
				}
				components.ShareProfilePage(data).Render(r.Context(), w)
				return
			}
		} else {
			// Not the owner — show empty placeholder.
			data := components.ShareProfileData{
				DisplayName: user.DisplayName,
				Slug:        slug,
				Blocks:      nil,
			}
			components.ShareProfilePage(data).Render(r.Context(), w)
			return
		}
	}

	// Render blocks with real data.
	blocks, err := h.renderPublicBlocks(r, pub)
	if err != nil {
		addContext(r, "share_page_render_error", err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := components.ShareProfileData{
		DisplayName: pub.DisplayName,
		AvatarURL:   pub.AvatarURL,
		Slug:        pub.Slug,
		Blocks:      blocks,
	}
	components.ShareProfilePage(data).Render(r.Context(), w)
}

// renderPublicBlocks converts domain block data into template-ready ShareBlocks.
// Uses the shared resolveBlocks for data fetching, then maps to template types.
func (h *Handler) renderPublicBlocks(r *http.Request, pub *core.PublicShareProfile) ([]components.ShareBlock, error) {
	profile := &pub.Profile
	resolved, err := h.resolveBlocks(r.Context(), profile.UserID, profile)
	if err != nil {
		return nil, err
	}

	var result []components.ShareBlock
	for _, rb := range resolved {
		switch rb.Type {
		case "top_genres":
			result = append(result, components.ShareBlock{
				Type:    "top_genres",
				Enabled: true,
				Bars:    toBars(rb.Tags),
			})

		case "mood_profile":
			summary := ""
			if len(rb.Tags) > 0 {
				summary = "mostly " + rb.Tags[0].Name
				if len(rb.Tags) > 1 {
					summary += " and " + rb.Tags[1].Name
				}
			}
			result = append(result, components.ShareBlock{
				Type:    "mood_profile",
				Enabled: true,
				Summary: summary,
				Bars:    toBars(rb.Tags),
			})

		case "top_creators":
			shareEntries := make([]components.ShareEntry, len(rb.Creators))
			for i, e := range rb.Creators {
				shareEntries[i] = components.ShareEntry{
					Rank:     i + 1,
					Name:     e.Creator,
					Icon:     mediaTypeIcon(e.MediaType),
					Platform: e.Platform,
				}
			}
			result = append(result, components.ShareBlock{
				Type:    "top_creators",
				Enabled: true,
				Entries: shareEntries,
			})

		case "platform_mix":
			var total int64
			for _, e := range rb.Mix {
				total += e.Count
			}
			bars := make([]components.ShareBar, len(rb.Mix))
			for i, e := range rb.Mix {
				pct := 0
				if total > 0 {
					pct = int(e.Count * 100 / total)
				}
				bars[i] = components.ShareBar{Label: e.Platform, Pct: pct}
			}
			result = append(result, components.ShareBlock{
				Type:    "platform_mix",
				Enabled: true,
				Bars:    bars,
			})

		case "currently_into":
			result = append(result, components.ShareBlock{
				Type:    "currently_into",
				Enabled: true,
				Text:    rb.Text,
			})

		case "recent_favorites":
			items := make([]components.ShareFavorite, len(rb.Items))
			for i, f := range rb.Items {
				items[i] = components.ShareFavorite{
					Title:    f.Title,
					Creator:  f.Creator,
					Platform: f.Platform,
					Type:     f.Type,
					Icon:     mediaTypeIcon(f.Type),
				}
			}
			result = append(result, components.ShareBlock{
				Type:      "recent_favorites",
				Enabled:   true,
				Favorites: items,
			})

		case "listening_stats":
			result = append(result, components.ShareBlock{
				Type:       "listening_stats",
				Enabled:    true,
				TotalItems: rb.Stats,
			})
		}
	}

	return result, nil
}

// toBars converts tag distribution entries into template bars with percentages.
func toBars(entries []core.TagDistributionEntry) []components.ShareBar {
	var total int64
	for _, e := range entries {
		total += e.Count
	}
	bars := make([]components.ShareBar, len(entries))
	for i, e := range entries {
		pct := 0
		if total > 0 {
			pct = int(e.Count * 100 / total)
		}
		bars[i] = components.ShareBar{Label: e.Name, Pct: pct}
	}
	return bars
}

// mediaTypeIcon returns a simple text icon for a media type.
func mediaTypeIcon(t string) string {
	switch core.MediaType(t) {
	case core.MediaMusic:
		return "♫"
	case core.MediaVideo:
		return "▶"
	case core.MediaArticle:
		return "📄"
	case core.MediaPodcast:
		return "🎙"
	default:
		return "•"
	}
}
