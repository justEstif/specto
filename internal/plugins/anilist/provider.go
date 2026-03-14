// Package anilist implements an EnrichmentProvider that uses the AniList
// public GraphQL API to add genre and format tags to anime and manga items.
// No API key is required.
package anilist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/justestif/specto/internal/core"
)

const (
	defaultEndpoint = "https://graphql.anilist.co"

	// AniList rate limit: 90 requests per minute ≈ 1 request per 667ms.
	rateInterval = 667 * time.Millisecond

	// Only use AniList tags with rank >= this threshold.
	minTagRank = 60
)

// Compile-time interface check.
var _ core.EnrichmentProvider = (*Provider)(nil)

// Provider enriches anime/manga items with genre and format tags from AniList.
type Provider struct {
	httpClient *http.Client
	logger     *slog.Logger
	limiter    *rateLimiter
	endpoint   string
}

// New creates a new AniList enrichment provider. No API key is needed.
func New() *Provider {
	return &Provider{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     slog.Default(),
		limiter:    newRateLimiter(rateInterval),
		endpoint:   defaultEndpoint,
	}
}

// newForTest creates a provider with a custom endpoint and no rate limiting.
func newForTest(endpoint string) *Provider {
	return &Provider{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     slog.Default(),
		limiter:    newRateLimiter(0),
		endpoint:   endpoint,
	}
}

// Name returns the provider identifier.
func (p *Provider) Name() string { return "anilist" }

// Supports returns true for video items from anime or manga platforms.
// Items from general video platforms (YouTube, Netflix, etc.) are skipped
// to avoid unnecessary API calls that could trigger rate limiting.
func (p *Provider) Supports(mediaType string, platform string) bool {
	if mediaType != string(core.MediaVideo) {
		return false
	}
	lower := strings.ToLower(platform)
	return animePlatforms[lower] || mangaPlatforms[lower]
}

// Enrich adds genre, mood, topic, and format tags to items by querying AniList.
//
// For each item:
//  1. Determine whether to search as ANIME or MANGA
//  2. Search AniList by title
//  3. If a match is found, extract genres and tags
//  4. Map to the fixed tag set
//  5. Add format tag based on AniList media format
//  6. Store AniList ID in RawMetadata
func (p *Provider) Enrich(ctx context.Context, items []core.MediaItem) ([]core.MediaItem, error) {
	if len(items) == 0 {
		return items, nil
	}

	// Make a copy so we don't mutate the original slice.
	enriched := make([]core.MediaItem, len(items))
	copy(enriched, items)

	for i := range enriched {
		item := &enriched[i]

		mediaType := detectMediaType(item)
		result, err := p.searchMedia(ctx, item.Title, mediaType)
		if err != nil {
			p.logger.Warn("anilist search failed",
				"title", item.Title,
				"error", err,
			)
			// Per-item failure: skip and continue.
			continue
		}
		if result == nil {
			// No match found — not an error.
			continue
		}

		// Map genres and tags to fixed tag set.
		tags := mapGenresToTags(result.Genres)
		tags = mergeUnique(tags, mapAniListTags(result.Tags))
		tags = mergeUnique(tags, formatTag(result.Format, mediaType))

		// Only keep valid tags.
		var validated []string
		for _, t := range tags {
			if core.IsValidTag(t) {
				validated = append(validated, t)
			}
		}

		if len(validated) > 0 {
			item.Tags = mergeUnique(item.Tags, validated)
		}

		// Store AniList metadata.
		if item.RawMetadata == nil {
			item.RawMetadata = make(map[string]any)
		}
		item.RawMetadata["anilist_id"] = result.ID
		if result.AverageScore > 0 {
			item.RawMetadata["anilist_score"] = result.AverageScore
		}
		if result.Episodes > 0 {
			item.RawMetadata["anilist_episodes"] = result.Episodes
		}
		if result.Duration > 0 {
			item.RawMetadata["anilist_episode_duration"] = result.Duration
		}
	}

	return enriched, nil
}

// ── AniList GraphQL API ─────────────────────────────────────────────────

// graphqlRequest is the shape of a GraphQL request body.
type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

// graphqlResponse is the top-level shape of an AniList GraphQL response.
type graphqlResponse struct {
	Data   *mediaData     `json:"data"`
	Errors []graphqlError `json:"errors"`
}

type graphqlError struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

type mediaData struct {
	Media *mediaResult `json:"Media"`
}

// mediaResult holds the fields we extract from an AniList Media query.
type mediaResult struct {
	ID           int          `json:"id"`
	Genres       []string     `json:"genres"`
	Tags         []anilistTag `json:"tags"`
	Format       string       `json:"format"`
	Episodes     int          `json:"episodes"`
	Duration     int          `json:"duration"` // minutes per episode
	AverageScore int          `json:"averageScore"`
}

type anilistTag struct {
	Name           string `json:"name"`
	Rank           int    `json:"rank"`
	IsMediaSpoiler bool   `json:"isMediaSpoiler"`
}

const searchQuery = `query ($search: String, $type: MediaType) {
  Media(search: $search, type: $type) {
    id
    genres
    tags {
      name
      rank
      isMediaSpoiler
    }
    format
    episodes
    duration
    averageScore
  }
}`

// searchMedia queries AniList for a media item by title and type.
// Returns nil if no match is found.
func (p *Provider) searchMedia(ctx context.Context, title string, mediaType string) (*mediaResult, error) {
	if title == "" {
		return nil, nil
	}

	if err := p.limiter.wait(ctx); err != nil {
		return nil, err
	}

	reqBody := graphqlRequest{
		Query: searchQuery,
		Variables: map[string]any{
			"search": title,
			"type":   mediaType,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling graphql request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("calling AniList API: %s", err.Error()),
			Retry:   true,
			Raw:     err,
		}
	}
	defer resp.Body.Close()

	if err := checkHTTPError(resp); err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("reading AniList response: %s", err.Error()),
			Raw:     err,
		}
	}

	var gqlResp graphqlResponse
	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return nil, &core.PluginError{
			Code:    core.ErrInvalidData,
			Message: fmt.Sprintf("decoding AniList response: %s", err.Error()),
			Raw:     err,
		}
	}

	// Check for GraphQL-level errors.
	if len(gqlResp.Errors) > 0 {
		e := gqlResp.Errors[0]
		// Status 404 means not found — not an error from our perspective.
		if e.Status == http.StatusNotFound {
			return nil, nil
		}
		return nil, &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("AniList GraphQL error: %s", e.Message),
			Retry:   e.Status >= 500,
		}
	}

	if gqlResp.Data == nil || gqlResp.Data.Media == nil {
		return nil, nil // no match
	}

	return gqlResp.Data.Media, nil
}

// ── Media type detection ────────────────────────────────────────────────

// animePlatforms are platforms known to serve anime content.
var animePlatforms = map[string]bool{
	"crunchyroll": true,
	"funimation":  true,
	"hidive":      true,
	"animelab":    true,
	"vrv":         true,
}

// mangaPlatforms are platforms known to serve manga content.
var mangaPlatforms = map[string]bool{
	"mangadex":      true,
	"mangaplus":     true,
	"manga-plus":    true,
	"comixology":    true,
	"webtoon":       true,
	"shonen-jump":   true,
	"viz":           true,
	"manga-kakalot": true,
	"mangakakalot":  true,
}

// detectMediaType determines whether to search for ANIME or MANGA.
// Checks RawMetadata hints, platform name, and defaults based on media type.
func detectMediaType(item *core.MediaItem) string {
	// Check RawMetadata for explicit type hints.
	if item.RawMetadata != nil {
		if mt, ok := item.RawMetadata["media_type"].(string); ok {
			lower := strings.ToLower(mt)
			if lower == "manga" {
				return "MANGA"
			}
			if lower == "anime" {
				return "ANIME"
			}
		}
		if ct, ok := item.RawMetadata["content_type"].(string); ok {
			lower := strings.ToLower(ct)
			if lower == "manga" {
				return "MANGA"
			}
		}
	}

	// Check platform name.
	platform := strings.ToLower(item.Platform)
	if mangaPlatforms[platform] {
		return "MANGA"
	}
	if animePlatforms[platform] {
		return "ANIME"
	}

	// Default: video items are most likely anime.
	return "ANIME"
}

// ── Genre / tag mapping ─────────────────────────────────────────────────

// genreMap maps AniList genre names (case-insensitive) to fixed tags.
var genreMap = map[string]string{
	"action":        "action",
	"adventure":     "adventure",
	"comedy":        "comedy",
	"drama":         "drama",
	"fantasy":       "fantasy",
	"horror":        "horror",
	"mystery":       "mystery",
	"romance":       "romance",
	"sci-fi":        "sci-fi",
	"thriller":      "thriller",
	"supernatural":  "fantasy",
	"slice of life": "drama",
	"sports":        "sports",
	"music":         "musical",
}

// mapGenresToTags converts AniList genres to fixed tags.
func mapGenresToTags(genres []string) []string {
	var tags []string
	for _, g := range genres {
		key := strings.ToLower(g)
		if mapped, ok := genreMap[key]; ok {
			tags = append(tags, mapped)
		}
	}
	return tags
}

// anilistTagMap maps specific AniList tag names to fixed tags.
var anilistTagMap = map[string]string{
	"psychological": "intense",
	"gore":          "dark",
	"mecha":         "technology",
	"isekai":        "fantasy",
	"time travel":   "sci-fi",
}

// mapAniListTags converts AniList tags to fixed tags.
// Filters out spoiler tags and tags with rank below the threshold.
func mapAniListTags(tags []anilistTag) []string {
	var result []string
	for _, t := range tags {
		if t.IsMediaSpoiler {
			continue
		}
		if t.Rank < minTagRank {
			continue
		}
		key := strings.ToLower(t.Name)
		if mapped, ok := anilistTagMap[key]; ok {
			result = append(result, mapped)
		}
	}
	return result
}

// formatTag returns the format tag based on AniList format and media type.
func formatTag(format string, mediaType string) []string {
	if mediaType == "MANGA" {
		return []string{"graphic-novel"}
	}

	upper := strings.ToUpper(format)
	switch upper {
	case "TV", "TV_SHORT":
		return []string{"series"}
	case "MOVIE":
		return []string{"film"}
	case "OVA", "ONA", "SPECIAL":
		return []string{"episode"}
	default:
		return nil
	}
}

// ── Shared helpers ──────────────────────────────────────────────────────

// checkHTTPError inspects the response status and returns a *core.PluginError
// for non-200 responses.
func checkHTTPError(resp *http.Response) error {
	switch {
	case resp.StatusCode == http.StatusOK:
		return nil
	case resp.StatusCode == http.StatusTooManyRequests:
		return &core.PluginError{
			Code:    core.ErrRateLimit,
			Message: "AniList rate limit exceeded",
			Retry:   true,
			After:   60 * time.Second,
		}
	case resp.StatusCode >= 500:
		body, _ := io.ReadAll(resp.Body)
		return &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("AniList returned %d: %s", resp.StatusCode, truncate(string(body), 200)),
			Retry:   true,
		}
	default:
		body, _ := io.ReadAll(resp.Body)
		return &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("AniList returned unexpected status %d: %s", resp.StatusCode, truncate(string(body), 200)),
		}
	}
}

// mergeUnique merges two string slices, deduplicating entries.
func mergeUnique(a, b []string) []string {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(a)+len(b))
	merged := make([]string, 0, len(a)+len(b))

	for _, s := range a {
		if _, ok := seen[s]; !ok {
			merged = append(merged, s)
			seen[s] = struct{}{}
		}
	}
	for _, s := range b {
		if _, ok := seen[s]; !ok {
			merged = append(merged, s)
			seen[s] = struct{}{}
		}
	}

	return merged
}

// truncate returns s truncated to maxLen, with "..." appended if needed.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ── Rate limiter ────────────────────────────────────────────────────────

// rateLimiter enforces a minimum interval between API calls using a
// channel-based token bucket (capacity 1). Zero interval means no limiting.
type rateLimiter struct {
	interval time.Duration
	tokens   chan struct{}
	stopCh   chan struct{}
}

// newRateLimiter creates a rate limiter that allows one call per interval.
// If interval is zero, no rate limiting is applied.
func newRateLimiter(interval time.Duration) *rateLimiter {
	rl := &rateLimiter{
		interval: interval,
	}

	if interval > 0 {
		rl.tokens = make(chan struct{}, 1)
		rl.stopCh = make(chan struct{})
		// Seed with one token so the first request goes through immediately.
		rl.tokens <- struct{}{}
		go rl.refill()
	}

	return rl
}

// refill continuously adds tokens at the configured interval.
func (rl *rateLimiter) refill() {
	ticker := time.NewTicker(rl.interval)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stopCh:
			return
		case <-ticker.C:
			select {
			case rl.tokens <- struct{}{}:
			default:
				// Token bucket is full (cap 1), discard.
			}
		}
	}
}

// wait blocks until a token is available or the context is cancelled.
func (rl *rateLimiter) wait(ctx context.Context) error {
	if rl.interval == 0 {
		return nil // no limiting
	}

	select {
	case <-rl.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
