package anilist

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/justestif/specto/internal/core"
)

// newTestGraphQLServer creates an httptest.Server that responds to GraphQL
// queries with the given mediaResult. If result is nil, it returns a
// response with null data (no match).
func newTestGraphQLServer(t *testing.T, result *mediaResult) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}

		// Parse the request to verify it's valid GraphQL.
		var req graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decoding request body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		var resp graphqlResponse
		if result != nil {
			resp.Data = &mediaData{Media: result}
		} else {
			resp.Data = &mediaData{Media: nil}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestName(t *testing.T) {
	p := New()
	if got := p.Name(); got != "anilist" {
		t.Errorf("Name() = %q, want %q", got, "anilist")
	}
}

func TestSupportsVideoOnly(t *testing.T) {
	p := New()

	tests := []struct {
		mediaType string
		platform  string
		want      bool
	}{
		{"video", "crunchyroll", true},
		{"video", "funimation", true},
		{"video", "hidive", true},
		{"video", "mangadex", true},
		{"video", "youtube", false},
		{"video", "netflix", false},
		{"video", "", false},
		{"music", "spotify", false},
		{"article", "medium", false},
		{"podcast", "apple", false},
	}

	for _, tt := range tests {
		t.Run(tt.mediaType+"_"+tt.platform, func(t *testing.T) {
			got := p.Supports(tt.mediaType, tt.platform)
			if got != tt.want {
				t.Errorf("Supports(%q, %q) = %v, want %v", tt.mediaType, tt.platform, got, tt.want)
			}
		})
	}
}

func TestEnrichSuccessfulAnimeLookup(t *testing.T) {
	result := &mediaResult{
		ID:           21, // Naruto
		Genres:       []string{"Action", "Adventure", "Fantasy"},
		Format:       "TV",
		Episodes:     220,
		Duration:     23,
		AverageScore: 79,
		Tags: []anilistTag{
			{Name: "Shounen", Rank: 93, IsMediaSpoiler: false},
			{Name: "Isekai", Rank: 70, IsMediaSpoiler: false},
			{Name: "Time Travel", Rank: 65, IsMediaSpoiler: false},
		},
	}

	srv := newTestGraphQLServer(t, result)
	defer srv.Close()

	p := newForTest(srv.URL)
	items := []core.MediaItem{
		{
			Platform:   "crunchyroll",
			Type:       core.MediaVideo,
			Title:      "Naruto",
			Creator:    "Masashi Kishimoto",
			ExternalID: "naruto-1",
		},
	}

	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d items, want 1", len(got))
	}

	item := got[0]

	// Should have genre tags mapped from AniList.
	wantTags := []string{"action", "adventure", "fantasy"}
	for _, tag := range wantTags {
		if !containsTag(item.Tags, tag) {
			t.Errorf("Tags = %v, want to contain %q", item.Tags, tag)
		}
	}

	// Format: TV → "series"
	if !containsTag(item.Tags, "series") {
		t.Errorf("Tags = %v, want to contain %q", item.Tags, "series")
	}

	// Isekai (rank 70 >= 60) → "fantasy" (already present, no dup)
	// Time Travel (rank 65 >= 60) → "sci-fi"
	if !containsTag(item.Tags, "sci-fi") {
		t.Errorf("Tags = %v, want to contain %q from Time Travel tag", item.Tags, "sci-fi")
	}

	// RawMetadata checks.
	if item.RawMetadata["anilist_id"] != 21 {
		t.Errorf("RawMetadata[anilist_id] = %v, want 21", item.RawMetadata["anilist_id"])
	}
	if item.RawMetadata["anilist_score"] != 79 {
		t.Errorf("RawMetadata[anilist_score] = %v, want 79", item.RawMetadata["anilist_score"])
	}
	if item.RawMetadata["anilist_episodes"] != 220 {
		t.Errorf("RawMetadata[anilist_episodes] = %v, want 220", item.RawMetadata["anilist_episodes"])
	}
	if item.RawMetadata["anilist_episode_duration"] != 23 {
		t.Errorf("RawMetadata[anilist_episode_duration] = %v, want 23", item.RawMetadata["anilist_episode_duration"])
	}
}

func TestEnrichNoMatch(t *testing.T) {
	srv := newTestGraphQLServer(t, nil)
	defer srv.Close()

	p := newForTest(srv.URL)
	items := []core.MediaItem{
		{
			Platform:   "youtube",
			Type:       core.MediaVideo,
			Title:      "Random Video That Is Not Anime",
			ExternalID: "xyz-123",
		},
	}

	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d items, want 1", len(got))
	}

	// No match means no tags added and no metadata set.
	if len(got[0].Tags) != 0 {
		t.Errorf("Tags = %v, want empty (no match)", got[0].Tags)
	}
	if got[0].RawMetadata != nil {
		t.Errorf("RawMetadata = %v, want nil (no match)", got[0].RawMetadata)
	}
}

func TestEnrichGenreMapping(t *testing.T) {
	tests := []struct {
		name    string
		genres  []string
		want    []string
		notWant []string
	}{
		{
			name:   "direct mappings",
			genres: []string{"Action", "Comedy", "Drama"},
			want:   []string{"action", "comedy", "drama"},
		},
		{
			name:   "supernatural maps to fantasy",
			genres: []string{"Supernatural"},
			want:   []string{"fantasy"},
		},
		{
			name:   "slice of life maps to drama",
			genres: []string{"Slice of Life"},
			want:   []string{"drama"},
		},
		{
			name:   "sports maps to topic",
			genres: []string{"Sports"},
			want:   []string{"sports"},
		},
		{
			name:   "music maps to musical",
			genres: []string{"Music"},
			want:   []string{"musical"},
		},
		{
			name:    "unknown genre is skipped",
			genres:  []string{"Ecchi", "Hentai"},
			notWant: []string{"ecchi", "hentai"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &mediaResult{
				ID:     1,
				Genres: tt.genres,
				Format: "TV",
			}

			srv := newTestGraphQLServer(t, result)
			defer srv.Close()

			p := newForTest(srv.URL)
			items := []core.MediaItem{
				{Platform: "crunchyroll", Type: core.MediaVideo, Title: "Test Anime", ExternalID: "test-1"},
			}

			got, err := p.Enrich(context.Background(), items)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, tag := range tt.want {
				if !containsTag(got[0].Tags, tag) {
					t.Errorf("Tags = %v, want to contain %q", got[0].Tags, tag)
				}
			}
			for _, tag := range tt.notWant {
				if containsTag(got[0].Tags, tag) {
					t.Errorf("Tags = %v, should not contain %q", got[0].Tags, tag)
				}
			}
		})
	}
}

func TestEnrichTagFiltering(t *testing.T) {
	result := &mediaResult{
		ID:     1,
		Genres: []string{"Action"},
		Format: "TV",
		Tags: []anilistTag{
			// Should be included (rank >= 60, not spoiler).
			{Name: "Psychological", Rank: 80, IsMediaSpoiler: false},
			// Should be excluded (rank < 60).
			{Name: "Gore", Rank: 50, IsMediaSpoiler: false},
			// Should be excluded (spoiler).
			{Name: "Time Travel", Rank: 90, IsMediaSpoiler: true},
			// Should be included (rank >= 60, not spoiler).
			{Name: "Mecha", Rank: 75, IsMediaSpoiler: false},
		},
	}

	srv := newTestGraphQLServer(t, result)
	defer srv.Close()

	p := newForTest(srv.URL)
	items := []core.MediaItem{
		{Platform: "crunchyroll", Type: core.MediaVideo, Title: "Test", ExternalID: "test-1"},
	}

	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Psychological (rank 80) → "intense"
	if !containsTag(got[0].Tags, "intense") {
		t.Errorf("Tags = %v, want to contain %q (Psychological, rank 80)", got[0].Tags, "intense")
	}

	// Mecha (rank 75) → "technology"
	if !containsTag(got[0].Tags, "technology") {
		t.Errorf("Tags = %v, want to contain %q (Mecha, rank 75)", got[0].Tags, "technology")
	}

	// Gore (rank 50 < 60) should NOT be mapped.
	if containsTag(got[0].Tags, "dark") {
		t.Errorf("Tags = %v, should not contain %q (Gore rank 50 < threshold)", got[0].Tags, "dark")
	}

	// Time Travel (spoiler) should NOT be mapped.
	if containsTag(got[0].Tags, "sci-fi") {
		t.Errorf("Tags = %v, should not contain %q (Time Travel was spoiler)", got[0].Tags, "sci-fi")
	}
}

func TestEnrichFormatDetection(t *testing.T) {
	tests := []struct {
		name      string
		format    string
		mediaType string
		wantTag   string
	}{
		{"anime TV", "TV", "ANIME", "series"},
		{"anime TV_SHORT", "TV_SHORT", "ANIME", "series"},
		{"anime movie", "MOVIE", "ANIME", "film"},
		{"anime OVA", "OVA", "ANIME", "episode"},
		{"anime ONA", "ONA", "ANIME", "episode"},
		{"anime SPECIAL", "SPECIAL", "ANIME", "episode"},
		{"manga any format", "MANGA", "MANGA", "graphic-novel"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTag(tt.format, tt.mediaType)
			if len(got) == 0 {
				t.Fatalf("formatTag(%q, %q) returned empty slice", tt.format, tt.mediaType)
			}
			if got[0] != tt.wantTag {
				t.Errorf("formatTag(%q, %q) = %v, want [%q]", tt.format, tt.mediaType, got, tt.wantTag)
			}
		})
	}
}

func TestEnrichFormatUnknown(t *testing.T) {
	got := formatTag("MUSIC", "ANIME")
	if len(got) != 0 {
		t.Errorf("formatTag(%q, %q) = %v, want empty", "MUSIC", "ANIME", got)
	}
}

func TestDetectMediaType(t *testing.T) {
	tests := []struct {
		name string
		item core.MediaItem
		want string
	}{
		{
			name: "crunchyroll platform",
			item: core.MediaItem{Platform: "crunchyroll"},
			want: "ANIME",
		},
		{
			name: "mangadex platform",
			item: core.MediaItem{Platform: "mangadex"},
			want: "MANGA",
		},
		{
			name: "raw metadata media_type manga",
			item: core.MediaItem{
				Platform:    "unknown",
				RawMetadata: map[string]any{"media_type": "manga"},
			},
			want: "MANGA",
		},
		{
			name: "raw metadata media_type anime",
			item: core.MediaItem{
				Platform:    "unknown",
				RawMetadata: map[string]any{"media_type": "anime"},
			},
			want: "ANIME",
		},
		{
			name: "raw metadata content_type manga",
			item: core.MediaItem{
				Platform:    "unknown",
				RawMetadata: map[string]any{"content_type": "manga"},
			},
			want: "MANGA",
		},
		{
			name: "unknown platform defaults to anime",
			item: core.MediaItem{Platform: "netflix"},
			want: "ANIME",
		},
		{
			name: "metadata takes priority over platform",
			item: core.MediaItem{
				Platform:    "crunchyroll",
				RawMetadata: map[string]any{"media_type": "manga"},
			},
			want: "MANGA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectMediaType(&tt.item)
			if got != tt.want {
				t.Errorf("detectMediaType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEnrichHTTP429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	p := newForTest(srv.URL)
	items := []core.MediaItem{
		{Platform: "crunchyroll", Type: core.MediaVideo, Title: "Naruto", ExternalID: "n-1"},
	}

	// Per-item failures don't abort the batch — the item is skipped.
	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("Enrich should not return batch-level error, got: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d items, want 1", len(got))
	}
	// Item should be unchanged (no tags added).
	if len(got[0].Tags) != 0 {
		t.Errorf("Tags = %v, want empty (rate limited item should be skipped)", got[0].Tags)
	}
}

func TestEnrichHTTP500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	p := newForTest(srv.URL)
	items := []core.MediaItem{
		{Platform: "crunchyroll", Type: core.MediaVideo, Title: "Naruto", ExternalID: "n-1"},
	}

	// Per-item failures don't abort the batch.
	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("Enrich should not return batch-level error, got: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d items, want 1", len(got))
	}
	if len(got[0].Tags) != 0 {
		t.Errorf("Tags = %v, want empty (server error item should be skipped)", got[0].Tags)
	}
}

func TestEnrichInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{invalid json"))
	}))
	defer srv.Close()

	p := newForTest(srv.URL)
	items := []core.MediaItem{
		{Platform: "crunchyroll", Type: core.MediaVideo, Title: "Test", ExternalID: "t-1"},
	}

	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("Enrich should not return batch-level error, got: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d items, want 1", len(got))
	}
	if len(got[0].Tags) != 0 {
		t.Errorf("Tags = %v, want empty (invalid JSON item should be skipped)", got[0].Tags)
	}
}

func TestEnrichGraphQLError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := graphqlResponse{
			Errors: []graphqlError{
				{Message: "something went wrong", Status: 500},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := newForTest(srv.URL)
	items := []core.MediaItem{
		{Platform: "crunchyroll", Type: core.MediaVideo, Title: "Test", ExternalID: "t-1"},
	}

	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("Enrich should not return batch-level error, got: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d items, want 1", len(got))
	}
	if len(got[0].Tags) != 0 {
		t.Errorf("Tags = %v, want empty (GraphQL error item should be skipped)", got[0].Tags)
	}
}

func TestEnrichGraphQLNotFoundError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := graphqlResponse{
			Errors: []graphqlError{
				{Message: "not found", Status: 404},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := newForTest(srv.URL)
	items := []core.MediaItem{
		{Platform: "crunchyroll", Type: core.MediaVideo, Title: "Test", ExternalID: "t-1"},
	}

	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 404 GraphQL error should be treated as no match (not an error).
	if len(got[0].Tags) != 0 {
		t.Errorf("Tags = %v, want empty (404 should mean no match)", got[0].Tags)
	}
}

func TestEnrichEmptyItems(t *testing.T) {
	p := newForTest("http://unused")
	got, err := p.Enrich(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestEnrichEmptyTitle(t *testing.T) {
	// An item with empty title should be skipped (no API call).
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		resp := graphqlResponse{Data: &mediaData{Media: nil}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := newForTest(srv.URL)
	items := []core.MediaItem{
		{Platform: "crunchyroll", Type: core.MediaVideo, Title: "", ExternalID: "t-1"},
	}

	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d items, want 1", len(got))
	}
	if requestCount != 0 {
		t.Errorf("API request count = %d, want 0 (empty title should skip)", requestCount)
	}
}

func TestEnrichDoesNotMutateOriginal(t *testing.T) {
	result := &mediaResult{
		ID:     1,
		Genres: []string{"Action"},
		Format: "TV",
	}

	srv := newTestGraphQLServer(t, result)
	defer srv.Close()

	p := newForTest(srv.URL)
	original := []core.MediaItem{
		{Platform: "crunchyroll", Type: core.MediaVideo, Title: "Test", ExternalID: "t-1"},
	}

	got, err := p.Enrich(context.Background(), original)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Original should have no tags.
	if len(original[0].Tags) != 0 {
		t.Errorf("original[0].Tags = %v, want empty (should not be mutated)", original[0].Tags)
	}
	// Enriched item should have tags.
	if len(got[0].Tags) == 0 {
		t.Error("got[0].Tags is empty, want tags from enrichment")
	}
}

func TestEnrichPreservesExistingTags(t *testing.T) {
	result := &mediaResult{
		ID:     1,
		Genres: []string{"Action", "Comedy"},
		Format: "TV",
	}

	srv := newTestGraphQLServer(t, result)
	defer srv.Close()

	p := newForTest(srv.URL)
	items := []core.MediaItem{
		{
			Platform:   "crunchyroll",
			Type:       core.MediaVideo,
			Title:      "Test",
			ExternalID: "t-1",
			Tags:       []string{"action", "funny"}, // action already exists
		},
	}

	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain action (no dup), funny (existing), comedy (new).
	if !containsTag(got[0].Tags, "action") {
		t.Errorf("Tags = %v, want to contain %q", got[0].Tags, "action")
	}
	if !containsTag(got[0].Tags, "funny") {
		t.Errorf("Tags = %v, want to contain %q", got[0].Tags, "funny")
	}
	if !containsTag(got[0].Tags, "comedy") {
		t.Errorf("Tags = %v, want to contain %q", got[0].Tags, "comedy")
	}

	// Verify no duplicate action.
	count := 0
	for _, tag := range got[0].Tags {
		if tag == "action" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("action appears %d times in Tags, want exactly 1", count)
	}
}

func TestEnrichBatchPartialFailure(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		var req graphqlRequest
		json.NewDecoder(r.Body).Decode(&req)

		search := req.Variables["search"].(string)

		w.Header().Set("Content-Type", "application/json")

		if search == "Fail Item" {
			// Return a server error for this item.
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("server error"))
			return
		}

		// Return a valid result for other items.
		result := &mediaResult{
			ID:     1,
			Genres: []string{"Action"},
			Format: "TV",
		}
		resp := graphqlResponse{Data: &mediaData{Media: result}}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := newForTest(srv.URL)
	items := []core.MediaItem{
		{Platform: "crunchyroll", Type: core.MediaVideo, Title: "Success Item", ExternalID: "s-1"},
		{Platform: "crunchyroll", Type: core.MediaVideo, Title: "Fail Item", ExternalID: "f-1"},
		{Platform: "crunchyroll", Type: core.MediaVideo, Title: "Another Success", ExternalID: "s-2"},
	}

	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("Enrich should not return batch-level error, got: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d items, want 3", len(got))
	}

	// First item should be enriched.
	if !containsTag(got[0].Tags, "action") {
		t.Errorf("items[0].Tags = %v, want to contain %q", got[0].Tags, "action")
	}

	// Second item should be unenriched (server error).
	if len(got[1].Tags) != 0 {
		t.Errorf("items[1].Tags = %v, want empty (failed item)", got[1].Tags)
	}

	// Third item should be enriched.
	if !containsTag(got[2].Tags, "action") {
		t.Errorf("items[2].Tags = %v, want to contain %q", got[2].Tags, "action")
	}

	// All 3 items should have had API calls (failure doesn't skip others).
	if callCount != 3 {
		t.Errorf("API call count = %d, want 3", callCount)
	}
}

func TestEnrichMovieFormat(t *testing.T) {
	result := &mediaResult{
		ID:     1,
		Genres: []string{"Fantasy"},
		Format: "MOVIE",
	}

	srv := newTestGraphQLServer(t, result)
	defer srv.Close()

	p := newForTest(srv.URL)
	items := []core.MediaItem{
		{Platform: "crunchyroll", Type: core.MediaVideo, Title: "Anime Movie", ExternalID: "m-1"},
	}

	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !containsTag(got[0].Tags, "film") {
		t.Errorf("Tags = %v, want to contain %q for MOVIE format", got[0].Tags, "film")
	}
}

func TestEnrichMangaPlatformDetection(t *testing.T) {
	// Verify that mangadex triggers MANGA search by checking the request.
	var receivedType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphqlRequest
		json.NewDecoder(r.Body).Decode(&req)
		if t, ok := req.Variables["type"].(string); ok {
			receivedType = t
		}

		result := &mediaResult{
			ID:     1,
			Genres: []string{"Action"},
			Format: "MANGA",
		}
		resp := graphqlResponse{Data: &mediaData{Media: result}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := newForTest(srv.URL)
	items := []core.MediaItem{
		{Platform: "mangadex", Type: core.MediaVideo, Title: "One Piece", ExternalID: "op-1"},
	}

	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedType != "MANGA" {
		t.Errorf("GraphQL type variable = %q, want %q for mangadex platform", receivedType, "MANGA")
	}

	// Manga should get "graphic-novel" format tag.
	if !containsTag(got[0].Tags, "graphic-novel") {
		t.Errorf("Tags = %v, want to contain %q for manga", got[0].Tags, "graphic-novel")
	}
}

func TestSearchMediaRequestFormat(t *testing.T) {
	var receivedReq graphqlRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedReq)
		resp := graphqlResponse{Data: &mediaData{Media: nil}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := newForTest(srv.URL)
	p.searchMedia(context.Background(), "Naruto", "ANIME")

	if receivedReq.Variables["search"] != "Naruto" {
		t.Errorf("search variable = %q, want %q", receivedReq.Variables["search"], "Naruto")
	}
	if receivedReq.Variables["type"] != "ANIME" {
		t.Errorf("type variable = %q, want %q", receivedReq.Variables["type"], "ANIME")
	}
}

func TestEnrichContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a slow response.
		select {
		case <-r.Context().Done():
			return
		case <-make(chan struct{}):
			// never completes
		}
	}))
	defer srv.Close()

	p := newForTest(srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	items := []core.MediaItem{
		{Platform: "crunchyroll", Type: core.MediaVideo, Title: "Test", ExternalID: "t-1"},
	}

	// Should not hang; the cancelled context will cause the HTTP request to fail.
	got, err := p.Enrich(ctx, items)
	if err != nil {
		t.Fatalf("Enrich should not return batch-level error, got: %v", err)
	}
	// The item should be returned unenriched.
	if len(got) != 1 {
		t.Fatalf("got %d items, want 1", len(got))
	}
}

func TestEnrichOnlyValidTagsIncluded(t *testing.T) {
	// The "animation" tag from the task spec is actually a valid tag.
	// Verify that only valid tags from the fixed set are assigned.
	result := &mediaResult{
		ID:     1,
		Genres: []string{"Action", "Ecchi"}, // Ecchi is NOT in fixed tag set
		Format: "TV",
		Tags: []anilistTag{
			{Name: "SomeFakeTag", Rank: 90, IsMediaSpoiler: false},
		},
	}

	srv := newTestGraphQLServer(t, result)
	defer srv.Close()

	p := newForTest(srv.URL)
	items := []core.MediaItem{
		{Platform: "crunchyroll", Type: core.MediaVideo, Title: "Test", ExternalID: "t-1"},
	}

	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Every tag should be in the valid tag set.
	for _, tag := range got[0].Tags {
		if !core.IsValidTag(tag) {
			t.Errorf("Tag %q is not in the fixed tag set", tag)
		}
	}
}

// ── Helpers ─────────────────────────────────────────────────────────────

func containsTag(tags []string, target string) bool {
	for _, t := range tags {
		if t == target {
			return true
		}
	}
	return false
}
