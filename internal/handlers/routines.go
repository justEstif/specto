package handlers

import (
	"net/http"

	"github.com/justestif/specto/internal/auth"
	"github.com/justestif/specto/internal/core"
)

// InsightsHeatmap handles GET /api/v1/insights/heatmap
func (h *Handler) InsightsHeatmap(w http.ResponseWriter, r *http.Request) {
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

	filter := core.InsightsFilter{
		Platform:  nilIfEmpty(q.Get("platform")),
		MediaType: nilIfEmpty(q.Get("type")),
	}

	cells, err := h.App.Insights.GetConsumptionHeatmap(r.Context(), user.ID, from, to, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to load heatmap data")
		return
	}

	type heatmapResponse struct {
		DayOfWeek int   `json:"day_of_week"`
		HourOfDay int   `json:"hour_of_day"`
		Count     int64 `json:"count"`
	}

	data := make([]heatmapResponse, len(cells))
	for i, c := range cells {
		data[i] = heatmapResponse{
			DayOfWeek: c.DayOfWeek,
			HourOfDay: c.HourOfDay,
			Count:     c.Count,
		}
	}

	writeJSON(w, http.StatusOK, dataResponse{Data: data})
}
