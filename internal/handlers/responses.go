package handlers

import "time"

// --- API response types ---
// These structs define the JSON shape for API responses. Using structs
// instead of map[string]any catches field name inconsistencies at
// compile time and makes the API contract explicit.

// dataResponse wraps any payload in the standard {"data": ...} envelope.
type dataResponse struct {
	Data any `json:"data"`
}

// errorDetail is the standard error response body.
type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// --- Timeline ---

type timelineItemResponse struct {
	Platform     string    `json:"platform"`
	Type         string    `json:"type"`
	Title        string    `json:"title"`
	Creator      string    `json:"creator"`
	ConsumedAt   time.Time `json:"consumed_at"`
	ExternalID   string    `json:"external_id"`
	URL          string    `json:"url,omitempty"`
	DurationSec  *int64    `json:"duration_seconds,omitempty"`
	TimeSpentSec *int64    `json:"time_spent_seconds,omitempty"`
	Tags         []string  `json:"tags,omitempty"`
}

type timelineMeta struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type timelineResponse struct {
	Data []timelineItemResponse `json:"data"`
	Meta timelineMeta           `json:"meta"`
}

// --- Insights ---

type insightsSummaryResponse struct {
	TotalItems            int64  `json:"total_items"`
	TotalTimeSpentSeconds int64  `json:"total_time_spent_seconds"`
	TopPlatform           string `json:"top_platform"`
	TopType               string `json:"top_type"`
}

type platformBreakdownResponse struct {
	Platform             string `json:"platform"`
	Type                 string `json:"type"`
	Count                int64  `json:"count"`
	TotalDurationSeconds int64  `json:"total_duration_seconds"`
}

type tagDistributionResponse struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Count    int64  `json:"count"`
}

type timelineBucketResponse struct {
	BucketStart      time.Time `json:"bucket_start"`
	Count            int64     `json:"count"`
	TimeSpentSeconds int64     `json:"time_spent_seconds"`
}

// --- Sync ---

type syncResultResponse struct {
	Plugin       string     `json:"plugin"`
	Status       string     `json:"status"`
	ItemsAdded   int32      `json:"items_added"`
	ItemsSkipped int32      `json:"items_skipped"`
	ItemsUpdated int32      `json:"items_updated"`
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`
}

type syncHistoryEntryResponse struct {
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Status       string     `json:"status"`
	ItemsAdded   int32      `json:"items_added"`
	ItemsSkipped int32      `json:"items_skipped"`
	ItemsUpdated int32      `json:"items_updated"`
	ErrorCode    *string    `json:"error_code"`
	ErrorMessage *string    `json:"error_message"`
}
