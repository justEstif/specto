package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/justestif/specto/internal/core"
)

// newTestServer creates an httptest.Server that responds to /videos requests
// with the given videosResponse. The handler validates the Authorization header.
func newTestServer(t *testing.T, resp videosResponse) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/videos" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-token")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestEnrichBasic(t *testing.T) {
	apiResp := videosResponse{
		Items: []videoResource{
			{
				ID: "abc123",
				Snippet: videoSnippet{
					Title:        "Go Concurrency Explained",
					ChannelTitle: "Tech Channel",
					PublishedAt:  "2025-01-15T10:00:00Z",
					Tags:         []string{"programming", "golang"},
					CategoryID:   "28",
					Description:  "A deep dive into Go concurrency patterns.",
					Thumbnails: videoThumbnails{
						Medium: &thumbnail{URL: "https://i.ytimg.com/vi/abc123/mqdefault.jpg"},
					},
				},
				ContentDetails: contentDetails{Duration: "PT12M34S"},
				Statistics:     videoStats{ViewCount: "150000", LikeCount: "5000"},
			},
		},
	}

	srv := newTestServer(t, apiResp)
	defer srv.Close()

	p := NewWithBaseURL(srv.URL)
	items := []core.MediaItem{
		{
			Platform:   "youtube",
			Type:       core.MediaVideo,
			Title:      "Watched Go Concurrency Explained",
			Creator:    "Tech Channel",
			ExternalID: "abc123",
		},
	}

	got, err := p.Enrich(context.Background(), core.Credentials{AccessToken: "test-token"}, items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d items, want 1", len(got))
	}

	item := got[0]

	// Title should be updated to canonical API version.
	if item.Title != "Go Concurrency Explained" {
		t.Errorf("Title = %q, want %q", item.Title, "Go Concurrency Explained")
	}
	if item.Creator != "Tech Channel" {
		t.Errorf("Creator = %q, want %q", item.Creator, "Tech Channel")
	}

	// Duration should be parsed.
	wantDuration := 12*time.Minute + 34*time.Second
	if item.Duration == nil {
		t.Fatal("Duration is nil, want non-nil")
	}
	if *item.Duration != wantDuration {
		t.Errorf("Duration = %v, want %v", *item.Duration, wantDuration)
	}

	// Tags: "programming" is a valid tag in the taxonomy.
	if !containsTag(item.Tags, "programming") {
		t.Errorf("Tags = %v, want to contain %q", item.Tags, "programming")
	}
	// CategoryID 28 -> "technology"
	if !containsTag(item.Tags, "technology") {
		t.Errorf("Tags = %v, want to contain %q", item.Tags, "technology")
	}

	// RawMetadata checks.
	if item.RawMetadata["view_count"] != int64(150000) {
		t.Errorf("RawMetadata[view_count] = %v, want 150000", item.RawMetadata["view_count"])
	}
	if item.RawMetadata["like_count"] != int64(5000) {
		t.Errorf("RawMetadata[like_count] = %v, want 5000", item.RawMetadata["like_count"])
	}
	if item.RawMetadata["category_id"] != "28" {
		t.Errorf("RawMetadata[category_id] = %v, want %q", item.RawMetadata["category_id"], "28")
	}
	if item.RawMetadata["category_name"] != "Science & Technology" {
		t.Errorf("RawMetadata[category_name] = %v, want %q", item.RawMetadata["category_name"], "Science & Technology")
	}
	if item.RawMetadata["thumbnail_url"] != "https://i.ytimg.com/vi/abc123/mqdefault.jpg" {
		t.Errorf("RawMetadata[thumbnail_url] = %v, want thumbnail URL", item.RawMetadata["thumbnail_url"])
	}
	if item.RawMetadata["published_at"] != "2025-01-15T10:00:00Z" {
		t.Errorf("RawMetadata[published_at] = %v, want published date", item.RawMetadata["published_at"])
	}
}

func TestEnrichBatchMultipleVideos(t *testing.T) {
	apiResp := videosResponse{
		Items: []videoResource{
			{
				ID:             "vid1",
				Snippet:        videoSnippet{Title: "Video One", ChannelTitle: "Channel A", CategoryID: "20"},
				ContentDetails: contentDetails{Duration: "PT5M"},
				Statistics:     videoStats{ViewCount: "1000"},
			},
			{
				ID:             "vid2",
				Snippet:        videoSnippet{Title: "Video Two", ChannelTitle: "Channel B", CategoryID: "27"},
				ContentDetails: contentDetails{Duration: "PT1H2M3S"},
				Statistics:     videoStats{ViewCount: "2000"},
			},
		},
	}

	srv := newTestServer(t, apiResp)
	defer srv.Close()

	p := NewWithBaseURL(srv.URL)
	items := []core.MediaItem{
		{Platform: "youtube", ExternalID: "vid1", Title: "Old Title 1"},
		{Platform: "youtube", ExternalID: "vid2", Title: "Old Title 2"},
	}

	got, err := p.Enrich(context.Background(), core.Credentials{AccessToken: "test-token"}, items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d items, want 2", len(got))
	}

	if got[0].Title != "Video One" {
		t.Errorf("items[0].Title = %q, want %q", got[0].Title, "Video One")
	}
	if got[1].Title != "Video Two" {
		t.Errorf("items[1].Title = %q, want %q", got[1].Title, "Video Two")
	}

	// Check durations.
	wantDur0 := 5 * time.Minute
	if got[0].Duration == nil || *got[0].Duration != wantDur0 {
		t.Errorf("items[0].Duration = %v, want %v", got[0].Duration, wantDur0)
	}
	wantDur1 := 1*time.Hour + 2*time.Minute + 3*time.Second
	if got[1].Duration == nil || *got[1].Duration != wantDur1 {
		t.Errorf("items[1].Duration = %v, want %v", got[1].Duration, wantDur1)
	}

	// Category 20 -> gaming
	if !containsTag(got[0].Tags, "gaming") {
		t.Errorf("items[0].Tags = %v, want to contain %q", got[0].Tags, "gaming")
	}
	// Category 27 -> education
	if !containsTag(got[1].Tags, "education") {
		t.Errorf("items[1].Tags = %v, want to contain %q", got[1].Tags, "education")
	}
}

func TestEnrichDeletedVideos(t *testing.T) {
	// API returns only vid1; vid2 is deleted/private (not in response).
	apiResp := videosResponse{
		Items: []videoResource{
			{
				ID:             "vid1",
				Snippet:        videoSnippet{Title: "Available Video"},
				ContentDetails: contentDetails{Duration: "PT3M"},
			},
		},
	}

	srv := newTestServer(t, apiResp)
	defer srv.Close()

	p := NewWithBaseURL(srv.URL)
	items := []core.MediaItem{
		{Platform: "youtube", ExternalID: "vid1", Title: "Old Title"},
		{Platform: "youtube", ExternalID: "vid2", Title: "Deleted Video Title"},
	}

	got, err := p.Enrich(context.Background(), core.Credentials{AccessToken: "test-token"}, items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d items, want 2", len(got))
	}

	// vid1 should be enriched.
	if got[0].Title != "Available Video" {
		t.Errorf("items[0].Title = %q, want %q", got[0].Title, "Available Video")
	}

	// vid2 should be unchanged (deleted/private).
	if got[1].Title != "Deleted Video Title" {
		t.Errorf("items[1].Title = %q, want %q (should be unchanged)", got[1].Title, "Deleted Video Title")
	}
	if got[1].Duration != nil {
		t.Errorf("items[1].Duration = %v, want nil (should be unchanged)", got[1].Duration)
	}
}

func TestEnrichNoExternalID(t *testing.T) {
	srv := newTestServer(t, videosResponse{})
	defer srv.Close()

	p := NewWithBaseURL(srv.URL)
	items := []core.MediaItem{
		{Platform: "youtube", Title: "No ID Video"},
	}

	got, err := p.Enrich(context.Background(), core.Credentials{AccessToken: "test-token"}, items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d items, want 1", len(got))
	}
	if got[0].Title != "No ID Video" {
		t.Errorf("Title = %q, want %q", got[0].Title, "No ID Video")
	}
}

func TestEnrichEmptyItems(t *testing.T) {
	srv := newTestServer(t, videosResponse{})
	defer srv.Close()

	p := NewWithBaseURL(srv.URL)
	got, err := p.Enrich(context.Background(), core.Credentials{AccessToken: "test-token"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestEnrichNoAccessToken(t *testing.T) {
	p := NewWithBaseURL("http://unused")
	_, err := p.Enrich(context.Background(), core.Credentials{}, []core.MediaItem{
		{Platform: "youtube", ExternalID: "vid1"},
	})

	if err == nil {
		t.Fatal("expected error for empty access token, got nil")
	}
	pe, ok := err.(*core.PluginError)
	if !ok {
		t.Fatalf("error type = %T, want *core.PluginError", err)
	}
	if pe.Code != core.ErrAuthExpired {
		t.Errorf("error code = %q, want %q", pe.Code, core.ErrAuthExpired)
	}
}

func TestEnrichHTTP401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := NewWithBaseURL(srv.URL)
	_, err := p.Enrich(context.Background(), core.Credentials{AccessToken: "bad-token"}, []core.MediaItem{
		{Platform: "youtube", ExternalID: "vid1"},
	})

	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
	pe, ok := err.(*core.PluginError)
	if !ok {
		t.Fatalf("error type = %T, want *core.PluginError", err)
	}
	if pe.Code != core.ErrAuthExpired {
		t.Errorf("error code = %q, want %q", pe.Code, core.ErrAuthExpired)
	}
}

func TestEnrichHTTP403(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	p := NewWithBaseURL(srv.URL)
	_, err := p.Enrich(context.Background(), core.Credentials{AccessToken: "test-token"}, []core.MediaItem{
		{Platform: "youtube", ExternalID: "vid1"},
	})

	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
	pe, ok := err.(*core.PluginError)
	if !ok {
		t.Fatalf("error type = %T, want *core.PluginError", err)
	}
	if pe.Code != core.ErrPermissionDenied {
		t.Errorf("error code = %q, want %q", pe.Code, core.ErrPermissionDenied)
	}
}

func TestEnrichHTTP429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	p := NewWithBaseURL(srv.URL)
	_, err := p.Enrich(context.Background(), core.Credentials{AccessToken: "test-token"}, []core.MediaItem{
		{Platform: "youtube", ExternalID: "vid1"},
	})

	if err == nil {
		t.Fatal("expected error for 429, got nil")
	}
	pe, ok := err.(*core.PluginError)
	if !ok {
		t.Fatalf("error type = %T, want *core.PluginError", err)
	}
	if pe.Code != core.ErrRateLimit {
		t.Errorf("error code = %q, want %q", pe.Code, core.ErrRateLimit)
	}
	if !pe.Retry {
		t.Error("Retry = false, want true")
	}
}

func TestEnrichHTTP500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	p := NewWithBaseURL(srv.URL)
	_, err := p.Enrich(context.Background(), core.Credentials{AccessToken: "test-token"}, []core.MediaItem{
		{Platform: "youtube", ExternalID: "vid1"},
	})

	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
	pe, ok := err.(*core.PluginError)
	if !ok {
		t.Fatalf("error type = %T, want *core.PluginError", err)
	}
	if pe.Code != core.ErrUpstream {
		t.Errorf("error code = %q, want %q", pe.Code, core.ErrUpstream)
	}
}

func TestEnrichInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{invalid json"))
	}))
	defer srv.Close()

	p := NewWithBaseURL(srv.URL)
	_, err := p.Enrich(context.Background(), core.Credentials{AccessToken: "test-token"}, []core.MediaItem{
		{Platform: "youtube", ExternalID: "vid1"},
	})

	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	pe, ok := err.(*core.PluginError)
	if !ok {
		t.Fatalf("error type = %T, want *core.PluginError", err)
	}
	if pe.Code != core.ErrInvalidData {
		t.Errorf("error code = %q, want %q", pe.Code, core.ErrInvalidData)
	}
}

func TestEnrichBatching(t *testing.T) {
	// Create 75 items to verify batching (should make 2 API calls: 50 + 25).
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		ids := r.URL.Query().Get("id")
		idList := splitNonEmpty(ids, ",")

		// Return a video resource for each requested ID.
		var items []videoResource
		for _, id := range idList {
			items = append(items, videoResource{
				ID:             id,
				Snippet:        videoSnippet{Title: "Title for " + id},
				ContentDetails: contentDetails{Duration: "PT1M"},
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(videosResponse{Items: items})
	}))
	defer srv.Close()

	p := NewWithBaseURL(srv.URL)
	items := make([]core.MediaItem, 75)
	for i := range items {
		items[i] = core.MediaItem{
			Platform:   "youtube",
			ExternalID: fmt.Sprintf("vid%d", i),
			Title:      fmt.Sprintf("Old Title %d", i),
		}
	}

	got, err := p.Enrich(context.Background(), core.Credentials{AccessToken: "test-token"}, items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 75 {
		t.Fatalf("got %d items, want 75", len(got))
	}

	// Should have made exactly 2 API calls.
	if requestCount != 2 {
		t.Errorf("API request count = %d, want 2", requestCount)
	}

	// Verify all items were enriched.
	for i, item := range got {
		wantTitle := fmt.Sprintf("Title for vid%d", i)
		if item.Title != wantTitle {
			t.Errorf("items[%d].Title = %q, want %q", i, item.Title, wantTitle)
		}
	}
}

func TestEnrichPreservesExistingTags(t *testing.T) {
	apiResp := videosResponse{
		Items: []videoResource{
			{
				ID:             "vid1",
				Snippet:        videoSnippet{Title: "Test", Tags: []string{"comedy", "gaming"}, CategoryID: "23"},
				ContentDetails: contentDetails{Duration: "PT5M"},
			},
		},
	}

	srv := newTestServer(t, apiResp)
	defer srv.Close()

	p := NewWithBaseURL(srv.URL)
	items := []core.MediaItem{
		{
			Platform:   "youtube",
			ExternalID: "vid1",
			Tags:       []string{"comedy", "funny"}, // comedy already exists
		},
	}

	got, err := p.Enrich(context.Background(), core.Credentials{AccessToken: "test-token"}, items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain comedy (no dup), funny (existing), gaming (new from tags).
	// Category 23 -> comedy, already present.
	if !containsTag(got[0].Tags, "comedy") {
		t.Errorf("Tags = %v, want to contain %q", got[0].Tags, "comedy")
	}
	if !containsTag(got[0].Tags, "funny") {
		t.Errorf("Tags = %v, want to contain %q", got[0].Tags, "funny")
	}
	if !containsTag(got[0].Tags, "gaming") {
		t.Errorf("Tags = %v, want to contain %q", got[0].Tags, "gaming")
	}

	// Verify no duplicate comedy.
	count := 0
	for _, tag := range got[0].Tags {
		if tag == "comedy" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("comedy appears %d times in Tags, want exactly 1", count)
	}
}

func TestEnrichHighThumbnailFallback(t *testing.T) {
	apiResp := videosResponse{
		Items: []videoResource{
			{
				ID: "vid1",
				Snippet: videoSnippet{
					Title: "Test",
					Thumbnails: videoThumbnails{
						High: &thumbnail{URL: "https://i.ytimg.com/vi/vid1/hqdefault.jpg"},
					},
				},
				ContentDetails: contentDetails{Duration: "PT1M"},
			},
		},
	}

	srv := newTestServer(t, apiResp)
	defer srv.Close()

	p := NewWithBaseURL(srv.URL)
	items := []core.MediaItem{
		{Platform: "youtube", ExternalID: "vid1"},
	}

	got, err := p.Enrich(context.Background(), core.Credentials{AccessToken: "test-token"}, items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got[0].RawMetadata["thumbnail_url"] != "https://i.ytimg.com/vi/vid1/hqdefault.jpg" {
		t.Errorf("thumbnail_url = %v, want high thumbnail URL", got[0].RawMetadata["thumbnail_url"])
	}
}

func TestEnrichNonYouTubeItemsIgnored(t *testing.T) {
	srv := newTestServer(t, videosResponse{})
	defer srv.Close()

	p := NewWithBaseURL(srv.URL)
	items := []core.MediaItem{
		{Platform: "spotify", ExternalID: "spotify:track:123", Title: "Spotify Track"},
		{Platform: "youtube", ExternalID: "", Title: "YouTube No ID"},
	}

	got, err := p.Enrich(context.Background(), core.Credentials{AccessToken: "test-token"}, items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d items, want 2", len(got))
	}

	// Both should be unchanged.
	if got[0].Title != "Spotify Track" {
		t.Errorf("items[0].Title = %q, want %q", got[0].Title, "Spotify Track")
	}
	if got[1].Title != "YouTube No ID" {
		t.Errorf("items[1].Title = %q, want %q", got[1].Title, "YouTube No ID")
	}
}

func TestEnrichDescriptionTruncation(t *testing.T) {
	longDesc := ""
	for i := 0; i < 600; i++ {
		longDesc += "x"
	}

	apiResp := videosResponse{
		Items: []videoResource{
			{
				ID:             "vid1",
				Snippet:        videoSnippet{Title: "Test", Description: longDesc},
				ContentDetails: contentDetails{Duration: "PT1M"},
			},
		},
	}

	srv := newTestServer(t, apiResp)
	defer srv.Close()

	p := NewWithBaseURL(srv.URL)
	items := []core.MediaItem{
		{Platform: "youtube", ExternalID: "vid1"},
	}

	got, err := p.Enrich(context.Background(), core.Credentials{AccessToken: "test-token"}, items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	desc := got[0].RawMetadata["description"].(string)
	if len(desc) > 503 { // 500 + "..."
		t.Errorf("description length = %d, want <= 503", len(desc))
	}
}

func TestEnrichDoesNotMutateOriginal(t *testing.T) {
	apiResp := videosResponse{
		Items: []videoResource{
			{
				ID:             "vid1",
				Snippet:        videoSnippet{Title: "New Title"},
				ContentDetails: contentDetails{Duration: "PT1M"},
			},
		},
	}

	srv := newTestServer(t, apiResp)
	defer srv.Close()

	p := NewWithBaseURL(srv.URL)
	original := []core.MediaItem{
		{Platform: "youtube", ExternalID: "vid1", Title: "Original Title"},
	}

	got, err := p.Enrich(context.Background(), core.Credentials{AccessToken: "test-token"}, original)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Original should be unchanged.
	if original[0].Title != "Original Title" {
		t.Errorf("original[0].Title = %q, want %q (should not be mutated)", original[0].Title, "Original Title")
	}
	if got[0].Title != "New Title" {
		t.Errorf("got[0].Title = %q, want %q", got[0].Title, "New Title")
	}
}

// --- ISO 8601 duration parsing tests ---

func TestParseISO8601Duration(t *testing.T) {
	tests := []struct {
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"PT12M34S", 12*time.Minute + 34*time.Second, false},
		{"PT1H", 1 * time.Hour, false},
		{"PT1H2M3S", 1*time.Hour + 2*time.Minute + 3*time.Second, false},
		{"PT5M", 5 * time.Minute, false},
		{"PT30S", 30 * time.Second, false},
		{"PT0S", 0, false},
		{"PT2H30M", 2*time.Hour + 30*time.Minute, false},
		{"", 0, true},
		{"P1D", 0, true},   // Days not supported
		{"12:34", 0, true}, // Not ISO 8601
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseISO8601Duration(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseISO8601Duration(%q) = %v, want error", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseISO8601Duration(%q) error = %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("parseISO8601Duration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// --- Category mapping tests ---

func TestCategoryToTag(t *testing.T) {
	tests := []struct {
		categoryID string
		want       string
	}{
		{"20", "gaming"},
		{"27", "education"},
		{"28", "technology"},
		{"23", "comedy"},
		{"17", "sports"},
		{"35", "documentary"},
		{"10", ""}, // Music — too generic
		{"24", ""}, // Entertainment — too generic
		{"99", ""}, // Unknown
		{"", ""},   // Empty
	}

	for _, tt := range tests {
		t.Run("category_"+tt.categoryID, func(t *testing.T) {
			got := categoryToTag(tt.categoryID)
			if got != tt.want {
				t.Errorf("categoryToTag(%q) = %q, want %q", tt.categoryID, got, tt.want)
			}
		})
	}
}

// --- Tag normalization tests ---

func TestNormalizeTags(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"rock", []string{"rock"}},
		{"Hip Hop", []string{"hip-hop", "hip hop"}},
		{"sci_fi", []string{"sci-fi", "sci_fi"}},
		{"  ", nil},
		{"", nil},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeTags(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("normalizeTags(%q) = %v, want %v", tt.input, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("normalizeTags(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestEnrichPluginName(t *testing.T) {
	p := NewWithEnrich()
	if got := p.Name(); got != "youtube" {
		t.Errorf("Name() = %q, want %q", got, "youtube")
	}
}

func TestEnrichPluginAuthType(t *testing.T) {
	p := NewWithEnrich()
	if got := p.AuthType(); got != core.AuthFileImport {
		t.Errorf("AuthType() = %v, want AuthFileImport", got)
	}
}

// --- Helpers ---

func containsTag(tags []string, target string) bool {
	for _, t := range tags {
		if t == target {
			return true
		}
	}
	return false
}

func splitNonEmpty(s, sep string) []string {
	if s == "" {
		return nil
	}
	parts := make([]string, 0)
	for _, p := range splitString(s, sep) {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func splitString(s, sep string) []string {
	result := make([]string, 0)
	for len(s) > 0 {
		idx := indexOf(s, sep)
		if idx < 0 {
			result = append(result, s)
			break
		}
		result = append(result, s[:idx])
		s = s[idx+len(sep):]
	}
	return result
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
