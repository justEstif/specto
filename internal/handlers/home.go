package handlers

import (
	"log"
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
	h.renderDashboard(w, r, user, "30d")
}

// renderDashboard fetches all dashboard data and renders the page.
func (h *Handler) renderDashboard(w http.ResponseWriter, r *http.Request, user *core.UserInfo, rangeStr string) {
	ctx := r.Context()
	from, to := rangeToTime(rangeStr)

	summary, err := h.App.Insights.GetSummary(ctx, user.ID, from, to)
	if err != nil {
		log.Printf("dashboard: summary error: %v", err)
		summary = &core.Summary{}
	}

	timeline, err := h.App.Insights.GetTimeline(ctx, user.ID, core.BucketDay, from, to)
	if err != nil {
		log.Printf("dashboard: timeline error: %v", err)
	}

	recentItems, err := h.App.MediaItems.List(ctx, user.ID, from, to, 5, 0)
	if err != nil {
		log.Printf("dashboard: recent items error: %v", err)
	}

	tags, err := h.App.Insights.GetTagDistribution(ctx, user.ID, from, to, 5)
	if err != nil {
		log.Printf("dashboard: tags error: %v", err)
	}

	platforms, err := h.App.Insights.GetPlatformBreakdown(ctx, user.ID, from, to)
	if err != nil {
		log.Printf("dashboard: platforms error: %v", err)
	}

	data := components.DashboardData{
		User:              user,
		Summary:           summary,
		Timeline:          timeline,
		RecentItems:       recentItems,
		Tags:              tags,
		PlatformBreakdown: platforms,
		ActiveRange:       rangeStr,
	}
	components.Dashboard(data).Render(ctx, w)
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

	timeline, err := h.App.Insights.GetTimeline(r.Context(), user.ID, core.BucketDay, from, to)
	if err != nil {
		log.Printf("activity chart partial: %v", err)
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

	now := time.Now().UTC()
	from := now.AddDate(0, 0, -90)

	items, err := h.App.MediaItems.List(r.Context(), user.ID, from, now, int32(limit), int32(offset))
	if err != nil {
		log.Printf("recent items partial: %v", err)
		return
	}

	for _, item := range items {
		components.TimelineRow(item).Render(r.Context(), w)
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
