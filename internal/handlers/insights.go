package handlers

import (
	"net/http"

	"github.com/justestif/specto/internal/auth"
	"github.com/justestif/specto/internal/core"
)

// InsightsSummary handles GET /api/v1/insights/summary
// Returns top-level dashboard numbers.
func (h *Handler) InsightsSummary(w http.ResponseWriter, r *http.Request) {
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

	summary, err := h.App.Insights.GetSummary(r.Context(), user.ID, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to load summary")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"total_items":              summary.TotalItems,
			"total_time_spent_seconds": summary.TotalDurationSec,
			"top_platform":             summary.TopPlatform,
			"top_type":                 summary.TopMediaType,
		},
	})
}

// InsightsPlatformBreakdown handles GET /api/v1/insights/platform-breakdown
func (h *Handler) InsightsPlatformBreakdown(w http.ResponseWriter, r *http.Request) {
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

	entries, err := h.App.Insights.GetPlatformBreakdown(r.Context(), user.ID, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to load platform breakdown")
		return
	}

	data := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		data = append(data, map[string]any{
			"platform":               e.Platform,
			"type":                   e.MediaType,
			"count":                  e.Count,
			"total_duration_seconds": e.TotalDurationSec,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

// InsightsTags handles GET /api/v1/insights/tags
// Returns aggregate tag counts.
func (h *Handler) InsightsTags(w http.ResponseWriter, r *http.Request) {
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

	limit := parseIntParam(q.Get("limit"), 20, 1, 200)

	entries, err := h.App.Insights.GetTagDistribution(r.Context(), user.ID, from, to, int32(limit))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to load tag distribution")
		return
	}

	data := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		data = append(data, map[string]any{
			"name":     e.Name,
			"category": e.Category,
			"count":    e.Count,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

// InsightsTimeline handles GET /api/v1/insights/timeline
// Returns time-bucketed consumption data for charts.
func (h *Handler) InsightsTimeline(w http.ResponseWriter, r *http.Request) {
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

	bucketStr := q.Get("bucket")
	if bucketStr == "" {
		bucketStr = "day"
	}
	bucket := core.TimeBucket(bucketStr)

	timeline, err := h.App.Insights.GetTimeline(r.Context(), user.ID, bucket, from, to)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	data := make([]map[string]any, 0, len(timeline))
	for _, e := range timeline {
		data = append(data, map[string]any{
			"bucket_start":       e.Bucket,
			"count":              e.Count,
			"time_spent_seconds": e.TotalDurationSec,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}
