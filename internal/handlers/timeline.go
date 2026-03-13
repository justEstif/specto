package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/justestif/specto/internal/auth"
)

// Timeline handles GET /api/v1/timeline
// Returns paginated items for the dashboard timeline.
func (h *Handler) Timeline(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	q := r.URL.Query()

	// Parse date range with defaults
	from, to, err := parseDateRange(q.Get("from"), q.Get("to"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	// Parse pagination
	limit := parseIntParam(q.Get("limit"), 50, 1, 100)
	offset := parseIntParam(q.Get("offset"), 0, 0, 10000)

	items, err := h.App.MediaItems.List(r.Context(), user.ID, from, to, int32(limit), int32(offset))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to load timeline")
		return
	}

	// Build response items
	data := make([]map[string]any, 0, len(items))
	for _, item := range items {
		entry := map[string]any{
			"platform":    item.Platform,
			"type":        string(item.Type),
			"title":       item.Title,
			"creator":     item.Creator,
			"consumed_at": item.ConsumedAt,
			"external_id": item.ExternalID,
		}
		if item.URL != "" {
			entry["url"] = item.URL
		}
		if item.Duration != nil {
			entry["duration_seconds"] = int64(item.Duration.Seconds())
		}
		if item.TimeSpent != nil {
			entry["time_spent_seconds"] = int64(item.TimeSpent.Seconds())
		}
		if len(item.Tags) > 0 {
			entry["tags"] = item.Tags
		}
		data = append(data, entry)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": data,
		"meta": map[string]any{
			"limit":  limit,
			"offset": offset,
		},
	})
}

// --- helpers ---

// parseDateRange parses optional from/to RFC3339 strings.
// Defaults to the last 30 days if omitted.
func parseDateRange(fromStr, toStr string) (time.Time, time.Time, error) {
	now := time.Now().UTC()
	from := now.AddDate(0, 0, -30)
	to := now

	if fromStr != "" {
		parsed, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid 'from' timestamp: must be RFC3339 format")
		}
		from = parsed
	}

	if toStr != "" {
		parsed, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid 'to' timestamp: must be RFC3339 format")
		}
		to = parsed
	}

	if to.Before(from) {
		return time.Time{}, time.Time{}, fmt.Errorf("'to' must be after 'from'")
	}

	return from, to, nil
}

// parseIntParam parses an integer query parameter with bounds.
func parseIntParam(s string, defaultVal, min, max int) int {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < min {
		return defaultVal
	}
	if v > max {
		return max
	}
	return v
}
