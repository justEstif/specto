package handlers

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

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
func (h *Handler) renderPublicBlocks(r *http.Request, pub *core.PublicShareProfile) ([]components.ShareBlock, error) {
	profile := &pub.Profile
	var result []components.ShareBlock

	for _, block := range profile.Blocks {
		if !block.Enabled {
			continue
		}

		from, to := timeRangeBounds(block.TimeRange)

		switch block.Type {
		case "top_genres":
			entries, err := h.App.ShareProfiles.GetPublicTagDistribution(
				r.Context(), profile.UserID, from, to, limit(block.Count, 10),
				profile.ExcludedPlatforms, profile.ExcludedTags,
				strPtr("genre"),
			)
			if err != nil {
				return nil, fmt.Errorf("top_genres: %w", err)
			}
			result = append(result, components.ShareBlock{
				Type:    "top_genres",
				Enabled: true,
				Bars:    toBars(entries),
			})

		case "mood_profile":
			entries, err := h.App.ShareProfiles.GetPublicTagDistribution(
				r.Context(), profile.UserID, from, to, limit(block.Count, 10),
				profile.ExcludedPlatforms, profile.ExcludedTags,
				strPtr("mood"),
			)
			if err != nil {
				return nil, fmt.Errorf("mood_profile: %w", err)
			}
			summary := ""
			if len(entries) > 0 {
				summary = "mostly " + entries[0].Name
				if len(entries) > 1 {
					summary += " and " + entries[1].Name
				}
			}
			result = append(result, components.ShareBlock{
				Type:    "mood_profile",
				Enabled: true,
				Summary: summary,
				Bars:    toBars(entries),
			})

		case "top_creators":
			entries, err := h.App.ShareProfiles.GetPublicTopCreators(
				r.Context(), profile.UserID, from, to, limit(block.Count, 10),
				profile.ExcludedPlatforms,
			)
			if err != nil {
				return nil, fmt.Errorf("top_creators: %w", err)
			}
			shareEntries := make([]components.ShareEntry, len(entries))
			for i, e := range entries {
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
			entries, err := h.App.ShareProfiles.GetPublicPlatformMix(
				r.Context(), profile.UserID, from, to,
				profile.ExcludedPlatforms,
			)
			if err != nil {
				return nil, fmt.Errorf("platform_mix: %w", err)
			}
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
				Text:    block.Text,
			})

		case "recent_favorites":
			if len(block.ItemIDs) == 0 {
				continue
			}
			var items []components.ShareFavorite
			for _, idStr := range block.ItemIDs {
				id, err := uuid.Parse(idStr)
				if err != nil {
					continue
				}
				item, err := h.App.MediaItems.Get(r.Context(), profile.UserID, id)
				if err != nil || item == nil {
					continue
				}
				items = append(items, components.ShareFavorite{
					Title:    item.Title,
					Creator:  item.Creator,
					Platform: item.Platform,
					Type:     string(item.Type),
					Icon:     mediaTypeIcon(string(item.Type)),
				})
			}
			if len(items) > 0 {
				result = append(result, components.ShareBlock{
					Type:      "recent_favorites",
					Enabled:   true,
					Favorites: items,
				})
			}

		case "listening_stats":
			entries, err := h.App.ShareProfiles.GetPublicPlatformMix(
				r.Context(), profile.UserID, from, to,
				profile.ExcludedPlatforms,
			)
			if err != nil {
				return nil, fmt.Errorf("listening_stats: %w", err)
			}
			var totalItems int64
			for _, e := range entries {
				totalItems += e.Count
			}
			result = append(result, components.ShareBlock{
				Type:       "listening_stats",
				Enabled:    true,
				TotalItems: totalItems,
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
