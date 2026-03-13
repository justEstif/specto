package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/justestif/specto/components"
	"github.com/justestif/specto/internal/auth"
	"github.com/justestif/specto/internal/core"
)

// TimelinePage renders GET /timeline — the full chronological feed page.
func (h *Handler) TimelinePage(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	filters := parseTimelineFilters(r)
	items, hasMore := h.fetchTimelineItems(r, user, filters)

	data := components.TimelinePageData{
		User:      user,
		Items:     items,
		Filters:   filters,
		HasMore:   hasMore,
		Platforms: h.App.Registry.Platforms(),
	}
	components.TimelinePage(data).Render(r.Context(), w)
}

// TimelinePagePartial handles GET /partials/timeline-page — returns
// just the item list for HTMX filter/pagination swaps.
func (h *Handler) TimelinePagePartial(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	filters := parseTimelineFilters(r)
	items, hasMore := h.fetchTimelineItems(r, user, filters)

	components.TimelineItems(items, filters, hasMore).Render(r.Context(), w)
}

// fetchTimelineItems loads timeline items with DB-level filtering applied.
func (h *Handler) fetchTimelineItems(r *http.Request, user *core.UserInfo, f components.TimelineFilters) ([]core.MediaItem, bool) {
	now := time.Now().UTC()
	from := now.AddDate(0, 0, -90) // 90 days of history

	// Fetch one extra to determine if there are more items.
	fetchLimit := int32(f.Limit + 1)

	// Convert empty filter strings to nil for the store's optional params.
	platform := nilIfEmpty(f.Platform)
	mediaType := nilIfEmpty(f.Type)
	search := nilIfEmpty(f.Search)

	items, err := h.App.MediaItems.ListFiltered(r.Context(), user.ID, from, now, fetchLimit, int32(f.Offset), platform, mediaType, search)
	if err != nil {
		log.Printf("timeline page: list error: %v", err)
		return nil, false
	}

	hasMore := len(items) > f.Limit
	if hasMore {
		items = items[:f.Limit]
	}

	return items, hasMore
}

// nilIfEmpty returns nil for empty strings, or a pointer to s otherwise.
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func parseTimelineFilters(r *http.Request) components.TimelineFilters {
	q := r.URL.Query()
	return components.TimelineFilters{
		Platform: q.Get("platform"),
		Type:     q.Get("type"),
		Search:   q.Get("search"),
		Offset:   parseIntParam(q.Get("offset"), 0, 0, 10000),
		Limit:    parseIntParam(q.Get("limit"), 30, 1, 100),
	}
}
