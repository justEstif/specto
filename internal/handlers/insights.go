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

	writeJSON(w, http.StatusOK, dataResponse{Data: insightsSummaryResponse{
		TotalItems:            summary.TotalItems,
		TotalTimeSpentSeconds: summary.TotalDurationSec,
		TopPlatform:           summary.TopPlatform,
		TopType:               summary.TopMediaType,
	}})
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

	data := make([]platformBreakdownResponse, 0, len(entries))
	for _, e := range entries {
		data = append(data, platformBreakdownResponse{
			Platform:             e.Platform,
			Type:                 e.MediaType,
			Count:                e.Count,
			TotalDurationSeconds: e.TotalDurationSec,
		})
	}

	writeJSON(w, http.StatusOK, dataResponse{Data: data})
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

	data := make([]tagDistributionResponse, 0, len(entries))
	for _, e := range entries {
		data = append(data, tagDistributionResponse{
			Name:     e.Name,
			Category: e.Category,
			Count:    e.Count,
		})
	}

	writeJSON(w, http.StatusOK, dataResponse{Data: data})
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

	data := make([]timelineBucketResponse, 0, len(timeline))
	for _, e := range timeline {
		data = append(data, timelineBucketResponse{
			BucketStart:      e.Bucket,
			Count:            e.Count,
			TimeSpentSeconds: e.TotalDurationSec,
		})
	}

	writeJSON(w, http.StatusOK, dataResponse{Data: data})
}
