package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/justestif/specto/internal/auth"
	"github.com/justestif/specto/internal/core"
)

// --- Request/Response types ---

type shareProfileResponse struct {
	Published         bool              `json:"published"`
	ProfileSlug       *string           `json:"profile_slug,omitempty"`
	ExcludedPlatforms []string          `json:"excluded_platforms"`
	ExcludedTags      []string          `json:"excluded_tags"`
	Blocks            []core.ShareBlock `json:"blocks"`
}

type updateShareProfileRequest struct {
	Published         bool              `json:"published"`
	Slug              *string           `json:"slug,omitempty"`
	ExcludedPlatforms []string          `json:"excluded_platforms"`
	ExcludedTags      []string          `json:"excluded_tags"`
	Blocks            []core.ShareBlock `json:"blocks"`
}

type itemPrivacyRequest struct {
	Private bool `json:"private"`
}

type itemPrivacyResponse struct {
	ID      string `json:"id"`
	Private bool   `json:"private"`
}

// --- Share Profile block rendering types ---

type previewBlockResponse struct {
	Type  string `json:"type"`
	Title string `json:"title"`
	Items any    `json:"items,omitempty"`
	Text  string `json:"text,omitempty"`
}

type barItem struct {
	Name    string `json:"name"`
	Count   int64  `json:"count"`
	Percent int    `json:"percent,omitempty"`
}

type creatorItem struct {
	Rank     int    `json:"rank"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Type     string `json:"type"`
	Count    int64  `json:"count"`
}

// --- Handlers ---

// GetShareProfile handles GET /api/v1/share-profile
func (h *Handler) GetShareProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	profile, err := h.App.ShareProfiles.Get(r.Context(), user.ID)
	if err != nil {
		// No profile yet — return empty defaults
		writeJSON(w, http.StatusOK, dataResponse{Data: shareProfileResponse{
			Published:         false,
			ProfileSlug:       user.ProfileSlug,
			ExcludedPlatforms: []string{},
			ExcludedTags:      []string{},
			Blocks:            defaultBlocks(),
		}})
		return
	}

	slug := profile.Slug
	if slug == nil {
		slug = user.ProfileSlug
	}

	writeJSON(w, http.StatusOK, dataResponse{Data: shareProfileResponse{
		Published:         profile.Published,
		ProfileSlug:       slug,
		ExcludedPlatforms: profile.ExcludedPlatforms,
		ExcludedTags:      profile.ExcludedTags,
		Blocks:            profile.Blocks,
	}})
}

// UpdateShareProfile handles PUT /api/v1/share-profile
func (h *Handler) UpdateShareProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	var req updateShareProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body")
		return
	}

	// Use user's profile slug if not provided in request
	slug := req.Slug
	if slug == nil {
		slug = user.ProfileSlug
	}

	profile := core.ShareProfile{
		Blocks:            req.Blocks,
		ExcludedPlatforms: req.ExcludedPlatforms,
		ExcludedTags:      req.ExcludedTags,
		Published:         req.Published,
		Slug:              slug,
	}

	result, err := h.App.ShareProfiles.Upsert(r.Context(), user.ID, profile)
	if err != nil {
		addContext(r, "share_update_error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update share profile")
		return
	}

	writeJSON(w, http.StatusOK, dataResponse{Data: shareProfileResponse{
		Published:         result.Published,
		ProfileSlug:       result.Slug,
		ExcludedPlatforms: result.ExcludedPlatforms,
		ExcludedTags:      result.ExcludedTags,
		Blocks:            result.Blocks,
	}})
}

// SharePreview handles GET /api/v1/share-profile/preview
func (h *Handler) SharePreview(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	profile, err := h.App.ShareProfiles.Get(r.Context(), user.ID)
	if err != nil {
		// No profile — return empty preview with default blocks
		writeJSON(w, http.StatusOK, dataResponse{Data: map[string]any{
			"blocks": []previewBlockResponse{},
		}})
		return
	}

	blocks, err := h.renderBlocks(r.Context(), user.ID, profile)
	if err != nil {
		addContext(r, "share_preview_error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to render preview")
		return
	}

	writeJSON(w, http.StatusOK, dataResponse{Data: map[string]any{
		"blocks": blocks,
	}})
}

// ToggleItemPrivate handles POST /api/v1/items/{id}/privacy
func (h *Handler) ToggleItemPrivate(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	idStr := chi.URLParam(r, "id")
	itemID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid item ID")
		return
	}

	var req itemPrivacyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body")
		return
	}

	if err := h.App.ShareProfiles.SetItemPrivacy(r.Context(), user.ID, itemID, req.Private); err != nil {
		addContext(r, "share_privacy_error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update item privacy")
		return
	}

	writeJSON(w, http.StatusOK, dataResponse{Data: itemPrivacyResponse{
		ID:      itemID.String(),
		Private: req.Private,
	}})
}

// --- Block data resolution ---

// resolvedBlock holds the raw data fetched for a share block, before
// conversion to a format-specific output type (JSON preview or HTML template).
// This eliminates duplicate data-fetching logic between renderBlocks and
// renderPublicBlocks.
type resolvedBlock struct {
	Type     string
	Tags     []core.TagDistributionEntry // top_genres, mood_profile
	Creators []core.TopCreatorEntry      // top_creators
	Mix      []core.PlatformMixEntry     // platform_mix
	Text     string                      // currently_into
	Items    []resolvedFavorite          // recent_favorites
	Stats    int64                       // listening_stats total
}

type resolvedFavorite struct {
	Title    string
	Creator  string
	Platform string
	Type     string
}

// resolveBlocks fetches the underlying data for each enabled block in the profile.
// The caller converts resolvedBlock values to the appropriate output format.
func (h *Handler) resolveBlocks(ctx context.Context, userID uuid.UUID, profile *core.ShareProfile) ([]resolvedBlock, error) {
	var result []resolvedBlock

	for _, block := range profile.Blocks {
		if !block.Enabled {
			continue
		}

		from, to := timeRangeBounds(block.TimeRange)

		switch block.Type {
		case "top_genres":
			entries, err := h.App.ShareProfiles.GetPublicTagDistribution(
				ctx, userID, from, to, limit(block.Count, 10),
				profile.ExcludedPlatforms, profile.ExcludedTags,
				strPtr("genre"),
			)
			if err != nil {
				return nil, fmt.Errorf("top_genres: %w", err)
			}
			result = append(result, resolvedBlock{Type: "top_genres", Tags: entries})

		case "mood_profile":
			entries, err := h.App.ShareProfiles.GetPublicTagDistribution(
				ctx, userID, from, to, limit(block.Count, 10),
				profile.ExcludedPlatforms, profile.ExcludedTags,
				strPtr("mood"),
			)
			if err != nil {
				return nil, fmt.Errorf("mood_profile: %w", err)
			}
			result = append(result, resolvedBlock{Type: "mood_profile", Tags: entries})

		case "top_creators":
			entries, err := h.App.ShareProfiles.GetPublicTopCreators(
				ctx, userID, from, to, limit(block.Count, 10),
				profile.ExcludedPlatforms,
			)
			if err != nil {
				return nil, fmt.Errorf("top_creators: %w", err)
			}
			result = append(result, resolvedBlock{Type: "top_creators", Creators: entries})

		case "platform_mix":
			entries, err := h.App.ShareProfiles.GetPublicPlatformMix(
				ctx, userID, from, to, profile.ExcludedPlatforms,
			)
			if err != nil {
				return nil, fmt.Errorf("platform_mix: %w", err)
			}
			result = append(result, resolvedBlock{Type: "platform_mix", Mix: entries})

		case "currently_into":
			result = append(result, resolvedBlock{Type: "currently_into", Text: block.Text})

		case "recent_favorites":
			if len(block.ItemIDs) == 0 {
				continue
			}
			var items []resolvedFavorite
			for _, idStr := range block.ItemIDs {
				id, err := uuid.Parse(idStr)
				if err != nil {
					continue
				}
				item, err := h.App.MediaItems.Get(ctx, userID, id)
				if err != nil || item == nil {
					continue
				}
				items = append(items, resolvedFavorite{
					Title:    item.Title,
					Creator:  item.Creator,
					Platform: item.Platform,
					Type:     string(item.Type),
				})
			}
			if len(items) > 0 {
				result = append(result, resolvedBlock{Type: "recent_favorites", Items: items})
			}

		case "listening_stats":
			entries, err := h.App.ShareProfiles.GetPublicPlatformMix(
				ctx, userID, from, to, profile.ExcludedPlatforms,
			)
			if err != nil {
				return nil, fmt.Errorf("listening_stats: %w", err)
			}
			var total int64
			for _, e := range entries {
				total += e.Count
			}
			result = append(result, resolvedBlock{Type: "listening_stats", Stats: total})
		}
	}

	return result, nil
}

// --- Block rendering (JSON preview) ---

// renderBlocks produces the preview data for each enabled block.
func (h *Handler) renderBlocks(ctx context.Context, userID uuid.UUID, profile *core.ShareProfile) ([]previewBlockResponse, error) {
	resolved, err := h.resolveBlocks(ctx, userID, profile)
	if err != nil {
		return nil, err
	}

	var blocks []previewBlockResponse
	for _, rb := range resolved {
		switch rb.Type {
		case "top_genres":
			blocks = append(blocks, previewBlockResponse{
				Type:  "top_genres",
				Title: "Top Genres",
				Items: toBarItems(rb.Tags),
			})
		case "mood_profile":
			blocks = append(blocks, previewBlockResponse{
				Type:  "mood_profile",
				Title: "Mood Profile",
				Items: toBarItems(rb.Tags),
			})
		case "top_creators":
			items := make([]creatorItem, len(rb.Creators))
			for i, e := range rb.Creators {
				items[i] = creatorItem{
					Rank:     i + 1,
					Name:     e.Creator,
					Platform: e.Platform,
					Type:     e.MediaType,
					Count:    e.Count,
				}
			}
			blocks = append(blocks, previewBlockResponse{
				Type:  "top_creators",
				Title: "Top Creators",
				Items: items,
			})
		case "platform_mix":
			blocks = append(blocks, previewBlockResponse{
				Type:  "platform_mix",
				Title: "Platform Mix",
				Items: toPlatformBarItems(rb.Mix),
			})
		case "currently_into":
			blocks = append(blocks, previewBlockResponse{
				Type:  "currently_into",
				Title: "Currently Into",
				Text:  rb.Text,
			})
		}
	}

	return blocks, nil
}

// toPlatformBarItems converts platform mix entries to bar items with percentages.
func toPlatformBarItems(entries []core.PlatformMixEntry) []barItem {
	var total int64
	for _, e := range entries {
		total += e.Count
	}
	items := make([]barItem, len(entries))
	for i, e := range entries {
		pct := 0
		if total > 0 {
			pct = int(e.Count * 100 / total)
		}
		items[i] = barItem{
			Name:    e.Platform,
			Count:   e.Count,
			Percent: pct,
		}
	}
	return items
}

// --- Helpers ---

func defaultBlocks() []core.ShareBlock {
	return []core.ShareBlock{
		{Type: "top_genres", Enabled: true, TimeRange: "30d"},
		{Type: "mood_profile", Enabled: true, TimeRange: "30d"},
		{Type: "top_creators", Enabled: true, TimeRange: "30d", Count: 10},
		{Type: "platform_mix", Enabled: false, TimeRange: "30d"},
		{Type: "currently_into", Enabled: true},
	}
}

func timeRangeBounds(tr string) (time.Time, time.Time) {
	now := time.Now().UTC()
	to := now
	var from time.Time

	switch tr {
	case "7d":
		from = now.AddDate(0, 0, -7)
	case "90d":
		from = now.AddDate(0, 0, -90)
	case "all":
		from = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	default: // "30d" or empty
		from = now.AddDate(0, 0, -30)
	}

	return from, to
}

func limit(n, defaultN int) int32 {
	if n > 0 {
		return int32(n)
	}
	return int32(defaultN)
}

func strPtr(s string) *string {
	return &s
}

func toBarItems(entries []core.TagDistributionEntry) []barItem {
	var total int64
	for _, e := range entries {
		total += e.Count
	}
	items := make([]barItem, len(entries))
	for i, e := range entries {
		pct := 0
		if total > 0 {
			pct = int(e.Count * 100 / total)
		}
		items[i] = barItem{
			Name:    e.Name,
			Count:   e.Count,
			Percent: pct,
		}
	}
	return items
}
