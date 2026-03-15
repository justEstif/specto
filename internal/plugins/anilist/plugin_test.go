package anilist

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/justestif/specto/internal/core"
)

// ── Test fixtures ───────────────────────────────────────────────────────

func viewerJSON(userID int) string {
	return fmt.Sprintf(`{"data":{"Viewer":{"id":%d}}}`, userID)
}

func sampleAnimeListJSON() string {
	return `{
		"data": {
			"MediaListCollection": {
				"lists": [{
					"entries": [
						{
							"id": 1001,
							"mediaId": 20,
							"status": "COMPLETED",
							"score": 9.0,
							"progress": 24,
							"progressVolumes": 0,
							"updatedAt": 1710000000,
							"completedAt": {"year": 2024, "month": 3, "day": 10},
							"startedAt": {"year": 2024, "month": 1, "day": 15},
							"media": {
								"id": 20,
								"title": {"romaji": "Naruto", "english": "Naruto", "native": "ナルト"},
								"format": "TV",
								"episodes": 220,
								"chapters": 0,
								"volumes": 0,
								"genres": ["Action", "Adventure"],
								"averageScore": 79,
								"siteUrl": "https://anilist.co/anime/20"
							}
						},
						{
							"id": 1002,
							"mediaId": 21,
							"status": "WATCHING",
							"score": 8.5,
							"progress": 12,
							"progressVolumes": 0,
							"updatedAt": 1710100000,
							"completedAt": {"year": null, "month": null, "day": null},
							"startedAt": {"year": 2024, "month": 2, "day": 1},
							"media": {
								"id": 21,
								"title": {"romaji": "Shingeki no Kyojin", "english": "Attack on Titan", "native": "進撃の巨人"},
								"format": "TV",
								"episodes": 75,
								"chapters": 0,
								"volumes": 0,
								"genres": ["Action", "Drama"],
								"averageScore": 84,
								"siteUrl": "https://anilist.co/anime/21"
							}
						}
					]
				}]
			}
		}
	}`
}

func sampleMangaListJSON() string {
	return `{
		"data": {
			"MediaListCollection": {
				"lists": [{
					"entries": [
						{
							"id": 2001,
							"mediaId": 30,
							"status": "READING",
							"score": 10.0,
							"progress": 300,
							"progressVolumes": 30,
							"updatedAt": 1710200000,
							"completedAt": {"year": null, "month": null, "day": null},
							"startedAt": {"year": 2023, "month": 6, "day": 1},
							"media": {
								"id": 30,
								"title": {"romaji": "One Piece", "english": "One Piece", "native": "ワンピース"},
								"format": "MANGA",
								"episodes": 0,
								"chapters": 1100,
								"volumes": 106,
								"genres": ["Action", "Adventure", "Comedy"],
								"averageScore": 88,
								"siteUrl": "https://anilist.co/manga/30"
							}
						}
					]
				}]
			}
		}
	}`
}

func emptyListJSON() string {
	return `{"data":{"MediaListCollection":{"lists":[]}}}`
}

// newSyncServer returns a test server that responds to Viewer and
// MediaListCollection queries. animeJSON/mangaJSON are the raw responses
// for each media type.
func newSyncServer(t *testing.T, animeJSON, mangaJSON string) *httptest.Server {
	t.Helper()
	callCount := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-token")
		}

		var req graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decoding request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		// First call is always Viewer query.
		if callCount == 0 {
			callCount++
			fmt.Fprint(w, viewerJSON(12345))
			return
		}

		// Subsequent calls are media list queries.
		callCount++
		mediaType, _ := req.Variables["type"].(string)
		switch mediaType {
		case "ANIME":
			fmt.Fprint(w, animeJSON)
		case "MANGA":
			fmt.Fprint(w, mangaJSON)
		default:
			t.Errorf("unexpected media type: %q", mediaType)
			fmt.Fprint(w, emptyListJSON())
		}
	}))
}

// ── Metadata tests ──────────────────────────────────────────────────────

func TestAPIPluginName(t *testing.T) {
	p := NewAPI()
	if got := p.Name(); got != "anilist-api" {
		t.Errorf("Name() = %q, want %q", got, "anilist-api")
	}
}

func TestAPIPluginAuthType(t *testing.T) {
	p := NewAPI()
	if got := p.AuthType(); got != core.AuthOAuth {
		t.Errorf("AuthType() = %v, want AuthOAuth (%v)", got, core.AuthOAuth)
	}
}

func TestAPIPluginAuthConfig(t *testing.T) {
	p := NewAPI()
	cfg := p.AuthConfig()
	if cfg == nil {
		t.Fatal("AuthConfig() returned nil")
	}
	if cfg.ProviderName != "AniList" {
		t.Errorf("ProviderName = %q, want %q", cfg.ProviderName, "AniList")
	}
	if cfg.AuthURL != "https://anilist.co/api/v2/oauth/authorize" {
		t.Errorf("AuthURL = %q", cfg.AuthURL)
	}
	if cfg.TokenURL != "https://anilist.co/api/v2/oauth/token" {
		t.Errorf("TokenURL = %q", cfg.TokenURL)
	}
	if cfg.Scopes != nil {
		t.Errorf("Scopes = %v, want nil (AniList has no scopes)", cfg.Scopes)
	}
}

// ── Sync success tests ──────────────────────────────────────────────────

func TestAPISyncSuccess(t *testing.T) {
	srv := newSyncServer(t, sampleAnimeListJSON(), sampleMangaListJSON())
	defer srv.Close()

	p := NewAPIWithEndpoint(srv.URL)
	result := p.Sync(context.Background(), core.Credentials{AccessToken: "test-token"}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	// 2 anime + 1 manga = 3 items.
	if len(result.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result.Items))
	}

	// Verify first anime item.
	anime1 := result.Items[0]
	if anime1.Platform != "anilist" {
		t.Errorf("Platform = %q, want %q", anime1.Platform, "anilist")
	}
	if anime1.Type != core.MediaVideo {
		t.Errorf("Type = %q, want %q", anime1.Type, core.MediaVideo)
	}
	if anime1.Title != "Naruto" {
		t.Errorf("Title = %q, want %q", anime1.Title, "Naruto")
	}
	if anime1.ExternalID != "20" {
		t.Errorf("ExternalID = %q, want %q", anime1.ExternalID, "20")
	}
	if anime1.URL != "https://anilist.co/anime/20" {
		t.Errorf("URL = %q", anime1.URL)
	}
	// completedAt is set → ConsumedAt should be 2024-03-10
	wantTime := time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC)
	if !anime1.ConsumedAt.Equal(wantTime) {
		t.Errorf("ConsumedAt = %v, want %v", anime1.ConsumedAt, wantTime)
	}
	// Check raw metadata.
	if anime1.RawMetadata["status"] != "COMPLETED" {
		t.Errorf("RawMetadata[status] = %v", anime1.RawMetadata["status"])
	}
	if anime1.RawMetadata["score"] != 9.0 {
		t.Errorf("RawMetadata[score] = %v", anime1.RawMetadata["score"])
	}
	if anime1.RawMetadata["progress"] != 24 {
		t.Errorf("RawMetadata[progress] = %v", anime1.RawMetadata["progress"])
	}
	if anime1.RawMetadata["episodes"] != 220 {
		t.Errorf("RawMetadata[episodes] = %v", anime1.RawMetadata["episodes"])
	}
	if anime1.RawMetadata["average_score"] != 79 {
		t.Errorf("RawMetadata[average_score] = %v", anime1.RawMetadata["average_score"])
	}
	if anime1.RawMetadata["started_at"] != "2024-01-15" {
		t.Errorf("RawMetadata[started_at] = %v", anime1.RawMetadata["started_at"])
	}

	// Verify second anime item (no completedAt → falls back to updatedAt).
	anime2 := result.Items[1]
	if anime2.Title != "Attack on Titan" {
		t.Errorf("Title = %q, want %q", anime2.Title, "Attack on Titan")
	}
	wantTime2 := time.Unix(1710100000, 0).UTC()
	if !anime2.ConsumedAt.Equal(wantTime2) {
		t.Errorf("ConsumedAt = %v, want %v (from updatedAt)", anime2.ConsumedAt, wantTime2)
	}

	// Verify manga item.
	manga := result.Items[2]
	if manga.Type != core.MediaBook {
		t.Errorf("manga Type = %q, want %q", manga.Type, core.MediaBook)
	}
	if manga.Title != "One Piece" {
		t.Errorf("manga Title = %q, want %q", manga.Title, "One Piece")
	}
	if manga.ExternalID != "30" {
		t.Errorf("manga ExternalID = %q, want %q", manga.ExternalID, "30")
	}
	if manga.RawMetadata["chapters"] != 1100 {
		t.Errorf("RawMetadata[chapters] = %v", manga.RawMetadata["chapters"])
	}
	if manga.RawMetadata["volumes"] != 106 {
		t.Errorf("RawMetadata[volumes] = %v", manga.RawMetadata["volumes"])
	}
	if manga.RawMetadata["progress_volumes"] != 30 {
		t.Errorf("RawMetadata[progress_volumes] = %v", manga.RawMetadata["progress_volumes"])
	}

	// Cursor should be the latest updatedAt across all entries (1710200000).
	if result.NextCursor != "1710200000" {
		t.Errorf("NextCursor = %q, want %q", result.NextCursor, "1710200000")
	}

	// AniList returns full lists.
	if result.HasMore {
		t.Error("HasMore = true, want false")
	}
}

// ── Title preference tests ──────────────────────────────────────────────

func TestTitlePreferenceEnglish(t *testing.T) {
	got := preferredTitle("Attack on Titan", "Shingeki no Kyojin", "進撃の巨人")
	if got != "Attack on Titan" {
		t.Errorf("preferredTitle() = %q, want %q", got, "Attack on Titan")
	}
}

func TestTitlePreferenceRomaji(t *testing.T) {
	got := preferredTitle("", "Shingeki no Kyojin", "進撃の巨人")
	if got != "Shingeki no Kyojin" {
		t.Errorf("preferredTitle() = %q, want %q", got, "Shingeki no Kyojin")
	}
}

func TestTitlePreferenceNative(t *testing.T) {
	got := preferredTitle("", "", "進撃の巨人")
	if got != "進撃の巨人" {
		t.Errorf("preferredTitle() = %q, want %q", got, "進撃の巨人")
	}
}

// ── Cursor / incremental sync ───────────────────────────────────────────

func TestAPISyncWithCursor(t *testing.T) {
	// Set cursor to 1710050000 — only entries with updatedAt > cursor
	// should be included.
	// Anime entries: updatedAt 1710000000 (excluded), 1710100000 (included)
	// Manga entries: updatedAt 1710200000 (included)
	srv := newSyncServer(t, sampleAnimeListJSON(), sampleMangaListJSON())
	defer srv.Close()

	p := NewAPIWithEndpoint(srv.URL)
	result := p.Sync(context.Background(), core.Credentials{AccessToken: "test-token"}, "1710050000")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	// Only 2 items should pass the cursor filter.
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items (cursor filtered), got %d", len(result.Items))
	}

	// Verify filtered items: Attack on Titan (anime) and One Piece (manga).
	if result.Items[0].Title != "Attack on Titan" {
		t.Errorf("Items[0].Title = %q, want %q", result.Items[0].Title, "Attack on Titan")
	}
	if result.Items[1].Title != "One Piece" {
		t.Errorf("Items[1].Title = %q, want %q", result.Items[1].Title, "One Piece")
	}

	// Cursor should be latest: 1710200000.
	if result.NextCursor != "1710200000" {
		t.Errorf("NextCursor = %q, want %q", result.NextCursor, "1710200000")
	}
}

// ── Error handling tests ────────────────────────────────────────────────

func TestAPISyncNoAccessToken(t *testing.T) {
	p := NewAPI()
	result := p.Sync(context.Background(), core.Credentials{}, "")

	if result.Err == nil {
		t.Fatal("expected error for empty access token")
	}
	if result.Err.Code != core.ErrAuthExpired {
		t.Errorf("error code = %q, want %q", result.Err.Code, core.ErrAuthExpired)
	}
}

func TestAPISyncAuthExpired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"errors":[{"message":"Invalid token","status":401}]}`)
	}))
	defer srv.Close()

	p := NewAPIWithEndpoint(srv.URL)
	result := p.Sync(context.Background(), core.Credentials{AccessToken: "expired-token"}, "")

	if result.Err == nil {
		t.Fatal("expected error for 401 response")
	}
	if result.Err.Code != core.ErrAuthExpired {
		t.Errorf("error code = %q, want %q", result.Err.Code, core.ErrAuthExpired)
	}
}

func TestAPISyncRateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "90")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	p := NewAPIWithEndpoint(srv.URL)
	result := p.Sync(context.Background(), core.Credentials{AccessToken: "tok"}, "")

	if result.Err == nil {
		t.Fatal("expected error for 429 response")
	}
	if result.Err.Code != core.ErrRateLimit {
		t.Errorf("error code = %q, want %q", result.Err.Code, core.ErrRateLimit)
	}
	if !result.Err.Retry {
		t.Error("Retry = false, want true")
	}
	if result.Err.After != 90*time.Second {
		t.Errorf("After = %v, want %v", result.Err.After, 90*time.Second)
	}
}

func TestAPISyncRateLimitDefaultRetryAfter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No Retry-After header.
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	p := NewAPIWithEndpoint(srv.URL)
	result := p.Sync(context.Background(), core.Credentials{AccessToken: "tok"}, "")

	if result.Err == nil {
		t.Fatal("expected error for 429 response")
	}
	// Should default to 60 seconds.
	if result.Err.After != 60*time.Second {
		t.Errorf("After = %v, want %v (default)", result.Err.After, 60*time.Second)
	}
}

func TestAPISyncServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal server error")
	}))
	defer srv.Close()

	p := NewAPIWithEndpoint(srv.URL)
	result := p.Sync(context.Background(), core.Credentials{AccessToken: "tok"}, "")

	if result.Err == nil {
		t.Fatal("expected error for 500 response")
	}
	if result.Err.Code != core.ErrUpstream {
		t.Errorf("error code = %q, want %q", result.Err.Code, core.ErrUpstream)
	}
}

func TestAPISyncInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{not valid json}`)
	}))
	defer srv.Close()

	p := NewAPIWithEndpoint(srv.URL)
	result := p.Sync(context.Background(), core.Credentials{AccessToken: "tok"}, "")

	if result.Err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if result.Err.Code != core.ErrInvalidData {
		t.Errorf("error code = %q, want %q", result.Err.Code, core.ErrInvalidData)
	}
}

func TestAPISyncEmptyLists(t *testing.T) {
	srv := newSyncServer(t, emptyListJSON(), emptyListJSON())
	defer srv.Close()

	p := NewAPIWithEndpoint(srv.URL)
	result := p.Sync(context.Background(), core.Credentials{AccessToken: "test-token"}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
	if result.NextCursor != "" {
		t.Errorf("NextCursor = %q, want empty", result.NextCursor)
	}
	if result.HasMore {
		t.Error("HasMore = true, want false")
	}
}

// ── Enrich passthrough test ─────────────────────────────────────────────

func TestAPIEnrichPassthrough(t *testing.T) {
	p := NewAPI()
	input := []core.MediaItem{
		{Platform: "anilist", Title: "Naruto", Type: core.MediaVideo},
	}

	got, err := p.Enrich(context.Background(), core.Credentials{}, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 item, got %d", len(got))
	}
	if got[0].Title != "Naruto" {
		t.Errorf("Title = %q, want %q", got[0].Title, "Naruto")
	}
}

// ── FuzzyDate tests ─────────────────────────────────────────────────────

func TestFuzzyDateIsSet(t *testing.T) {
	year := 2024
	zero := 0
	tests := []struct {
		name string
		date fuzzyDate
		want bool
	}{
		{"full date", fuzzyDate{Year: &year, Month: intPtr(3), Day: intPtr(10)}, true},
		{"year only", fuzzyDate{Year: &year}, true},
		{"nil year", fuzzyDate{}, false},
		{"zero year", fuzzyDate{Year: &zero}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.date.isSet(); got != tt.want {
				t.Errorf("isSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFuzzyDateToTime(t *testing.T) {
	year := 2024

	// Full date.
	d := fuzzyDate{Year: &year, Month: intPtr(3), Day: intPtr(10)}
	want := time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC)
	if got := d.toTime(); !got.Equal(want) {
		t.Errorf("toTime() = %v, want %v", got, want)
	}

	// Year only — month/day default to 1.
	d2 := fuzzyDate{Year: &year}
	want2 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := d2.toTime(); !got.Equal(want2) {
		t.Errorf("toTime() year-only = %v, want %v", got, want2)
	}

	// Not set — returns zero time.
	d3 := fuzzyDate{}
	if got := d3.toTime(); !got.IsZero() {
		t.Errorf("toTime() nil = %v, want zero", got)
	}
}

// ── parseRetryAfter for plugin ──────────────────────────────────────────

func TestPluginParseRetryAfter(t *testing.T) {
	tests := []struct {
		header string
		want   time.Duration
	}{
		{"90", 90 * time.Second},
		{"1", 1 * time.Second},
		{"", 60 * time.Second},
		{"abc", 60 * time.Second},
		{"-5", 60 * time.Second},
		{"0", 60 * time.Second},
	}

	for _, tt := range tests {
		got := parseRetryAfter(tt.header)
		if got != tt.want {
			t.Errorf("parseRetryAfter(%q) = %v, want %v", tt.header, got, tt.want)
		}
	}
}

// ── GraphQL error on media list query ───────────────────────────────────

func TestAPISyncGraphQLErrorOnMediaList(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		callCount++
		if callCount == 1 {
			// Viewer succeeds.
			fmt.Fprint(w, viewerJSON(123))
			return
		}
		// MediaList returns a GraphQL error.
		fmt.Fprint(w, `{"errors":[{"message":"server error","status":500}]}`)
	}))
	defer srv.Close()

	p := NewAPIWithEndpoint(srv.URL)
	result := p.Sync(context.Background(), core.Credentials{AccessToken: "test-token"}, "")

	if result.Err == nil {
		t.Fatal("expected error for GraphQL error on media list")
	}
	if result.Err.Code != core.ErrUpstream {
		t.Errorf("error code = %q, want %q", result.Err.Code, core.ErrUpstream)
	}
}

// ── Viewer query receives auth header ───────────────────────────────────

func TestAPISyncSendsAuthHeader(t *testing.T) {
	var receivedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, viewerJSON(1))
	}))
	defer srv.Close()

	p := NewAPIWithEndpoint(srv.URL)
	// This will fail on the media list call (viewer returns OK but next call
	// gets an unexpected response), but we only care about auth header.
	p.Sync(context.Background(), core.Credentials{AccessToken: "my-secret-token"}, "")

	if receivedAuth != "Bearer my-secret-token" {
		t.Errorf("Authorization = %q, want %q", receivedAuth, "Bearer my-secret-token")
	}
}

// ── mapMediaListEntry unit tests ────────────────────────────────────────

func TestMapMediaListEntryAnime(t *testing.T) {
	year := 2024
	entry := mediaListEntry{
		ID:          1,
		MediaID:     20,
		Status:      "COMPLETED",
		Score:       9.0,
		Progress:    24,
		UpdatedAt:   1710000000,
		CompletedAt: fuzzyDate{Year: &year, Month: intPtr(3), Day: intPtr(10)},
		StartedAt:   fuzzyDate{Year: &year, Month: intPtr(1), Day: intPtr(15)},
		Media: mediaEntry{
			ID:           20,
			Format:       "TV",
			Episodes:     220,
			Genres:       []string{"Action"},
			AverageScore: 79,
			SiteURL:      "https://anilist.co/anime/20",
		},
	}
	entry.Media.Title.English = "Naruto"
	entry.Media.Title.Romaji = "Naruto"
	entry.Media.Title.Native = "ナルト"

	item := mapMediaListEntry(entry, core.MediaVideo)

	if item.Platform != "anilist" {
		t.Errorf("Platform = %q", item.Platform)
	}
	if item.Type != core.MediaVideo {
		t.Errorf("Type = %q", item.Type)
	}
	if item.Title != "Naruto" {
		t.Errorf("Title = %q", item.Title)
	}
	if item.ExternalID != "20" {
		t.Errorf("ExternalID = %q", item.ExternalID)
	}
	wantTime := time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC)
	if !item.ConsumedAt.Equal(wantTime) {
		t.Errorf("ConsumedAt = %v, want %v", item.ConsumedAt, wantTime)
	}
}

func TestMapMediaListEntryNoCompletedAt(t *testing.T) {
	entry := mediaListEntry{
		ID:        1,
		MediaID:   21,
		Status:    "WATCHING",
		Score:     8.0,
		Progress:  12,
		UpdatedAt: 1710100000,
		Media: mediaEntry{
			ID:      21,
			Format:  "TV",
			SiteURL: "https://anilist.co/anime/21",
		},
	}
	entry.Media.Title.Romaji = "Shingeki no Kyojin"

	item := mapMediaListEntry(entry, core.MediaVideo)

	// Should fall back to updatedAt.
	wantTime := time.Unix(1710100000, 0).UTC()
	if !item.ConsumedAt.Equal(wantTime) {
		t.Errorf("ConsumedAt = %v, want %v", item.ConsumedAt, wantTime)
	}
	if item.Title != "Shingeki no Kyojin" {
		t.Errorf("Title = %q, want romaji fallback", item.Title)
	}
}

func TestMapMediaListEntryManga(t *testing.T) {
	entry := mediaListEntry{
		ID:              1,
		MediaID:         30,
		Status:          "READING",
		Score:           10.0,
		Progress:        300,
		ProgressVolumes: 30,
		UpdatedAt:       1710200000,
		Media: mediaEntry{
			ID:       30,
			Format:   "MANGA",
			Chapters: 1100,
			Volumes:  106,
			SiteURL:  "https://anilist.co/manga/30",
		},
	}
	entry.Media.Title.English = "One Piece"

	item := mapMediaListEntry(entry, core.MediaBook)

	if item.Type != core.MediaBook {
		t.Errorf("Type = %q, want %q", item.Type, core.MediaBook)
	}
	if item.RawMetadata["chapters"] != 1100 {
		t.Errorf("RawMetadata[chapters] = %v", item.RawMetadata["chapters"])
	}
	if item.RawMetadata["volumes"] != 106 {
		t.Errorf("RawMetadata[volumes] = %v", item.RawMetadata["volumes"])
	}
	if item.RawMetadata["progress_volumes"] != 30 {
		t.Errorf("RawMetadata[progress_volumes] = %v", item.RawMetadata["progress_volumes"])
	}
}

// ── gqlErrorToPluginError tests ─────────────────────────────────────────

func TestGqlErrorToPluginError(t *testing.T) {
	tests := []struct {
		name      string
		err       graphqlError
		wantCode  core.ErrorCode
		wantRetry bool
	}{
		{
			name:     "401 auth expired",
			err:      graphqlError{Message: "invalid token", Status: 401},
			wantCode: core.ErrAuthExpired,
		},
		{
			name:      "429 rate limit",
			err:       graphqlError{Message: "rate limited", Status: 429},
			wantCode:  core.ErrRateLimit,
			wantRetry: true,
		},
		{
			name:      "500 upstream",
			err:       graphqlError{Message: "server error", Status: 500},
			wantCode:  core.ErrUpstream,
			wantRetry: true,
		},
		{
			name:     "400 other",
			err:      graphqlError{Message: "bad request", Status: 400},
			wantCode: core.ErrUpstream,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pe := gqlErrorToPluginError(tt.err)
			if pe.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", pe.Code, tt.wantCode)
			}
			if pe.Retry != tt.wantRetry {
				t.Errorf("Retry = %v, want %v", pe.Retry, tt.wantRetry)
			}
		})
	}
}

// ── Compile-time interface check test ───────────────────────────────────

func TestCompileTimeInterfaceCheck(t *testing.T) {
	// This test is a no-op; the compile-time check is at package level.
	// It exists to document the intention.
	var _ core.SourcePlugin = (*APIPlugin)(nil)
}

// ── Title preference with only romaji in full sync ──────────────────────

func TestAPISyncTitlePreferenceRomajiOnly(t *testing.T) {
	// Build a custom anime response where english is empty.
	animeJSON := `{
		"data": {
			"MediaListCollection": {
				"lists": [{
					"entries": [{
						"id": 3001,
						"mediaId": 50,
						"status": "WATCHING",
						"score": 7.0,
						"progress": 5,
						"progressVolumes": 0,
						"updatedAt": 1710300000,
						"completedAt": {"year": null, "month": null, "day": null},
						"startedAt": {"year": null, "month": null, "day": null},
						"media": {
							"id": 50,
							"title": {"romaji": "Kimetsu no Yaiba", "english": "", "native": "鬼滅の刃"},
							"format": "TV",
							"episodes": 26,
							"chapters": 0,
							"volumes": 0,
							"genres": ["Action"],
							"averageScore": 83,
							"siteUrl": "https://anilist.co/anime/50"
						}
					}]
				}]
			}
		}
	}`

	srv := newSyncServer(t, animeJSON, emptyListJSON())
	defer srv.Close()

	p := NewAPIWithEndpoint(srv.URL)
	result := p.Sync(context.Background(), core.Credentials{AccessToken: "test-token"}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	// Should fall back to romaji.
	if result.Items[0].Title != "Kimetsu no Yaiba" {
		t.Errorf("Title = %q, want %q (romaji fallback)", result.Items[0].Title, "Kimetsu no Yaiba")
	}
}

// ── Cursor all filtered out ─────────────────────────────────────────────

func TestAPISyncCursorFiltersAllEntries(t *testing.T) {
	// Set cursor after all entries — everything should be filtered out.
	srv := newSyncServer(t, sampleAnimeListJSON(), sampleMangaListJSON())
	defer srv.Close()

	p := NewAPIWithEndpoint(srv.URL)
	// All entries have updatedAt <= 1710200000.
	result := p.Sync(context.Background(), core.Credentials{AccessToken: "test-token"}, strconv.FormatInt(1710200000, 10))

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected 0 items (all filtered by cursor), got %d", len(result.Items))
	}
	if result.NextCursor != "" {
		t.Errorf("NextCursor = %q, want empty (no items passed filter)", result.NextCursor)
	}
}

// ── Helpers ─────────────────────────────────────────────────────────────

func intPtr(v int) *int {
	return &v
}
