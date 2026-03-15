package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/csrf"

	"github.com/justestif/specto/components"
	"github.com/justestif/specto/internal/auth"
	"github.com/justestif/specto/internal/core"
)

// validTimelineTab returns the tab if valid, defaulting to "overview".
func validTimelineTab(tab string) components.TimelineTab {
	switch components.TimelineTab(tab) {
	case components.TimelineTabOverview, components.TimelineTabActivity, components.TimelineTabEras:
		return components.TimelineTab(tab)
	default:
		return components.TimelineTabOverview
	}
}

// TimelinePage renders GET /timeline and GET /timeline/activity — the full page.
func (h *Handler) TimelinePage(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	tab := validTimelineTab(chi.URLParam(r, "tab"))
	data := h.buildTimelinePageData(r, user, tab)

	components.TimelinePage(data).Render(r.Context(), w)
}

// TimelineTabPartial handles GET /partials/timeline/{tab} — HTMX swap target.
func (h *Handler) TimelineTabPartial(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	tab := validTimelineTab(chi.URLParam(r, "tab"))
	data := h.buildTimelinePageData(r, user, tab)

	components.TimelineTabContent(data).Render(r.Context(), w)
}

// TimelinePagePartial handles GET /partials/timeline-page — returns
// just the item list for HTMX filter/pagination swaps within the activity tab.
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

func (h *Handler) buildTimelinePageData(r *http.Request, user *core.UserInfo, tab components.TimelineTab) components.TimelinePageData {
	data := components.TimelinePageData{
		User:      user,
		ActiveTab: tab,
		Platforms: h.App.Registry.Platforms(),
	}

	switch tab {
	case components.TimelineTabActivity:
		filters := parseTimelineFilters(r)
		items, hasMore := h.fetchTimelineItems(r, user, filters)
		data.Items = items
		data.Filters = filters
		data.HasMore = hasMore

	case components.TimelineTabEras:
		data.Eras = h.fetchErasData(r, user)

	default: // overview
		data.Scorecard = h.fetchScorecardData(r, user)
	}

	return data
}

// fetchScorecardData loads aggregated consumption data for the scorecard.
func (h *Handler) fetchScorecardData(r *http.Request, user *core.UserInfo) *components.ScorecardData {
	ctx := r.Context()
	now := time.Now().UTC()
	from := now.AddDate(0, 0, -90)
	noFilter := core.InsightsFilter{}

	addContext(r, "handler", "timeline_scorecard")

	// Get summary (total items, duration, top platform/type)
	summary, err := h.App.Insights.GetSummary(ctx, user.ID, from, now, noFilter)
	if err != nil {
		addContext(r, "scorecard_summary_error", err.Error())
		return nil
	}

	if summary.TotalItems == 0 {
		return nil
	}

	// Get platform breakdown to count unique platforms and types
	breakdown, err := h.App.Insights.GetPlatformBreakdown(ctx, user.ID, from, now, noFilter)
	if err != nil {
		addContext(r, "scorecard_breakdown_error", err.Error())
	}
	platforms := make(map[string]bool)
	mediaTypes := make(map[string]bool)
	for _, entry := range breakdown {
		platforms[entry.Platform] = true
		mediaTypes[entry.MediaType] = true
	}

	// Get top tags by category
	genres, err := h.App.Insights.GetTagDistributionByCategory(ctx, user.ID, from, now, 3, "genre", noFilter)
	if err != nil {
		addContext(r, "scorecard_genres_error", err.Error())
	}
	moods, err := h.App.Insights.GetTagDistributionByCategory(ctx, user.ID, from, now, 3, "mood", noFilter)
	if err != nil {
		addContext(r, "scorecard_moods_error", err.Error())
	}
	topics, err := h.App.Insights.GetTagDistributionByCategory(ctx, user.ID, from, now, 3, "topic", noFilter)
	if err != nil {
		addContext(r, "scorecard_topics_error", err.Error())
	}

	// Get trending spikes
	recentStart := now.AddDate(0, 0, -90/4) // recent quarter
	spikes, err := h.App.Insights.GetTopicSpikes(ctx, user.ID, from, now, recentStart, 3, noFilter)
	if err != nil {
		addContext(r, "scorecard_spikes_error", err.Error())
	}

	return &components.ScorecardData{
		TotalItems:     summary.TotalItems,
		TotalHours:     float64(summary.TotalDurationSec) / 3600.0,
		PlatformCount:  len(platforms),
		MediaTypeCount: len(mediaTypes),
		TopGenres:      genres,
		TopMoods:       moods,
		TopTopics:      topics,
		Spikes:         spikes,
	}
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
		addContext(r, "timeline_list_error", err.Error())
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

// fetchErasData loads detected eras for the eras tab, grouped by media type.
func (h *Handler) fetchErasData(r *http.Request, user *core.UserInfo) *components.ErasData {
	ctx := r.Context()
	addContext(r, "handler", "timeline_eras")

	musicType := string(core.MediaMusic)
	videoType := string(core.MediaVideo)

	musicEras, err := h.App.Eras.List(ctx, user.ID, &musicType)
	if err != nil {
		addContext(r, "eras_music_error", err.Error())
	}

	videoEras, err := h.App.Eras.List(ctx, user.ID, &videoType)
	if err != nil {
		addContext(r, "eras_video_error", err.Error())
	}

	// Load tags for each era.
	for i := range musicEras {
		tags, err := h.App.Eras.GetTags(ctx, musicEras[i].ID)
		if err != nil {
			addContext(r, "eras_tags_error", err.Error())
			continue
		}
		musicEras[i].Tags = tags
	}
	for i := range videoEras {
		tags, err := h.App.Eras.GetTags(ctx, videoEras[i].ID)
		if err != nil {
			addContext(r, "eras_tags_error", err.Error())
			continue
		}
		videoEras[i].Tags = tags
	}

	if len(musicEras) == 0 && len(videoEras) == 0 {
		return nil
	}

	return &components.ErasData{
		MusicEras: musicEras,
		VideoEras: videoEras,
		CSRFToken: csrf.Token(r),
	}
}
