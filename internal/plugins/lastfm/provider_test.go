package lastfm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/justestif/specto/internal/core"
)

// ── Test helpers ────────────────────────────────────────────────────────

// newLastfmServer creates an httptest.Server that routes Last.fm API requests
// to the appropriate handler based on the "method" query parameter.
func newLastfmServer(t *testing.T, trackTags, artistTags map[string]lastfmTagsResponse) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := r.URL.Query().Get("method")
		artist := r.URL.Query().Get("artist")
		track := r.URL.Query().Get("track")

		w.Header().Set("Content-Type", "application/json")

		switch method {
		case "track.getTopTags":
			key := artist + "|" + track
			resp, ok := trackTags[key]
			if !ok {
				// Return empty tags for unknown tracks.
				resp = lastfmTagsResponse{}
			}
			json.NewEncoder(w).Encode(resp)

		case "artist.getTopTags":
			resp, ok := artistTags[artist]
			if !ok {
				resp = lastfmTagsResponse{}
			}
			json.NewEncoder(w).Encode(resp)

		default:
			http.Error(w, "unknown method", http.StatusBadRequest)
		}
	}))
}

// newMBServer creates an httptest.Server that handles MusicBrainz recording
// search and lookup requests.
func newMBServer(t *testing.T, recordings map[string]mbRecordingResponse) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Verify User-Agent header.
		if ua := r.Header.Get("User-Agent"); ua != mbUserAgent {
			t.Errorf("MusicBrainz User-Agent = %q, want %q", ua, mbUserAgent)
		}

		path := r.URL.Path

		// Extract MBID from path if present (e.g., /recording/mb-id-1).
		mbid := strings.TrimPrefix(path, "/recording/")
		mbid = strings.TrimRight(mbid, "/")

		if mbid != "" {
			// Lookup: /recording/{mbid}?inc=genres&fmt=json
			resp, ok := recordings[mbid]
			if !ok {
				http.NotFound(w, r)
				return
			}
			json.NewEncoder(w).Encode(resp)
		} else if path == "/recording/" || path == "/recording" {
			// Search: /recording/?query=...&fmt=json&limit=1
			if len(recordings) > 0 {
				var results []mbRecording
				for id, rec := range recordings {
					results = append(results, mbRecording{
						ID:    id,
						Title: rec.Title,
					})
					break
				}
				json.NewEncoder(w).Encode(mbSearchResponse{Recordings: results})
			} else {
				json.NewEncoder(w).Encode(mbSearchResponse{})
			}
		} else {
			http.NotFound(w, r)
		}
	}))
}

func makeLastfmTags(tags ...string) lastfmTagsResponse {
	var resp lastfmTagsResponse
	for _, name := range tags {
		resp.TopTags.Tag = append(resp.TopTags.Tag, lastfmTag{
			Name:  name,
			Count: 100, // above minTagCount
		})
	}
	return resp
}

func containsTag(tags []string, target string) bool {
	for _, t := range tags {
		if t == target {
			return true
		}
	}
	return false
}

// ── Tests ───────────────────────────────────────────────────────────────

func TestProviderName(t *testing.T) {
	p := New("test-key")
	if got := p.Name(); got != "lastfm" {
		t.Errorf("Name() = %q, want %q", got, "lastfm")
	}
}

func TestProviderSupports(t *testing.T) {
	p := New("test-key")

	tests := []struct {
		mediaType string
		platform  string
		want      bool
	}{
		{"music", "spotify", true},
		{"music", "youtube", true},
		{"music", "", true},
		{"video", "youtube", false},
		{"article", "pocket", false},
		{"podcast", "spotify", false},
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

func TestEnrichBasic(t *testing.T) {
	trackTags := map[string]lastfmTagsResponse{
		"Radiohead|Creep": makeLastfmTags("rock", "alternative", "90s"),
	}
	artistTags := map[string]lastfmTagsResponse{
		"Radiohead": makeLastfmTags("alternative", "rock", "indie"),
	}
	recordings := map[string]mbRecordingResponse{
		"mb-creep-id": {
			ID:    "mb-creep-id",
			Title: "Creep",
			Genres: []mbGenre{
				{Name: "rock", Count: 5},
				{Name: "alternative rock", Count: 3},
			},
		},
	}

	lastfmSrv := newLastfmServer(t, trackTags, artistTags)
	defer lastfmSrv.Close()

	mbSrv := newMBServer(t, recordings)
	defer mbSrv.Close()

	p := newForTest("test-key", lastfmSrv.URL, mbSrv.URL)

	items := []core.MediaItem{
		{
			Platform:   "spotify",
			Type:       core.MediaMusic,
			Title:      "Creep",
			Creator:    "Radiohead",
			ExternalID: "spotify:track:123",
		},
	}

	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d items, want 1", len(got))
	}

	// Should have "rock" and "alternative" from Last.fm, "indie" from artist tags.
	if !containsTag(got[0].Tags, "rock") {
		t.Errorf("Tags = %v, want to contain %q", got[0].Tags, "rock")
	}
	if !containsTag(got[0].Tags, "alternative") {
		t.Errorf("Tags = %v, want to contain %q", got[0].Tags, "alternative")
	}
	if !containsTag(got[0].Tags, "indie") {
		t.Errorf("Tags = %v, want to contain %q", got[0].Tags, "indie")
	}
}

func TestEnrichEmptyItems(t *testing.T) {
	p := newForTest("test-key", "http://unused", "http://unused")

	got, err := p.Enrich(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestEnrichArtistDedup(t *testing.T) {
	// Track two songs by the same artist — artist tags should be fetched once.
	var artistCallCount atomic.Int32

	lastfmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := r.URL.Query().Get("method")
		w.Header().Set("Content-Type", "application/json")

		switch method {
		case "track.getTopTags":
			json.NewEncoder(w).Encode(makeLastfmTags("rock"))
		case "artist.getTopTags":
			artistCallCount.Add(1)
			json.NewEncoder(w).Encode(makeLastfmTags("rock", "alternative"))
		}
	}))
	defer lastfmSrv.Close()

	// MB server returns no results (simpler test).
	mbSrv := newMBServer(t, nil)
	defer mbSrv.Close()

	p := newForTest("test-key", lastfmSrv.URL, mbSrv.URL)

	items := []core.MediaItem{
		{Platform: "spotify", Type: core.MediaMusic, Title: "Song A", Creator: "Radiohead", ExternalID: "a"},
		{Platform: "spotify", Type: core.MediaMusic, Title: "Song B", Creator: "Radiohead", ExternalID: "b"},
		{Platform: "spotify", Type: core.MediaMusic, Title: "Song C", Creator: "radiohead", ExternalID: "c"}, // same artist, different case
	}

	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d items, want 3", len(got))
	}

	// Artist endpoint should be called exactly once (all three items share the same artist).
	if count := artistCallCount.Load(); count != 1 {
		t.Errorf("artist.getTopTags called %d times, want 1", count)
	}

	// All items should have artist tags.
	for i, item := range got {
		if !containsTag(item.Tags, "rock") {
			t.Errorf("items[%d].Tags = %v, want to contain %q", i, item.Tags, "rock")
		}
		if !containsTag(item.Tags, "alternative") {
			t.Errorf("items[%d].Tags = %v, want to contain %q", i, item.Tags, "alternative")
		}
	}
}

func TestEnrichPreservesExistingTags(t *testing.T) {
	lastfmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(makeLastfmTags("rock", "indie"))
	}))
	defer lastfmSrv.Close()

	mbSrv := newMBServer(t, nil)
	defer mbSrv.Close()

	p := newForTest("test-key", lastfmSrv.URL, mbSrv.URL)

	items := []core.MediaItem{
		{
			Platform:   "spotify",
			Type:       core.MediaMusic,
			Title:      "Test Song",
			Creator:    "Test Artist",
			ExternalID: "x",
			Tags:       []string{"rock", "energetic"}, // rock already exists
		},
	}

	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain rock (no dup), energetic (existing), indie (new).
	if !containsTag(got[0].Tags, "rock") {
		t.Errorf("Tags = %v, want to contain %q", got[0].Tags, "rock")
	}
	if !containsTag(got[0].Tags, "energetic") {
		t.Errorf("Tags = %v, want to contain %q", got[0].Tags, "energetic")
	}
	if !containsTag(got[0].Tags, "indie") {
		t.Errorf("Tags = %v, want to contain %q", got[0].Tags, "indie")
	}

	// Verify no duplicate rock.
	count := 0
	for _, tag := range got[0].Tags {
		if tag == "rock" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("rock appears %d times in Tags, want exactly 1", count)
	}
}

func TestEnrichDoesNotMutateOriginal(t *testing.T) {
	lastfmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(makeLastfmTags("rock"))
	}))
	defer lastfmSrv.Close()

	mbSrv := newMBServer(t, nil)
	defer mbSrv.Close()

	p := newForTest("test-key", lastfmSrv.URL, mbSrv.URL)

	original := []core.MediaItem{
		{Platform: "spotify", Type: core.MediaMusic, Title: "Song", Creator: "Artist", ExternalID: "x"},
	}

	got, err := p.Enrich(context.Background(), original)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Original should have no tags.
	if len(original[0].Tags) != 0 {
		t.Errorf("original[0].Tags = %v, want empty (should not be mutated)", original[0].Tags)
	}
	// Enriched should have tags.
	if len(got[0].Tags) == 0 {
		t.Error("got[0].Tags is empty, want tags from enrichment")
	}
}

func TestEnrichLastfmHTTP429(t *testing.T) {
	lastfmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer lastfmSrv.Close()

	mbSrv := newMBServer(t, nil)
	defer mbSrv.Close()

	p := newForTest("test-key", lastfmSrv.URL, mbSrv.URL)

	items := []core.MediaItem{
		{Platform: "spotify", Type: core.MediaMusic, Title: "Song", Creator: "Artist", ExternalID: "x"},
	}

	// 429 is non-fatal for individual items — enrichment should succeed
	// but items won't have tags (errors are logged and skipped).
	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d items, want 1", len(got))
	}
	// No tags added due to rate limit errors.
	if len(got[0].Tags) != 0 {
		t.Errorf("Tags = %v, want empty (rate limited)", got[0].Tags)
	}
}

func TestEnrichLastfmHTTP500(t *testing.T) {
	lastfmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer lastfmSrv.Close()

	mbSrv := newMBServer(t, nil)
	defer mbSrv.Close()

	p := newForTest("test-key", lastfmSrv.URL, mbSrv.URL)

	items := []core.MediaItem{
		{Platform: "spotify", Type: core.MediaMusic, Title: "Song", Creator: "Artist", ExternalID: "x"},
	}

	// 500 is non-fatal per item — should complete without error.
	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d items, want 1", len(got))
	}
}

func TestEnrichLastfmInvalidJSON(t *testing.T) {
	lastfmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{invalid json"))
	}))
	defer lastfmSrv.Close()

	mbSrv := newMBServer(t, nil)
	defer mbSrv.Close()

	p := newForTest("test-key", lastfmSrv.URL, mbSrv.URL)

	items := []core.MediaItem{
		{Platform: "spotify", Type: core.MediaMusic, Title: "Song", Creator: "Artist", ExternalID: "x"},
	}

	// Invalid JSON is non-fatal per item.
	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d items, want 1", len(got))
	}
}

func TestEnrichMBHTTP429(t *testing.T) {
	lastfmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(makeLastfmTags("rock"))
	}))
	defer lastfmSrv.Close()

	mbSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer mbSrv.Close()

	p := newForTest("test-key", lastfmSrv.URL, mbSrv.URL)

	items := []core.MediaItem{
		{Platform: "spotify", Type: core.MediaMusic, Title: "Song", Creator: "Artist", ExternalID: "x"},
	}

	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Last.fm tags should still be present even though MB failed.
	if !containsTag(got[0].Tags, "rock") {
		t.Errorf("Tags = %v, want to contain %q (Last.fm should succeed)", got[0].Tags, "rock")
	}
}

func TestEnrichContextCancelled(t *testing.T) {
	lastfmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response.
		time.Sleep(2 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(makeLastfmTags("rock"))
	}))
	defer lastfmSrv.Close()

	mbSrv := newMBServer(t, nil)
	defer mbSrv.Close()

	p := newForTest("test-key", lastfmSrv.URL, mbSrv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	items := []core.MediaItem{
		{Platform: "spotify", Type: core.MediaMusic, Title: "Song", Creator: "Artist", ExternalID: "x"},
	}

	// Should still return items (errors are non-fatal).
	got, err := p.Enrich(ctx, items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d items, want 1", len(got))
	}
}

func TestEnrichNoCreator(t *testing.T) {
	lastfmSrv := newLastfmServer(t, nil, nil)
	defer lastfmSrv.Close()

	mbSrv := newMBServer(t, nil)
	defer mbSrv.Close()

	p := newForTest("test-key", lastfmSrv.URL, mbSrv.URL)

	items := []core.MediaItem{
		{Platform: "spotify", Type: core.MediaMusic, Title: "Unknown Track", ExternalID: "x"},
	}

	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d items, want 1", len(got))
	}
	// No creator → no tags fetched.
	if len(got[0].Tags) != 0 {
		t.Errorf("Tags = %v, want empty for item without creator", got[0].Tags)
	}
}

func TestEnrichMultipleArtists(t *testing.T) {
	trackTags := map[string]lastfmTagsResponse{
		"Radiohead|Creep":        makeLastfmTags("rock", "alternative"),
		"Kendrick Lamar|HUMBLE.": makeLastfmTags("hip-hop", "rap"),
		"Miles Davis|So What":    makeLastfmTags("jazz"),
	}
	artistTags := map[string]lastfmTagsResponse{
		"Radiohead":      makeLastfmTags("rock", "alternative", "indie"),
		"Kendrick Lamar": makeLastfmTags("hip-hop"),
		"Miles Davis":    makeLastfmTags("jazz", "blues"),
	}

	lastfmSrv := newLastfmServer(t, trackTags, artistTags)
	defer lastfmSrv.Close()

	mbSrv := newMBServer(t, nil)
	defer mbSrv.Close()

	p := newForTest("test-key", lastfmSrv.URL, mbSrv.URL)

	items := []core.MediaItem{
		{Platform: "spotify", Type: core.MediaMusic, Title: "Creep", Creator: "Radiohead", ExternalID: "a"},
		{Platform: "spotify", Type: core.MediaMusic, Title: "HUMBLE.", Creator: "Kendrick Lamar", ExternalID: "b"},
		{Platform: "spotify", Type: core.MediaMusic, Title: "So What", Creator: "Miles Davis", ExternalID: "c"},
	}

	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Radiohead track
	if !containsTag(got[0].Tags, "rock") {
		t.Errorf("Radiohead track Tags = %v, want rock", got[0].Tags)
	}
	if !containsTag(got[0].Tags, "indie") {
		t.Errorf("Radiohead track Tags = %v, want indie (from artist)", got[0].Tags)
	}

	// Kendrick track
	if !containsTag(got[1].Tags, "hip-hop") {
		t.Errorf("Kendrick track Tags = %v, want hip-hop", got[1].Tags)
	}

	// Miles Davis track
	if !containsTag(got[2].Tags, "jazz") {
		t.Errorf("Miles Davis track Tags = %v, want jazz", got[2].Tags)
	}
	if !containsTag(got[2].Tags, "blues") {
		t.Errorf("Miles Davis track Tags = %v, want blues (from artist)", got[2].Tags)
	}
}

func TestEnrichLowCountTagsFiltered(t *testing.T) {
	// Return tags with counts below minTagCount — they should be filtered out.
	lastfmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := lastfmTagsResponse{}
		resp.TopTags.Tag = []lastfmTag{
			{Name: "rock", Count: 100},          // above threshold
			{Name: "alternative", Count: 5},     // below threshold
			{Name: "indie", Count: minTagCount}, // at threshold
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer lastfmSrv.Close()

	mbSrv := newMBServer(t, nil)
	defer mbSrv.Close()

	p := newForTest("test-key", lastfmSrv.URL, mbSrv.URL)

	items := []core.MediaItem{
		{Platform: "spotify", Type: core.MediaMusic, Title: "Song", Creator: "Artist", ExternalID: "x"},
	}

	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !containsTag(got[0].Tags, "rock") {
		t.Errorf("Tags = %v, want rock (count=100, above threshold)", got[0].Tags)
	}
	if containsTag(got[0].Tags, "alternative") {
		t.Errorf("Tags = %v, should NOT contain alternative (count=5, below threshold)", got[0].Tags)
	}
	if !containsTag(got[0].Tags, "indie") {
		t.Errorf("Tags = %v, want indie (count=%d, at threshold)", got[0].Tags, minTagCount)
	}
}

func TestEnrichLastfmAPIError(t *testing.T) {
	// Last.fm returns an error in the JSON body (e.g., track not found).
	lastfmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"error":   6,
			"message": "Track not found",
		})
	}))
	defer lastfmSrv.Close()

	mbSrv := newMBServer(t, nil)
	defer mbSrv.Close()

	p := newForTest("test-key", lastfmSrv.URL, mbSrv.URL)

	items := []core.MediaItem{
		{Platform: "spotify", Type: core.MediaMusic, Title: "Nonexistent", Creator: "Nobody", ExternalID: "x"},
	}

	// Should not fail — API errors are non-fatal per item.
	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d items, want 1", len(got))
	}
}

func TestEnrichMBGenresApplied(t *testing.T) {
	// Last.fm returns no tags, but MusicBrainz has genres.
	lastfmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(lastfmTagsResponse{})
	}))
	defer lastfmSrv.Close()

	recordings := map[string]mbRecordingResponse{
		"mb-id-1": {
			ID:    "mb-id-1",
			Title: "Jazz Standard",
			Genres: []mbGenre{
				{Name: "jazz", Count: 10},
				{Name: "bebop", Count: 5},
			},
		},
	}
	mbSrv := newMBServer(t, recordings)
	defer mbSrv.Close()

	p := newForTest("test-key", lastfmSrv.URL, mbSrv.URL)

	items := []core.MediaItem{
		{Platform: "spotify", Type: core.MediaMusic, Title: "Jazz Standard", Creator: "Miles Davis", ExternalID: "x"},
	}

	got, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !containsTag(got[0].Tags, "jazz") {
		t.Errorf("Tags = %v, want jazz from MusicBrainz", got[0].Tags)
	}
}

func TestEnrichCompileTimeCheck(t *testing.T) {
	// Verify Provider implements EnrichmentProvider at compile time.
	var _ core.EnrichmentProvider = (*Provider)(nil)
}

// ── Tag normalization tests ─────────────────────────────────────────────

func TestNormalizeTag(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Direct matches
		{"rock", "rock"},
		{"Rock", "rock"},
		{"ROCK", "rock"},
		{"  rock  ", "rock"},

		// Alias matches
		{"Hip-Hop", "hip-hop"},
		{"hip hop", "hip-hop"},
		{"hiphop", "hip-hop"},
		{"Hip Hop", "hip-hop"},
		{"rap", "hip-hop"},

		// R&B variants
		{"R&B", "r-and-b"},
		{"r&b", "r-and-b"},
		{"rnb", "r-and-b"},
		{"rhythm and blues", "r-and-b"},

		// Sci-fi variants
		{"sci fi", "sci-fi"},
		{"scifi", "sci-fi"},
		{"Sci-Fi", "sci-fi"},

		// Electronic variants
		{"electronica", "electronic"},
		{"EDM", "electronic"},
		{"edm", "electronic"},
		{"techno", "electronic"},

		// Sub-genre → parent mappings
		{"progressive rock", "rock"},
		{"prog rock", "rock"},
		{"indie rock", "indie"},
		{"indie pop", "indie"},
		{"heavy metal", "metal"},
		{"post-punk", "punk"},
		{"pop punk", "punk"},

		// Alternative variants
		{"alt rock", "alternative"},
		{"grunge", "alternative"},
		{"shoegaze", "alternative"},

		// Other mappings
		{"bossa nova", "jazz"},
		{"neo soul", "soul"},
		{"bluegrass", "country"},
		{"cumbia", "latin"},
		{"ska", "reggae"},
		{"delta blues", "blues"},
		{"downtempo", "ambient"},
		{"chillout", "chill"},
		{"mellow", "chill"},
		{"happy", "uplifting"},
		{"sad", "melancholic"},

		// Standard normalization (no alias match)
		{"some genre", "some-genre"},
		{"a_b_c", "a-b-c"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeTag(tt.input)
			if got != tt.want {
				t.Errorf("normalizeTag(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeTagOnlyValidTags(t *testing.T) {
	// Test that all tag aliases map to valid tags.
	for input, expected := range tagAliases {
		if !core.IsValidTag(expected) {
			t.Errorf("tagAliases[%q] = %q, which is not a valid tag", input, expected)
		}
	}
}

// ── Rate limiter tests ──────────────────────────────────────────────────

func TestRateLimiterZeroInterval(t *testing.T) {
	rl := newRateLimiter(0)
	// Should not block.
	for i := 0; i < 100; i++ {
		if err := rl.wait(context.Background()); err != nil {
			t.Fatalf("wait() error on iteration %d: %v", i, err)
		}
	}
}

func TestRateLimiterContextCancel(t *testing.T) {
	rl := newRateLimiter(time.Hour) // very slow, would block forever
	defer close(rl.stopCh)

	// Drain the initial token.
	if err := rl.wait(context.Background()); err != nil {
		t.Fatalf("first wait() error: %v", err)
	}

	// Second call should block until context is cancelled.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := rl.wait(ctx)
	if err == nil {
		t.Error("expected context deadline exceeded, got nil")
	}
}

// ── mergeUnique tests ───────────────────────────────────────────────────

func TestMergeUnique(t *testing.T) {
	tests := []struct {
		name string
		a, b []string
		want []string
	}{
		{"both nil", nil, nil, nil},
		{"a nil", nil, []string{"x"}, []string{"x"}},
		{"b nil", []string{"x"}, nil, []string{"x"}},
		{"no overlap", []string{"a", "b"}, []string{"c", "d"}, []string{"a", "b", "c", "d"}},
		{"overlap", []string{"a", "b"}, []string{"b", "c"}, []string{"a", "b", "c"}},
		{"all same", []string{"a", "b"}, []string{"a", "b"}, []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeUnique(tt.a, tt.b)
			if len(got) != len(tt.want) {
				t.Errorf("mergeUnique() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("mergeUnique()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// ── checkHTTPError tests ────────────────────────────────────────────────

func TestCheckHTTPError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantCode   core.ErrorCode
		wantRetry  bool
		wantNil    bool
	}{
		{"200 OK", 200, "", false, true},
		{"429 rate limit", 429, core.ErrRateLimit, true, false},
		{"500 server error", 500, core.ErrUpstream, true, false},
		{"503 unavailable", 503, core.ErrUpstream, true, false},
		{"400 bad request", 400, core.ErrUpstream, false, false},
		{"404 not found", 404, core.ErrUpstream, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Body:       http.NoBody,
			}
			err := checkHTTPError(resp, "TestAPI")

			if tt.wantNil {
				if err != nil {
					t.Errorf("checkHTTPError() = %v, want nil", err)
				}
				return
			}

			if err == nil {
				t.Fatal("checkHTTPError() = nil, want error")
			}

			pe, ok := err.(*core.PluginError)
			if !ok {
				t.Fatalf("error type = %T, want *core.PluginError", err)
			}
			if pe.Code != tt.wantCode {
				t.Errorf("error code = %q, want %q", pe.Code, tt.wantCode)
			}
			if pe.Retry != tt.wantRetry {
				t.Errorf("Retry = %v, want %v", pe.Retry, tt.wantRetry)
			}
		})
	}
}
