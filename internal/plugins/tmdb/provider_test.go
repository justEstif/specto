package tmdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/justestif/specto/internal/core"
)

// --- Test helpers ---

// newTestProvider creates a Provider pointing at a test server.
func newTestProvider(ts *httptest.Server) *Provider {
	return NewWithBaseURL("test-api-key", ts.URL)
}

// makeItem creates a video MediaItem for testing.
func makeItem(title, platform string) core.MediaItem {
	return core.MediaItem{
		Platform:   platform,
		Type:       core.MediaVideo,
		Title:      title,
		ExternalID: "ext-" + strings.ToLower(strings.ReplaceAll(title, " ", "-")),
		ConsumedAt: time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC),
	}
}

// mustJSON marshals v to JSON or panics.
func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// --- Fake TMDB responses ---

func movieSearchResponse(id int, title, releaseDate string) searchResponse {
	return searchResponse{
		Results: []searchResult{
			{
				ID:          id,
				Title:       title,
				ReleaseDate: releaseDate,
				Popularity:  50.0,
			},
		},
		TotalPages: 1,
	}
}

func tvSearchResponse(id int, name, firstAirDate string) searchResponse {
	return searchResponse{
		Results: []searchResult{
			{
				ID:           id,
				Name:         name,
				FirstAirDate: firstAirDate,
				Popularity:   50.0,
			},
		},
		TotalPages: 1,
	}
}

func movieDetailsResponseJSON(id int, genres []tmdbGenre, runtime int, keywords []tmdbKeyword) []byte {
	return mustJSON(movieDetailsResponse{
		ID:      id,
		Genres:  genres,
		Runtime: runtime,
		Keywords: keywordsWrapper{
			Keywords: keywords,
		},
	})
}

func tvDetailsResponseJSON(id int, genres []tmdbGenre, tvType string, keywords []tmdbKeyword) []byte {
	return mustJSON(tvDetailsResponse{
		ID:             id,
		Genres:         genres,
		Type:           tvType,
		EpisodeRunTime: []int{45},
		Keywords: keywordsWrapper{
			Results: keywords,
		},
		NumberOfSeasons: 3,
	})
}

// --- Tests ---

func TestName(t *testing.T) {
	p := New("test-key")
	if p.Name() != "tmdb" {
		t.Errorf("Name() = %q, want %q", p.Name(), "tmdb")
	}
}

func TestSupports(t *testing.T) {
	p := New("test-key")

	tests := []struct {
		mediaType string
		platform  string
		want      bool
	}{
		{"video", "netflix", true},
		{"video", "youtube", true},
		{"video", "", true},
		{"video", "crunchyroll", false},
		{"video", "funimation", false},
		{"video", "hidive", false},
		{"music", "spotify", false},
		{"article", "pocket", false},
		{"podcast", "apple", false},
	}

	for _, tt := range tests {
		got := p.Supports(tt.mediaType, tt.platform)
		if got != tt.want {
			t.Errorf("Supports(%q, %q) = %v, want %v", tt.mediaType, tt.platform, got, tt.want)
		}
	}
}

func TestEnrichMovie(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/search/movie"):
			resp := movieSearchResponse(550, "Fight Club", "1999-10-15")
			w.Write(mustJSON(resp))

		case strings.HasPrefix(r.URL.Path, "/movie/550"):
			genres := []tmdbGenre{
				{ID: 18, Name: "Drama"},
				{ID: 53, Name: "Thriller"},
			}
			keywords := []tmdbKeyword{
				{ID: 1, Name: "dual identity"},
				{ID: 2, Name: "philosophy"},
			}
			w.Write(movieDetailsResponseJSON(550, genres, 139, keywords))

		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	p := newTestProvider(ts)
	items := []core.MediaItem{makeItem("Fight Club", "netflix")}

	enriched, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("Enrich() error: %v", err)
	}

	if len(enriched) != 1 {
		t.Fatalf("expected 1 enriched item, got %d", len(enriched))
	}

	item := enriched[0]

	// Check genre tags.
	assertHasTag(t, item.Tags, "drama")
	assertHasTag(t, item.Tags, "thriller")

	// "philosophy" is a valid topic tag.
	assertHasTag(t, item.Tags, "philosophy")

	// "dual identity" normalizes to "dual-identity" which is not a valid tag — should be absent.
	assertNoTag(t, item.Tags, "dual-identity")

	// Format tag.
	assertHasTag(t, item.Tags, "film")

	// TMDB metadata stored.
	if item.RawMetadata["tmdb_id"] != 550 {
		t.Errorf("tmdb_id = %v, want 550", item.RawMetadata["tmdb_id"])
	}
	if item.RawMetadata["tmdb_type"] != "movie" {
		t.Errorf("tmdb_type = %v, want %q", item.RawMetadata["tmdb_type"], "movie")
	}
}

func TestEnrichTV(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/search/movie"):
			// No movie results — force fallback to TV.
			w.Write(mustJSON(searchResponse{Results: nil}))

		case strings.HasPrefix(r.URL.Path, "/search/tv"):
			resp := tvSearchResponse(1399, "Breaking Bad", "2008-01-20")
			w.Write(mustJSON(resp))

		case strings.HasPrefix(r.URL.Path, "/tv/1399"):
			genres := []tmdbGenre{
				{ID: 18, Name: "Drama"},
				{ID: 80, Name: "Crime"},
			}
			keywords := []tmdbKeyword{
				{ID: 10, Name: "chemistry"},
			}
			w.Write(tvDetailsResponseJSON(1399, genres, "Scripted", keywords))

		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	p := newTestProvider(ts)
	items := []core.MediaItem{makeItem("Breaking Bad", "netflix")}

	enriched, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("Enrich() error: %v", err)
	}

	item := enriched[0]

	assertHasTag(t, item.Tags, "drama")
	assertHasTag(t, item.Tags, "crime")
	assertHasTag(t, item.Tags, "series")

	if item.RawMetadata["tmdb_type"] != "tv" {
		t.Errorf("tmdb_type = %v, want %q", item.RawMetadata["tmdb_type"], "tv")
	}
}

func TestEnrichTVMiniseries(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/search/movie"):
			w.Write(mustJSON(searchResponse{Results: nil}))

		case strings.HasPrefix(r.URL.Path, "/search/tv"):
			resp := tvSearchResponse(100, "Chernobyl", "2019-05-06")
			w.Write(mustJSON(resp))

		case strings.HasPrefix(r.URL.Path, "/tv/100"):
			genres := []tmdbGenre{{ID: 18, Name: "Drama"}}
			w.Write(tvDetailsResponseJSON(100, genres, "Miniseries", nil))

		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	p := newTestProvider(ts)
	items := []core.MediaItem{makeItem("Chernobyl", "hbo")}

	enriched, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("Enrich() error: %v", err)
	}

	assertHasTag(t, enriched[0].Tags, "mini-series")
	assertHasTag(t, enriched[0].Tags, "drama")
}

func TestEnrichShortFilm(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/search/movie"):
			resp := movieSearchResponse(999, "Short Film", "2024-01-01")
			w.Write(mustJSON(resp))

		case strings.HasPrefix(r.URL.Path, "/movie/999"):
			genres := []tmdbGenre{{ID: 18, Name: "Drama"}}
			w.Write(movieDetailsResponseJSON(999, genres, 20, nil))

		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	p := newTestProvider(ts)
	items := []core.MediaItem{makeItem("Short Film", "youtube")}

	enriched, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("Enrich() error: %v", err)
	}

	assertHasTag(t, enriched[0].Tags, "short-film")
	assertNoTag(t, enriched[0].Tags, "film")
}

func TestEnrichNoMatch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return empty results for all searches.
		w.Write(mustJSON(searchResponse{Results: nil}))
	}))
	defer ts.Close()

	p := newTestProvider(ts)
	items := []core.MediaItem{makeItem("xyznonexistent123", "netflix")}

	enriched, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("Enrich() error: %v", err)
	}

	// Item should be returned unchanged.
	if len(enriched[0].Tags) != 0 {
		t.Errorf("expected no tags for unmatched item, got %v", enriched[0].Tags)
	}
	if enriched[0].RawMetadata != nil {
		t.Errorf("expected nil RawMetadata for unmatched item, got %v", enriched[0].RawMetadata)
	}
}

func TestEnrichRateLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"status_message":"rate limit"}`))
	}))
	defer ts.Close()

	p := newTestProvider(ts)
	items := []core.MediaItem{makeItem("Some Movie", "netflix")}

	enriched, err := p.Enrich(context.Background(), items)
	// Per-item failures are skipped, so err should be nil.
	if err != nil {
		t.Fatalf("Enrich() error: %v", err)
	}

	// Item should be returned unchanged (rate limit skipped it).
	if len(enriched[0].Tags) != 0 {
		t.Errorf("expected no tags after rate limit, got %v", enriched[0].Tags)
	}
}

func TestEnrichServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status_message":"internal error"}`))
	}))
	defer ts.Close()

	p := newTestProvider(ts)
	items := []core.MediaItem{makeItem("Some Movie", "netflix")}

	enriched, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("Enrich() error: %v", err)
	}

	// Item returned unchanged.
	if len(enriched[0].Tags) != 0 {
		t.Errorf("expected no tags after server error, got %v", enriched[0].Tags)
	}
}

func TestEnrichMultipleItemsPartialFailure(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/search/movie"):
			query := r.URL.Query().Get("query")
			if query == "Good Movie" {
				w.Write(mustJSON(movieSearchResponse(1, "Good Movie", "2024-01-01")))
			} else {
				// "Bad Movie" triggers a 500 on search.
				callCount++
				if callCount <= 2 {
					// First two search calls (movie + tv) for "Bad Movie" fail.
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error":"server error"}`))
					return
				}
				w.Write(mustJSON(searchResponse{Results: nil}))
			}

		case strings.HasPrefix(r.URL.Path, "/search/tv"):
			query := r.URL.Query().Get("query")
			if query == "Good Movie" {
				w.Write(mustJSON(searchResponse{Results: nil}))
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"server error"}`))
			}

		case strings.HasPrefix(r.URL.Path, "/movie/1"):
			genres := []tmdbGenre{{ID: 35, Name: "Comedy"}}
			w.Write(movieDetailsResponseJSON(1, genres, 100, nil))

		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	p := newTestProvider(ts)
	items := []core.MediaItem{
		makeItem("Bad Movie", "netflix"),
		makeItem("Good Movie", "netflix"),
	}

	enriched, err := p.Enrich(context.Background(), items)
	if err != nil {
		t.Fatalf("Enrich() error: %v", err)
	}

	// First item should be unchanged (failed).
	if len(enriched[0].Tags) != 0 {
		t.Errorf("expected no tags for failed item, got %v", enriched[0].Tags)
	}

	// Second item should be enriched.
	assertHasTag(t, enriched[1].Tags, "comedy")
	assertHasTag(t, enriched[1].Tags, "film")
}

func TestGenreMapping(t *testing.T) {
	tests := []struct {
		genreID  int
		wantTags []string
	}{
		{28, []string{"action"}},
		{12, []string{"adventure"}},
		{16, []string{"animation"}},
		{35, []string{"comedy"}},
		{80, []string{"crime"}},
		{99, []string{"documentary"}},
		{18, []string{"drama"}},
		{14, []string{"fantasy"}},
		{27, []string{"horror"}},
		{10402, []string{"musical"}},
		{9648, []string{"mystery"}},
		{10749, []string{"romance"}},
		{878, []string{"sci-fi"}},
		{53, []string{"thriller"}},
		{37, []string{"western"}},
		{10770, []string{"film"}},
		{10759, []string{"action", "adventure"}},
		{10765, []string{"sci-fi", "fantasy"}},
		{9999, nil}, // unknown genre
	}

	for _, tt := range tests {
		got := mapGenreID(tt.genreID)
		if len(got) != len(tt.wantTags) {
			t.Errorf("mapGenreID(%d) = %v, want %v", tt.genreID, got, tt.wantTags)
			continue
		}
		for i, g := range got {
			if g != tt.wantTags[i] {
				t.Errorf("mapGenreID(%d)[%d] = %q, want %q", tt.genreID, i, g, tt.wantTags[i])
			}
		}
	}
}

func TestKeywordNormalization(t *testing.T) {
	tests := []struct {
		keyword string
		want    string
	}{
		{"Artificial Intelligence", "artificial-intelligence"},
		{"sci-fi", "sci-fi"},
		{"  drama  ", "drama"},
		{"Pop Culture", "pop-culture"},
		{"AI", "ai"},
	}

	for _, tt := range tests {
		got := normalizeKeyword(tt.keyword)
		if got != tt.want {
			t.Errorf("normalizeKeyword(%q) = %q, want %q", tt.keyword, got, tt.want)
		}
	}
}

func TestKeywordToValidTag(t *testing.T) {
	// Keywords that should map to valid tags.
	validKeywords := []string{"philosophy", "ai", "science", "history"}
	for _, kw := range validKeywords {
		normalized := normalizeKeyword(kw)
		if !core.IsValidTag(normalized) {
			t.Errorf("expected %q (from keyword %q) to be a valid tag", normalized, kw)
		}
	}

	// Keywords that should NOT map to valid tags.
	invalidKeywords := []string{"dual identity", "based on novel", "revenge"}
	for _, kw := range invalidKeywords {
		normalized := normalizeKeyword(kw)
		if core.IsValidTag(normalized) {
			t.Errorf("expected %q (from keyword %q) to NOT be a valid tag", normalized, kw)
		}
	}
}

func TestDetermineFormat(t *testing.T) {
	tests := []struct {
		mediaType  string
		tmdbType   string
		runtimeMin int
		want       string
	}{
		{"movie", "", 120, "film"},
		{"movie", "", 20, "short-film"},
		{"movie", "", 0, "film"},
		{"tv", "Scripted", 45, "series"},
		{"tv", "Miniseries", 45, "mini-series"},
		{"tv", "", 0, "series"},
	}

	for _, tt := range tests {
		got := determineFormat(tt.mediaType, tt.tmdbType, tt.runtimeMin)
		if got != tt.want {
			t.Errorf("determineFormat(%q, %q, %d) = %q, want %q",
				tt.mediaType, tt.tmdbType, tt.runtimeMin, got, tt.want)
		}
	}
}

func TestGuessIsTV(t *testing.T) {
	tests := []struct {
		name string
		item core.MediaItem
		want bool
	}{
		{
			name: "explicit tmdb_type tv",
			item: core.MediaItem{
				Title:       "Breaking Bad",
				RawMetadata: map[string]any{"tmdb_type": "tv"},
			},
			want: true,
		},
		{
			name: "netflix series metadata",
			item: core.MediaItem{
				Title:       "Stranger Things",
				RawMetadata: map[string]any{"type": "TV Series"},
			},
			want: true,
		},
		{
			name: "title with season indicator",
			item: core.MediaItem{
				Title: "The Office S01E01",
			},
			want: true,
		},
		{
			name: "title with episode word",
			item: core.MediaItem{
				Title: "Podcast Episode 5",
			},
			want: true,
		},
		{
			name: "regular movie title",
			item: core.MediaItem{
				Title: "Inception",
			},
			want: false,
		},
		{
			name: "no metadata",
			item: core.MediaItem{
				Title: "Fight Club",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := guessIsTV(tt.item)
			if got != tt.want {
				t.Errorf("guessIsTV() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPickBestMatch(t *testing.T) {
	results := []searchResult{
		{ID: 1, Title: "The Matrix", ReleaseDate: "1999-03-31", Popularity: 80},
		{ID: 2, Title: "The Matrix Reloaded", ReleaseDate: "2003-05-15", Popularity: 60},
		{ID: 3, Title: "The Matrix Revolutions", ReleaseDate: "2003-11-05", Popularity: 50},
	}

	got := pickBestMatch(results, "The Matrix", 1999, "movie")
	if got == nil {
		t.Fatal("pickBestMatch returned nil")
	}
	if got.id != 1 {
		t.Errorf("pickBestMatch().id = %d, want 1", got.id)
	}
	if got.title != "The Matrix" {
		t.Errorf("pickBestMatch().title = %q, want %q", got.title, "The Matrix")
	}
}

func TestPickBestMatchNoResults(t *testing.T) {
	got := pickBestMatch(nil, "Nonexistent", 2024, "movie")
	if got != nil {
		t.Errorf("expected nil for empty results, got %v", got)
	}
}

func TestPickBestMatchTV(t *testing.T) {
	results := []searchResult{
		{ID: 10, Name: "Breaking Bad", FirstAirDate: "2008-01-20", Popularity: 90},
		{ID: 11, Name: "Bad Monkey", FirstAirDate: "2024-08-14", Popularity: 30},
	}

	got := pickBestMatch(results, "Breaking Bad", 2008, "tv")
	if got == nil {
		t.Fatal("pickBestMatch returned nil")
	}
	if got.id != 10 {
		t.Errorf("pickBestMatch().id = %d, want 10", got.id)
	}
}

func TestEnrichTVHintInMetadata(t *testing.T) {
	// Item with metadata suggesting it's a TV show — should try TV search first.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/search/tv"):
			resp := tvSearchResponse(200, "Stranger Things", "2016-07-15")
			w.Write(mustJSON(resp))

		case strings.HasPrefix(r.URL.Path, "/tv/200"):
			genres := []tmdbGenre{
				{ID: 18, Name: "Drama"},
				{ID: 10765, Name: "Sci-Fi & Fantasy"},
			}
			w.Write(tvDetailsResponseJSON(200, genres, "Scripted", nil))

		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	p := newTestProvider(ts)
	item := makeItem("Stranger Things", "netflix")
	item.RawMetadata = map[string]any{"type": "TV Series"}

	enriched, err := p.Enrich(context.Background(), []core.MediaItem{item})
	if err != nil {
		t.Fatalf("Enrich() error: %v", err)
	}

	assertHasTag(t, enriched[0].Tags, "drama")
	assertHasTag(t, enriched[0].Tags, "sci-fi")
	assertHasTag(t, enriched[0].Tags, "fantasy")
	assertHasTag(t, enriched[0].Tags, "series")

	if enriched[0].RawMetadata["tmdb_type"] != "tv" {
		t.Errorf("tmdb_type = %v, want %q", enriched[0].RawMetadata["tmdb_type"], "tv")
	}
}

func TestEnrichPreservesExistingTags(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/search/movie"):
			resp := movieSearchResponse(1, "Test Movie", "2024-01-01")
			w.Write(mustJSON(resp))

		case strings.HasPrefix(r.URL.Path, "/movie/1"):
			genres := []tmdbGenre{{ID: 35, Name: "Comedy"}}
			w.Write(movieDetailsResponseJSON(1, genres, 100, nil))

		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	p := newTestProvider(ts)
	item := makeItem("Test Movie", "netflix")
	item.Tags = []string{"existing-tag", "comedy"} // comedy is a dupe

	enriched, err := p.Enrich(context.Background(), []core.MediaItem{item})
	if err != nil {
		t.Fatalf("Enrich() error: %v", err)
	}

	// Existing tags preserved.
	assertHasTag(t, enriched[0].Tags, "existing-tag")
	// Duplicate not added twice.
	count := 0
	for _, tag := range enriched[0].Tags {
		if tag == "comedy" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected comedy tag once, found %d times", count)
	}
	// New tag added.
	assertHasTag(t, enriched[0].Tags, "film")
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		header string
		want   time.Duration
	}{
		{"", 10 * time.Second},
		{"5", 5 * time.Second},
		{"invalid", 10 * time.Second},
		{"-1", 10 * time.Second},
		{"0", 10 * time.Second},
	}

	for _, tt := range tests {
		got := parseRetryAfter(tt.header)
		if got != tt.want {
			t.Errorf("parseRetryAfter(%q) = %v, want %v", tt.header, got, tt.want)
		}
	}
}

func TestExtractYear(t *testing.T) {
	tests := []struct {
		dates []string
		want  int
	}{
		{[]string{"2024-01-15"}, 2024},
		{[]string{""}, 0},
		{[]string{"", "2020-06-01"}, 2020},
		{[]string{"invalid"}, 0},
	}

	for _, tt := range tests {
		got := extractYear(tt.dates...)
		if got != tt.want {
			t.Errorf("extractYear(%v) = %d, want %d", tt.dates, got, tt.want)
		}
	}
}

// --- Assertions ---

func assertHasTag(t *testing.T, tags []string, want string) {
	t.Helper()
	for _, tag := range tags {
		if tag == want {
			return
		}
	}
	t.Errorf("expected tag %q in %v", want, tags)
}

func assertNoTag(t *testing.T, tags []string, unwanted string) {
	t.Helper()
	for _, tag := range tags {
		if tag == unwanted {
			t.Errorf("unexpected tag %q in %v", unwanted, tags)
			return
		}
	}
}
