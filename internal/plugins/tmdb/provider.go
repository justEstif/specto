// Package tmdb implements an EnrichmentProvider that uses The Movie Database
// (TMDB) API to enrich video media items with genre, keyword, and format tags.
package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/justestif/specto/internal/core"
)

// Compile-time interface check.
var _ core.EnrichmentProvider = (*Provider)(nil)

const (
	defaultBaseURL    = "https://api.themoviedb.org/3"
	defaultTimeout    = 15 * time.Second
	defaultConcurrent = 4 // max concurrent TMDB requests (~40 req/10s budget)
)

// Provider enriches video items by searching TMDB for genre, keyword, and
// format metadata. It implements core.EnrichmentProvider.
type Provider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	sem        *semaphore.Weighted
}

// New creates a TMDB enrichment provider with the given API key.
func New(apiKey string) *Provider {
	return &Provider{
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
		sem:        semaphore.NewWeighted(defaultConcurrent),
	}
}

// NewWithBaseURL creates a Provider pointing at a custom base URL.
// Intended for testing with httptest.Server.
func NewWithBaseURL(apiKey, baseURL string) *Provider {
	return &Provider{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
		sem:        semaphore.NewWeighted(defaultConcurrent),
	}
}

// Name returns the unique identifier for this provider.
func (p *Provider) Name() string { return "tmdb" }

// Supports returns true for all video items regardless of platform.
func (p *Provider) Supports(mediaType string, _ string) bool {
	return mediaType == string(core.MediaVideo)
}

// Enrich searches TMDB for each item, retrieves genre and keyword data,
// and maps them to fixed tags. Per-item failures are skipped.
func (p *Provider) Enrich(ctx context.Context, items []core.MediaItem) ([]core.MediaItem, error) {
	for i, item := range items {
		if err := p.sem.Acquire(ctx, 1); err != nil {
			return items, &core.PluginError{
				Code:    core.ErrUpstream,
				Message: "context cancelled while waiting for rate limiter",
				Raw:     err,
			}
		}

		enriched, err := p.enrichItem(ctx, item)
		p.sem.Release(1)

		if err != nil {
			// Per-item failures do not abort the batch.
			continue
		}
		items[i] = enriched
	}
	return items, nil
}

// enrichItem searches TMDB and enriches a single item.
func (p *Provider) enrichItem(ctx context.Context, item core.MediaItem) (core.MediaItem, error) {
	year := 0
	if !item.ConsumedAt.IsZero() {
		year = item.ConsumedAt.Year()
	}

	isTV := guessIsTV(item)

	var result *tmdbResult
	var err error

	if isTV {
		// Try TV first, fall back to movie.
		result, err = p.searchTV(ctx, item.Title, year)
		if err != nil || result == nil {
			result, err = p.searchMovie(ctx, item.Title, year)
		}
	} else {
		// Try movie first, fall back to TV.
		result, err = p.searchMovie(ctx, item.Title, year)
		if err != nil || result == nil {
			result, err = p.searchTV(ctx, item.Title, year)
		}
	}

	if err != nil {
		return item, err
	}
	if result == nil {
		// No match found — not an error, just skip.
		return item, nil
	}

	// Fetch details (genres + keywords).
	details, err := p.getDetails(ctx, result.id, result.mediaType)
	if err != nil {
		return item, err
	}

	// Map genres to tags.
	var tags []string
	for _, gid := range details.genreIDs {
		mapped := mapGenreID(gid)
		for _, t := range mapped {
			if core.IsValidTag(t) {
				tags = append(tags, t)
			}
		}
	}

	// Map keywords to tags.
	for _, kw := range details.keywords {
		normalized := normalizeKeyword(kw)
		if core.IsValidTag(normalized) {
			tags = append(tags, normalized)
		}
	}

	// Add format tag.
	formatTag := determineFormat(result.mediaType, details.tmdbType, details.runtime)
	if formatTag != "" && core.IsValidTag(formatTag) {
		tags = append(tags, formatTag)
	}

	// Merge tags (deduplicate).
	item.Tags = mergeUnique(item.Tags, tags)

	// Store TMDB ID in RawMetadata.
	if item.RawMetadata == nil {
		item.RawMetadata = make(map[string]any)
	}
	item.RawMetadata["tmdb_id"] = result.id
	item.RawMetadata["tmdb_type"] = result.mediaType

	return item, nil
}

// --- TMDB API types ---

type searchResponse struct {
	Results    []searchResult `json:"results"`
	TotalPages int            `json:"total_pages"`
}

type searchResult struct {
	ID           int     `json:"id"`
	Title        string  `json:"title"`          // movie
	Name         string  `json:"name"`           // TV
	ReleaseDate  string  `json:"release_date"`   // movie: "2024-01-15"
	FirstAirDate string  `json:"first_air_date"` // TV
	Popularity   float64 `json:"popularity"`
}

type movieDetailsResponse struct {
	ID       int             `json:"id"`
	Genres   []tmdbGenre     `json:"genres"`
	Runtime  int             `json:"runtime"` // minutes
	Keywords keywordsWrapper `json:"keywords"`
}

type tvDetailsResponse struct {
	ID              int             `json:"id"`
	Genres          []tmdbGenre     `json:"genres"`
	Type            string          `json:"type"` // "Scripted", "Miniseries", etc.
	EpisodeRunTime  []int           `json:"episode_run_time"`
	Keywords        keywordsWrapper `json:"keywords"`
	NumberOfSeasons int             `json:"number_of_seasons"`
}

type tmdbGenre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type keywordsWrapper struct {
	Keywords []tmdbKeyword `json:"keywords"` // movie
	Results  []tmdbKeyword `json:"results"`  // TV
}

type tmdbKeyword struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// tmdbResult holds a matched search result.
type tmdbResult struct {
	id        int
	mediaType string // "movie" or "tv"
	title     string
}

// tmdbDetails holds fetched details for a movie or TV show.
type tmdbDetails struct {
	genreIDs []int
	keywords []string
	runtime  int    // minutes (0 if unknown)
	tmdbType string // TV-specific type: "Scripted", "Miniseries", etc.
}

// --- Search ---

func (p *Provider) searchMovie(ctx context.Context, title string, year int) (*tmdbResult, error) {
	params := url.Values{
		"api_key": {p.apiKey},
		"query":   {title},
	}
	if year > 0 {
		params.Set("year", strconv.Itoa(year))
	}

	reqURL := p.baseURL + "/search/movie?" + params.Encode()
	body, err := p.doGet(ctx, reqURL)
	if err != nil {
		return nil, err
	}

	var resp searchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, &core.PluginError{
			Code:    core.ErrInvalidData,
			Message: fmt.Sprintf("decoding TMDB movie search: %s", err),
			Raw:     err,
		}
	}

	best := pickBestMatch(resp.Results, title, year, "movie")
	return best, nil
}

func (p *Provider) searchTV(ctx context.Context, title string, year int) (*tmdbResult, error) {
	params := url.Values{
		"api_key": {p.apiKey},
		"query":   {title},
	}
	if year > 0 {
		params.Set("first_air_date_year", strconv.Itoa(year))
	}

	reqURL := p.baseURL + "/search/tv?" + params.Encode()
	body, err := p.doGet(ctx, reqURL)
	if err != nil {
		return nil, err
	}

	var resp searchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, &core.PluginError{
			Code:    core.ErrInvalidData,
			Message: fmt.Sprintf("decoding TMDB TV search: %s", err),
			Raw:     err,
		}
	}

	best := pickBestMatch(resp.Results, title, year, "tv")
	return best, nil
}

// --- Details ---

func (p *Provider) getDetails(ctx context.Context, id int, mediaType string) (*tmdbDetails, error) {
	path := fmt.Sprintf("/%s/%d", mediaType, id)
	params := url.Values{
		"api_key":            {p.apiKey},
		"append_to_response": {"keywords"},
	}

	reqURL := p.baseURL + path + "?" + params.Encode()
	body, err := p.doGet(ctx, reqURL)
	if err != nil {
		return nil, err
	}

	details := &tmdbDetails{}

	if mediaType == "movie" {
		var resp movieDetailsResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, &core.PluginError{
				Code:    core.ErrInvalidData,
				Message: fmt.Sprintf("decoding TMDB movie details: %s", err),
				Raw:     err,
			}
		}
		for _, g := range resp.Genres {
			details.genreIDs = append(details.genreIDs, g.ID)
		}
		for _, kw := range resp.Keywords.Keywords {
			details.keywords = append(details.keywords, kw.Name)
		}
		details.runtime = resp.Runtime
	} else {
		var resp tvDetailsResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, &core.PluginError{
				Code:    core.ErrInvalidData,
				Message: fmt.Sprintf("decoding TMDB TV details: %s", err),
				Raw:     err,
			}
		}
		for _, g := range resp.Genres {
			details.genreIDs = append(details.genreIDs, g.ID)
		}
		// TV uses "results" instead of "keywords" in the keywords wrapper.
		for _, kw := range resp.Keywords.Results {
			details.keywords = append(details.keywords, kw.Name)
		}
		if len(resp.EpisodeRunTime) > 0 {
			details.runtime = resp.EpisodeRunTime[0]
		}
		details.tmdbType = resp.Type
	}

	return details, nil
}

// --- HTTP ---

// doGet performs a GET request and returns the response body. It handles
// TMDB-specific HTTP error codes and maps them to PluginError.
func (p *Provider) doGet(ctx context.Context, reqURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("building TMDB request: %s", err),
			Raw:     err,
		}
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("calling TMDB API: %s", err),
			Retry:   true,
			Raw:     err,
		}
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		return nil, &core.PluginError{
			Code:    core.ErrRateLimit,
			Message: "TMDB rate limit exceeded",
			Retry:   true,
			After:   retryAfter,
		}
	case resp.StatusCode >= 500:
		body, _ := io.ReadAll(resp.Body)
		return nil, &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("TMDB returned %d: %s", resp.StatusCode, truncate(string(body), 200)),
			Retry:   true,
		}
	case resp.StatusCode != http.StatusOK:
		body, _ := io.ReadAll(resp.Body)
		return nil, &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("TMDB returned %d: %s", resp.StatusCode, truncate(string(body), 200)),
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("reading TMDB response: %s", err),
			Raw:     err,
		}
	}

	return body, nil
}

// parseRetryAfter parses a Retry-After header (seconds) into a Duration.
// Defaults to 10 seconds if the header is missing or invalid.
func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return 10 * time.Second
	}
	secs, err := strconv.Atoi(header)
	if err != nil || secs <= 0 {
		return 10 * time.Second
	}
	return time.Duration(secs) * time.Second
}

// --- Matching ---

// pickBestMatch selects the best search result based on title similarity
// and year matching. Returns nil if no reasonable match is found.
func pickBestMatch(results []searchResult, queryTitle string, queryYear int, mediaType string) *tmdbResult {
	if len(results) == 0 {
		return nil
	}

	queryLower := strings.ToLower(strings.TrimSpace(queryTitle))

	var bestResult *searchResult
	var bestScore int

	for i := range results {
		r := &results[i]

		resultTitle := r.Title
		if mediaType == "tv" {
			resultTitle = r.Name
		}
		resultLower := strings.ToLower(strings.TrimSpace(resultTitle))

		score := 0

		// Exact title match is the strongest signal.
		if resultLower == queryLower {
			score += 100
		} else if strings.Contains(resultLower, queryLower) || strings.Contains(queryLower, resultLower) {
			score += 50
		} else {
			// No meaningful title overlap — skip.
			continue
		}

		// Year matching bonus.
		resultYear := extractYear(r.ReleaseDate, r.FirstAirDate)
		if queryYear > 0 && resultYear > 0 {
			diff := queryYear - resultYear
			if diff < 0 {
				diff = -diff
			}
			if diff == 0 {
				score += 30
			} else if diff <= 1 {
				score += 15
			} else if diff <= 2 {
				score += 5
			}
		}

		// Popularity tiebreaker.
		if r.Popularity > 10 {
			score += 5
		}

		if score > bestScore {
			bestScore = score
			bestResult = r
		}
	}

	if bestResult == nil {
		return nil
	}

	title := bestResult.Title
	if mediaType == "tv" {
		title = bestResult.Name
	}

	return &tmdbResult{
		id:        bestResult.ID,
		mediaType: mediaType,
		title:     title,
	}
}

// extractYear pulls the year from a TMDB date string ("2024-01-15") or returns 0.
func extractYear(dates ...string) int {
	for _, d := range dates {
		if len(d) >= 4 {
			if y, err := strconv.Atoi(d[:4]); err == nil && y > 1800 {
				return y
			}
		}
	}
	return 0
}

// --- Genre mapping ---

// genreMap maps TMDB numeric genre IDs to fixed tags from core.
var genreMap = map[int][]string{
	28:    {"action"},
	12:    {"adventure"},
	16:    {"animation"},
	35:    {"comedy"},
	80:    {"crime"},
	99:    {"documentary"},
	18:    {"drama"},
	14:    {"fantasy"},
	27:    {"horror"},
	10402: {"musical"},
	9648:  {"mystery"},
	10749: {"romance"},
	878:   {"sci-fi"},
	53:    {"thriller"},
	37:    {"western"},
	10770: {"film"},
	// TV-specific composite genres
	10759: {"action", "adventure"},
	10765: {"sci-fi", "fantasy"},
}

// mapGenreID returns the fixed tags for a TMDB genre ID.
func mapGenreID(id int) []string {
	if tags, ok := genreMap[id]; ok {
		return tags
	}
	return nil
}

// --- Keyword mapping ---

// normalizeKeyword converts a TMDB keyword into a candidate tag by
// lowercasing and replacing spaces with hyphens.
func normalizeKeyword(keyword string) string {
	k := strings.ToLower(strings.TrimSpace(keyword))
	k = strings.ReplaceAll(k, " ", "-")
	return k
}

// --- Format determination ---

// determineFormat returns the appropriate format tag for a TMDB result.
func determineFormat(mediaType string, tmdbType string, runtimeMin int) string {
	if mediaType == "tv" {
		lower := strings.ToLower(tmdbType)
		if lower == "miniseries" {
			return "mini-series"
		}
		return "series"
	}

	// Movie types
	if runtimeMin > 0 && runtimeMin < 40 {
		return "short-film"
	}
	return "film"
}

// --- TV detection heuristics ---

// guessIsTV inspects the item's RawMetadata and title for clues that it's a
// TV show rather than a movie.
func guessIsTV(item core.MediaItem) bool {
	if item.RawMetadata != nil {
		// Check for explicit TMDB type from a previous enrichment.
		if t, ok := item.RawMetadata["tmdb_type"].(string); ok {
			return t == "tv"
		}

		// Check raw metadata for "series" hints (e.g., Netflix items).
		for _, key := range []string{"type", "content_type", "show_type"} {
			if v, ok := item.RawMetadata[key].(string); ok {
				lower := strings.ToLower(v)
				if strings.Contains(lower, "series") || strings.Contains(lower, "show") || strings.Contains(lower, "tv") {
					return true
				}
			}
		}
	}

	// Title heuristics: season/episode indicators.
	lower := strings.ToLower(item.Title)
	indicators := []string{" s0", " s1", " s2", " s3", " season ", " episode ", " ep.", " ep "}
	for _, ind := range indicators {
		if strings.Contains(lower, ind) {
			return true
		}
	}

	return false
}

// --- Helpers ---

// mergeUnique appends new tags to existing, skipping duplicates.
func mergeUnique(existing, add []string) []string {
	seen := make(map[string]struct{}, len(existing))
	for _, t := range existing {
		seen[t] = struct{}{}
	}

	merged := make([]string, len(existing))
	copy(merged, existing)

	for _, t := range add {
		if _, ok := seen[t]; !ok {
			merged = append(merged, t)
			seen[t] = struct{}{}
		}
	}
	return merged
}

// truncate returns s truncated to maxLen bytes, with "..." appended if needed.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
