package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/justestif/specto/components"
	"github.com/justestif/specto/internal/auth"
	"github.com/justestif/specto/internal/core"
)

// validInsightsTab returns the tab if valid, defaulting to "attention".
func validInsightsTab(tab string) components.InsightsTab {
	switch components.InsightsTab(tab) {
	case components.InsightsTabAttention, components.InsightsTabCrossover, components.InsightsTabObsessions:
		return components.InsightsTab(tab)
	default:
		return components.InsightsTabAttention
	}
}

// InsightsPageHandler renders GET /insights and GET /insights/{tab}.
func (h *Handler) InsightsPageHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	tab := validInsightsTab(chi.URLParam(r, "tab"))
	filters := parseInsightsFilters(r, tab)
	data := h.buildInsightsPageData(r, user, tab, filters)

	components.InsightsPage(data).Render(r.Context(), w)
}

// InsightsPartialHandler handles GET /partials/insights/{tab} — HTMX swap target.
func (h *Handler) InsightsPartialHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	tab := validInsightsTab(chi.URLParam(r, "tab"))
	filters := parseInsightsFilters(r, tab)
	data := h.buildInsightsPageData(r, user, tab, filters)

	components.InsightsTabContent(data).Render(r.Context(), w)
}

func (h *Handler) buildInsightsPageData(r *http.Request, user *core.UserInfo, tab components.InsightsTab, filters components.AttentionFilters) components.InsightsPageData {
	ctx := r.Context()
	from, to := rangeToTime(filters.Range)

	insightsFilter := core.InsightsFilter{
		Platform:  nilIfEmpty(filters.Platform),
		MediaType: nilIfEmpty(filters.Type),
	}

	addContext(r, "handler", "insights")
	addContext(r, "insights_tab", string(tab))
	addContext(r, "user_id", user.ID.String())

	data := components.InsightsPageData{
		User:      user,
		ActiveTab: tab,
		Filters:   filters,
	}

	switch tab {
	case components.InsightsTabCrossover:
		allDNA, err := h.App.Insights.GetCrossover(ctx, user.ID, from, to, 30, nil, insightsFilter)
		if err != nil {
			addContext(r, "crossover_error", err.Error())
		}
		genreDNA, err := h.App.Insights.GetCrossover(ctx, user.ID, from, to, 10, strPtr("genre"), insightsFilter)
		if err != nil {
			addContext(r, "crossover_genre_error", err.Error())
		}
		topicDNA, err := h.App.Insights.GetCrossover(ctx, user.ID, from, to, 10, strPtr("topic"), insightsFilter)
		if err != nil {
			addContext(r, "crossover_topic_error", err.Error())
		}
		moodDNA, err := h.App.Insights.GetCrossover(ctx, user.ID, from, to, 10, strPtr("mood"), insightsFilter)
		if err != nil {
			addContext(r, "crossover_mood_error", err.Error())
		}
		data.Crossover = &components.CrossoverData{
			User:    user,
			Filters: filters,
			AllDNA:  allDNA,
			Genres:  genreDNA,
			Topics:  topicDNA,
			Moods:   moodDNA,
		}

	case components.InsightsTabObsessions:
		rangeDuration := to.Sub(from)
		recentStart := to.Add(-rangeDuration / 4)

		spikes, err := h.App.Insights.GetTopicSpikes(ctx, user.ID, from, to, recentStart, 10, insightsFilter)
		if err != nil {
			addContext(r, "topic_spikes_error", err.Error())
		}
		selectedTag := r.URL.Query().Get("tag")
		timeSeries, err := h.App.Insights.GetTopicTimeSeries(ctx, user.ID, from, to, nilIfEmpty(selectedTag), nil, insightsFilter)
		if err != nil {
			addContext(r, "topic_time_series_error", err.Error())
		}
		data.Obsessions = &components.ObsessionsData{
			User:        user,
			Filters:     filters,
			Spikes:      spikes,
			TimeSeries:  timeSeries,
			SelectedTag: selectedTag,
		}

	default: // attention
		attentionByType, err := h.App.Insights.GetAttentionByType(ctx, user.ID, from, to, nilIfEmpty(filters.Platform))
		if err != nil {
			addContext(r, "attention_by_type_error", err.Error())
		}
		genres, err := h.App.Insights.GetTagDistributionByCategory(ctx, user.ID, from, to, 10, "genre", insightsFilter)
		if err != nil {
			addContext(r, "attention_genres_error", err.Error())
		}
		topics, err := h.App.Insights.GetTagDistributionByCategory(ctx, user.ID, from, to, 10, "topic", insightsFilter)
		if err != nil {
			addContext(r, "attention_topics_error", err.Error())
		}
		moods, err := h.App.Insights.GetTagDistributionByCategory(ctx, user.ID, from, to, 10, "mood", insightsFilter)
		if err != nil {
			addContext(r, "attention_moods_error", err.Error())
		}
		platforms, err := h.App.Insights.GetPlatformBreakdown(ctx, user.ID, from, to, insightsFilter)
		if err != nil {
			addContext(r, "attention_platforms_error", err.Error())
		}
		heatmapCells, err := h.App.Insights.GetConsumptionHeatmap(ctx, user.ID, from, to, insightsFilter)
		if err != nil {
			addContext(r, "heatmap_error", err.Error())
		}
		data.Attention = &components.AttentionData{
			User:            user,
			Filters:         filters,
			AttentionByType: attentionByType,
			Genres:          genres,
			Topics:          topics,
			Moods:           moods,
			Platforms:       platforms,
			HeatmapCells:    heatmapCells,
		}
	}

	return data
}

func parseInsightsFilters(r *http.Request, tab components.InsightsTab) components.AttentionFilters {
	q := r.URL.Query()
	rangeStr := q.Get("range")
	if rangeStr == "" {
		// Obsessions benefits from a wider default window.
		if tab == components.InsightsTabObsessions {
			rangeStr = "90d"
		} else {
			rangeStr = "30d"
		}
	}
	switch rangeStr {
	case "7d", "30d", "90d":
	default:
		rangeStr = "30d"
	}
	return components.AttentionFilters{
		Platform: q.Get("platform"),
		Type:     q.Get("type"),
		Range:    rangeStr,
	}
}
