package handlers

import (
	"net/http"
	"time"

	"github.com/justestif/specto/components"
	"github.com/justestif/specto/internal/auth"
	"github.com/justestif/specto/internal/core"
)

// Home renders the landing page for unauthenticated visitors, or the
// dashboard for authenticated users.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		components.Home(nil).Render(r.Context(), w)
		return
	}
	// Redirect new users to the plugins page on first login.
	if !user.Onboarded {
		http.Redirect(w, r, "/plugins", http.StatusSeeOther)
		return
	}
	filters := parseDashboardFilters(r)
	h.renderDashboard(w, r, user, filters, false)
}

// DashboardPartial handles GET /partials/dashboard — returns the dashboard
// content area (everything below the filter bar) for HTMX filter swaps.
func (h *Handler) DashboardPartial(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	filters := parseDashboardFilters(r)
	h.renderDashboard(w, r, user, filters, true)
}

// renderDashboard fetches all dashboard data and renders the page or partial.
func (h *Handler) renderDashboard(w http.ResponseWriter, r *http.Request, user *core.UserInfo, filters components.DashboardFilters, partial bool) {
	ctx := r.Context()
	from, to := rangeToTime(filters.Range)

	insightsFilter := core.InsightsFilter{
		Platform:  nilIfEmpty(filters.Platform),
		MediaType: nilIfEmpty(filters.Type),
	}

	addContext(r, "handler", "dashboard")
	addContext(r, "user_id", user.ID.String())

	summary, err := h.App.Insights.GetSummary(ctx, user.ID, from, to, insightsFilter)
	if err != nil {
		addContext(r, "dashboard_summary_error", err.Error())
		summary = &core.Summary{}
	}

	timeline, err := h.App.Insights.GetTimeline(ctx, user.ID, core.BucketDay, from, to, insightsFilter)
	if err != nil {
		addContext(r, "dashboard_timeline_error", err.Error())
	}

	// Recent items use ListFiltered when filters are active.
	var recentItems []core.MediaItem
	if filters.Platform != "" || filters.Type != "" {
		platform := nilIfEmpty(filters.Platform)
		mediaType := nilIfEmpty(filters.Type)
		recentItems, err = h.App.MediaItems.ListFiltered(ctx, user.ID, from, to, 5, 0, platform, mediaType, nil)
	} else {
		recentItems, err = h.App.MediaItems.List(ctx, user.ID, from, to, 5, 0)
	}
	if err != nil {
		addContext(r, "dashboard_items_error", err.Error())
	}

	tags, err := h.App.Insights.GetTagDistribution(ctx, user.ID, from, to, 5, insightsFilter)
	if err != nil {
		addContext(r, "dashboard_tags_error", err.Error())
	}

	platforms, err := h.App.Insights.GetPlatformBreakdown(ctx, user.ID, from, to, insightsFilter)
	if err != nil {
		addContext(r, "dashboard_platforms_error", err.Error())
	}

	// On This Day — only on unfiltered view (no platform/type filter)
	var onThisDayGroups []components.OnThisDayGroup
	if filters.Platform == "" && filters.Type == "" {
		otdItems, otdErr := h.App.MediaItems.OnThisDay(ctx, user.ID, 10)
		if otdErr != nil {
			addContext(r, "dashboard_on_this_day_error", otdErr.Error())
		} else {
			onThisDayGroups = groupOnThisDayByYear(otdItems)
		}
	}

	data := components.DashboardData{
		User:              user,
		Summary:           summary,
		Timeline:          timeline,
		RecentItems:       recentItems,
		Tags:              tags,
		PlatformBreakdown: platforms,
		ActiveRange:       filters.Range,
		Filters:           filters,
		OnThisDay:         onThisDayGroups,
		Platforms:         h.App.Registry.Platforms(),
	}

	if partial {
		components.DashboardContent(data).Render(ctx, w)
	} else {
		components.Dashboard(data).Render(ctx, w)
	}
}

// ActivityChartPartial handles GET /partials/activity-chart?range=7d|30d|90d.
// Returns only the chart HTML for HTMX swap.
func (h *Handler) ActivityChartPartial(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	rangeStr := r.URL.Query().Get("range")
	if rangeStr == "" {
		rangeStr = "30d"
	}
	from, to := rangeToTime(rangeStr)

	timeline, err := h.App.Insights.GetTimeline(r.Context(), user.ID, core.BucketDay, from, to, core.InsightsFilter{})
	if err != nil {
		addContext(r, "activity_chart_error", err.Error())
	}

	components.ActivityChart(timeline).Render(r.Context(), w)
}

// RecentItemsPartial handles GET /partials/timeline?offset=N&limit=N.
// Returns additional timeline rows for "show more" append.
func (h *Handler) RecentItemsPartial(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	q := r.URL.Query()
	limit := parseIntParam(q.Get("limit"), 5, 1, 50)
	offset := parseIntParam(q.Get("offset"), 0, 0, 10000)

	rangeStr := q.Get("range")
	if rangeStr == "" {
		rangeStr = "30d"
	}
	from, to := rangeToTime(rangeStr)

	platform := nilIfEmpty(q.Get("platform"))
	mediaType := nilIfEmpty(q.Get("type"))

	var items []core.MediaItem
	var err error
	if platform != nil || mediaType != nil {
		items, err = h.App.MediaItems.ListFiltered(r.Context(), user.ID, from, to, int32(limit), int32(offset), platform, mediaType, nil)
	} else {
		items, err = h.App.MediaItems.List(r.Context(), user.ID, from, to, int32(limit), int32(offset))
	}
	if err != nil {
		addContext(r, "recent_items_error", err.Error())
		return
	}

	for _, item := range items {
		components.TimelineRow(item).Render(r.Context(), w)
	}
}

// parseDashboardFilters extracts filter values from the request query.
func parseDashboardFilters(r *http.Request) components.DashboardFilters {
	q := r.URL.Query()
	rangeStr := q.Get("range")
	if rangeStr == "" {
		rangeStr = "30d"
	}
	// Validate range
	switch rangeStr {
	case "7d", "30d", "90d":
	default:
		rangeStr = "30d"
	}
	return components.DashboardFilters{
		Platform: q.Get("platform"),
		Type:     q.Get("type"),
		Range:    rangeStr,
	}
}

// rangeToTime converts "7d", "30d", "90d" to from/to time.Time values.
func rangeToTime(rangeStr string) (time.Time, time.Time) {
	now := time.Now().UTC()
	switch rangeStr {
	case "7d":
		return now.AddDate(0, 0, -7), now
	case "90d":
		return now.AddDate(0, 0, -90), now
	default:
		return now.AddDate(0, 0, -30), now
	}
}
