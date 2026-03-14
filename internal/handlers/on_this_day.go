package handlers

import (
	"net/http"

	"github.com/justestif/specto/components"
	"github.com/justestif/specto/internal/auth"
	"github.com/justestif/specto/internal/core"
)

// OnThisDay handles GET /api/v1/insights/on-this-day
// Returns items consumed on this date in previous years.
func (h *Handler) OnThisDay(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	limit := parseIntParam(r.URL.Query().Get("limit"), 20, 1, 100)

	items, err := h.App.MediaItems.OnThisDay(r.Context(), user.ID, int32(limit))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to load on-this-day data")
		return
	}

	data := make([]onThisDayResponse, len(items))
	for i, otd := range items {
		var durSec *int64
		if otd.Item.Duration != nil {
			s := int64(otd.Item.Duration.Seconds())
			durSec = &s
		}
		data[i] = onThisDayResponse{
			Year:        otd.Year,
			Platform:    otd.Item.Platform,
			Type:        string(otd.Item.Type),
			Title:       otd.Item.Title,
			Creator:     otd.Item.Creator,
			ConsumedAt:  otd.Item.ConsumedAt.Format("2006-01-02T15:04:05Z"),
			URL:         otd.Item.URL,
			DurationSec: durSec,
		}
	}

	writeJSON(w, http.StatusOK, dataResponse{Data: data})
}

// OnThisDayPartial handles GET /partials/on-this-day
// Returns the "On This Day" card for the dashboard.
func (h *Handler) OnThisDayPartial(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	items, err := h.App.MediaItems.OnThisDay(r.Context(), user.ID, 10)
	if err != nil {
		addContext(r, "on_this_day_error", err.Error())
		items = nil
	}

	// Group by year
	groups := groupOnThisDayByYear(items)

	components.OnThisDayCard(groups).Render(r.Context(), w)
}

// groupOnThisDayByYear groups items by their consumption year.
func groupOnThisDayByYear(items []core.OnThisDayItem) []components.OnThisDayGroup {
	if len(items) == 0 {
		return nil
	}

	groupMap := make(map[int][]core.MediaItem)
	var years []int
	for _, otd := range items {
		if _, exists := groupMap[otd.Year]; !exists {
			years = append(years, otd.Year)
		}
		groupMap[otd.Year] = append(groupMap[otd.Year], otd.Item)
	}

	// Sort years descending (most recent first)
	for i := range years {
		for j := i + 1; j < len(years); j++ {
			if years[j] > years[i] {
				years[i], years[j] = years[j], years[i]
			}
		}
	}

	groups := make([]components.OnThisDayGroup, len(years))
	for i, year := range years {
		groups[i] = components.OnThisDayGroup{
			Year:  year,
			Items: groupMap[year],
		}
	}
	return groups
}

type onThisDayResponse struct {
	Year        int    `json:"year"`
	Platform    string `json:"platform"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Creator     string `json:"creator"`
	ConsumedAt  string `json:"consumed_at"`
	URL         string `json:"url,omitempty"`
	DurationSec *int64 `json:"duration_seconds,omitempty"`
}
