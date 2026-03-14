package handlers

import (
	"net/http"

	"github.com/justestif/specto/components"
	"github.com/justestif/specto/internal/auth"
	"github.com/justestif/specto/internal/core"
)

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

// InsightsTagsByCategory handles GET /api/v1/insights/tags-by-category
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
