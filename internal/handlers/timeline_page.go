package handlers

import (
	"log"
	"net/http"
	"strings"
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
		User:    user,
		Items:   items,
		Filters: filters,
		HasMore: hasMore,
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

// fetchTimelineItems loads timeline items with filtering applied.
func (h *Handler) fetchTimelineItems(r *http.Request, user *core.UserInfo, f components.TimelineFilters) ([]core.MediaItem, bool) {
	now := time.Now().UTC()
	from := now.AddDate(0, 0, -90) // 90 days of history

	// Fetch extra to determine if there are more items (for load more button).
	fetchLimit := int32(f.Limit + 1)
	items, err := h.App.MediaItems.List(r.Context(), user.ID, from, now, fetchLimit, int32(f.Offset))
	if err != nil {
		log.Printf("timeline page: list error: %v", err)
		return nil, false
	}

	// Apply client-side filters (platform, type, search).
	// TODO: Move filtering to the store/query layer for better performance.
	items = filterItems(items, f)

	hasMore := len(items) > f.Limit
	if hasMore {
		items = items[:f.Limit]
	}

	return items, hasMore
}

// filterItems applies platform, type, and search filters to the item list.
func filterItems(items []core.MediaItem, f components.TimelineFilters) []core.MediaItem {
	if f.Platform == "" && f.Type == "" && f.Search == "" {
		return items
	}

	search := strings.ToLower(f.Search)
	filtered := make([]core.MediaItem, 0, len(items))
	for _, item := range items {
		if f.Platform != "" && item.Platform != f.Platform {
			continue
		}
		if f.Type != "" && string(item.Type) != f.Type {
			continue
		}
		if search != "" {
			titleMatch := strings.Contains(strings.ToLower(item.Title), search)
			creatorMatch := strings.Contains(strings.ToLower(item.Creator), search)
			if !titleMatch && !creatorMatch {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	return filtered
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
