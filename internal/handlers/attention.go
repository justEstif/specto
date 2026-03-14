package handlers

import (
	"net/http"

	"github.com/justestif/specto/components"
	"github.com/justestif/specto/internal/auth"
	"github.com/justestif/specto/internal/core"
)

// AttentionPage renders GET /attention — the attention audit dashboard.
func (h *Handler) AttentionPage(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	filters := parseAttentionFilters(r)
	h.renderAttention(w, r, user, filters, false)
}

// AttentionPartial handles GET /partials/attention — HTMX swap target for filters.
func (h *Handler) AttentionPartial(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	filters := parseAttentionFilters(r)
	h.renderAttention(w, r, user, filters, true)
}

func (h *Handler) renderAttention(w http.ResponseWriter, r *http.Request, user *core.UserInfo, filters components.AttentionFilters, partial bool) {
	ctx := r.Context()
	from, to := rangeToTime(filters.Range)

	insightsFilter := core.InsightsFilter{
		Platform:  nilIfEmpty(filters.Platform),
		MediaType: nilIfEmpty(filters.Type),
	}

	addContext(r, "handler", "attention")
	addContext(r, "user_id", user.ID.String())

	// Attention by media type
	attentionByType, err := h.App.Insights.GetAttentionByType(ctx, user.ID, from, to, nilIfEmpty(filters.Platform))
	if err != nil {
		addContext(r, "attention_by_type_error", err.Error())
	}

	// Tag breakdowns by category
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

	// Platform breakdown
	platforms, err := h.App.Insights.GetPlatformBreakdown(ctx, user.ID, from, to, insightsFilter)
	if err != nil {
		addContext(r, "attention_platforms_error", err.Error())
	}

	// Consumption heatmap
	heatmapCells, err := h.App.Insights.GetConsumptionHeatmap(ctx, user.ID, from, to, insightsFilter)
	if err != nil {
		addContext(r, "heatmap_error", err.Error())
	}

	data := components.AttentionData{
		User:            user,
		Filters:         filters,
		AttentionByType: attentionByType,
		Genres:          genres,
		Topics:          topics,
		Moods:           moods,
		Platforms:       platforms,
		HeatmapCells:    heatmapCells,
	}

	if partial {
		components.AttentionContent(data).Render(ctx, w)
	} else {
		components.AttentionPage(data).Render(ctx, w)
	}
}

// InsightsAttentionByType handles GET /api/v1/insights/attention-by-type
func (h *Handler) InsightsAttentionByType(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	q := r.URL.Query()
	from, to, err := parseDateRange(q.Get("from"), q.Get("to"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	entries, err := h.App.Insights.GetAttentionByType(r.Context(), user.ID, from, to, nilIfEmpty(q.Get("platform")))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to load attention data")
		return
	}

	data := make([]attentionByTypeResponse, len(entries))
	for i, e := range entries {
		data[i] = attentionByTypeResponse{
			MediaType:        e.MediaType,
			Count:            e.Count,
			TotalTimeSpent:   e.TotalTimeSpent,
			TotalDurationSec: e.TotalDurationSec,
		}
	}

	writeJSON(w, http.StatusOK, dataResponse{Data: data})
}

// InsightsTagsByCategory handles GET /api/v1/insights/tags/{category}
func (h *Handler) InsightsTagsByCategory(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	q := r.URL.Query()
	from, to, err := parseDateRange(q.Get("from"), q.Get("to"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	category := q.Get("category")
	if category == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "category parameter required")
		return
	}

	limit := parseIntParam(q.Get("limit"), 20, 1, 200)
	filter := core.InsightsFilter{
		Platform:  nilIfEmpty(q.Get("platform")),
		MediaType: nilIfEmpty(q.Get("type")),
	}

	entries, err := h.App.Insights.GetTagDistributionByCategory(r.Context(), user.ID, from, to, int32(limit), category, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to load tag distribution")
		return
	}

	data := make([]tagDistributionResponse, len(entries))
	for i, e := range entries {
		data[i] = tagDistributionResponse{
			Name:     e.Name,
			Category: e.Category,
			Count:    e.Count,
		}
	}

	writeJSON(w, http.StatusOK, dataResponse{Data: data})
}

func parseAttentionFilters(r *http.Request) components.AttentionFilters {
	q := r.URL.Query()
	rangeStr := q.Get("range")
	if rangeStr == "" {
		rangeStr = "30d"
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

type attentionByTypeResponse struct {
	MediaType        string `json:"media_type"`
	Count            int64  `json:"count"`
	TotalTimeSpent   int64  `json:"total_time_spent_seconds"`
	TotalDurationSec int64  `json:"total_duration_seconds"`
}
