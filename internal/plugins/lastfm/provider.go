// Package lastfm implements an EnrichmentProvider that uses the Last.fm
// and MusicBrainz APIs to add genre tags to music items.
package lastfm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/justestif/specto/internal/core"
)

const (
	defaultLastfmBaseURL = "https://ws.audioscrobbler.com/2.0/"
	defaultMBBaseURL     = "https://musicbrainz.org/ws/2"
	mbUserAgent          = "Specto/1.0 (https://github.com/justestif/specto)"

	// Last.fm allows 5 requests per second.
	lastfmRateInterval = 200 * time.Millisecond

	// MusicBrainz allows 1 request per second (strict).
	mbRateInterval = time.Second

	// Minimum tag "count" from Last.fm to consider it relevant.
	minTagCount = 10
)

// Compile-time interface check.
var _ core.EnrichmentProvider = (*Provider)(nil)

// Provider enriches music items with genre tags from Last.fm and MusicBrainz.
type Provider struct {
	apiKey     string
	httpClient *http.Client
	logger     *slog.Logger

	// Rate limiters implemented as time.Ticker intervals.
	// Each API call waits on the corresponding ticker before proceeding.
	lastfmLimiter *rateLimiter
	mbLimiter     *rateLimiter

	// Configurable base URLs (for testing).
	lastfmBaseURL string
	mbBaseURL     string
}

// New creates a new Last.fm enrichment provider with the given API key.
func New(apiKey string) *Provider {
	return &Provider{
		apiKey:        apiKey,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		logger:        slog.Default(),
		lastfmLimiter: newRateLimiter(lastfmRateInterval),
		mbLimiter:     newRateLimiter(mbRateInterval),
		lastfmBaseURL: defaultLastfmBaseURL,
		mbBaseURL:     defaultMBBaseURL,
	}
}

// newForTest creates a provider with custom base URLs and no rate limiting.
func newForTest(apiKey, lastfmURL, mbURL string) *Provider {
	return &Provider{
		apiKey:        apiKey,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
		logger:        slog.Default(),
		lastfmLimiter: newRateLimiter(0), // no delay in tests
		mbLimiter:     newRateLimiter(0), // no delay in tests
		lastfmBaseURL: lastfmURL,
		mbBaseURL:     mbURL,
	}
}

// Name returns the provider identifier.
func (p *Provider) Name() string { return "lastfm" }

// Supports returns true for all music items regardless of platform.
func (p *Provider) Supports(mediaType string, _ string) bool {
	return mediaType == string(core.MediaMusic)
}

// Enrich adds genre tags to music items by querying Last.fm and MusicBrainz.
//
// Strategy:
//  1. Group items by artist name
//  2. Fetch artist-level tags once per unique artist (Last.fm)
//  3. Fetch per-track tags to supplement (Last.fm)
//  4. Optionally fetch MusicBrainz genres for additional coverage
//  5. Normalize all tags against the fixed tag set
//  6. Assign validated tags to items
func (p *Provider) Enrich(ctx context.Context, items []core.MediaItem) ([]core.MediaItem, error) {
	if len(items) == 0 {
		return items, nil
	}

	// Make a copy so we don't mutate the original slice.
	enriched := make([]core.MediaItem, len(items))
	copy(enriched, items)

	// Group items by normalized artist name for dedup.
	type artistGroup struct {
		artist  string // original artist name (first seen)
		indices []int  // indices into enriched slice
	}
	groups := make(map[string]*artistGroup) // keyed by lowercase artist
	var groupOrder []string                 // preserve order

	for i, item := range enriched {
		if item.Creator == "" {
			continue
		}
		key := strings.ToLower(item.Creator)
		if g, ok := groups[key]; ok {
			g.indices = append(g.indices, i)
		} else {
			groups[key] = &artistGroup{
				artist:  item.Creator,
				indices: []int{i},
			}
			groupOrder = append(groupOrder, key)
		}
	}

	// Phase 1: Fetch artist-level tags (one request per unique artist).
	artistTagCache := make(map[string][]string) // lowercase artist -> validated tags
	for _, key := range groupOrder {
		g := groups[key]
		tags, err := p.fetchArtistTags(ctx, g.artist)
		if err != nil {
			p.logger.Warn("failed to fetch artist tags",
				"artist", g.artist,
				"error", err,
			)
			// Non-fatal: continue with empty artist tags.
			tags = nil
		}
		artistTagCache[key] = tags
	}

	// Phase 2: Fetch per-track tags and merge with artist tags.
	for _, key := range groupOrder {
		g := groups[key]
		artistTags := artistTagCache[key]

		for _, idx := range g.indices {
			item := &enriched[idx]
			trackTags, err := p.fetchTrackTags(ctx, item.Creator, item.Title)
			if err != nil {
				p.logger.Warn("failed to fetch track tags",
					"artist", item.Creator,
					"title", item.Title,
					"error", err,
				)
				// Non-fatal: fall through to artist tags only.
				trackTags = nil
			}

			// Merge: track-specific tags first, then artist tags.
			allTags := mergeUnique(trackTags, artistTags)

			// Phase 3: MusicBrainz genres as supplemental data.
			mbTags, err := p.fetchMBGenres(ctx, item.Creator, item.Title)
			if err != nil {
				p.logger.Warn("failed to fetch MusicBrainz genres",
					"artist", item.Creator,
					"title", item.Title,
					"error", err,
				)
				// Non-fatal: continue without MB data.
			} else {
				allTags = mergeUnique(allTags, mbTags)
			}

			// Merge new tags with any existing tags on the item.
			if len(allTags) > 0 {
				item.Tags = mergeUnique(item.Tags, allTags)
			}
		}
	}

	return enriched, nil
}

// ── Last.fm API ─────────────────────────────────────────────────────────

// lastfmTagsResponse is the JSON shape returned by Last.fm tag endpoints.
type lastfmTagsResponse struct {
	TopTags struct {
		Tag []lastfmTag `json:"tag"`
	} `json:"toptags"`
	Error   int    `json:"error"`
	Message string `json:"message"`
}

type lastfmTag struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// fetchTrackTags calls track.getTopTags on Last.fm and returns validated tags.
func (p *Provider) fetchTrackTags(ctx context.Context, artist, track string) ([]string, error) {
	params := url.Values{
		"method":  {"track.getTopTags"},
		"artist":  {artist},
		"track":   {track},
		"api_key": {p.apiKey},
		"format":  {"json"},
	}
	return p.fetchLastfmTags(ctx, params)
}

// fetchArtistTags calls artist.getTopTags on Last.fm and returns validated tags.
func (p *Provider) fetchArtistTags(ctx context.Context, artist string) ([]string, error) {
	params := url.Values{
		"method":  {"artist.getTopTags"},
		"artist":  {artist},
		"api_key": {p.apiKey},
		"format":  {"json"},
	}
	return p.fetchLastfmTags(ctx, params)
}

// fetchLastfmTags is the shared implementation for Last.fm tag fetching.
func (p *Provider) fetchLastfmTags(ctx context.Context, params url.Values) ([]string, error) {
	if err := p.lastfmLimiter.wait(ctx); err != nil {
		return nil, err
	}

	reqURL := p.lastfmBaseURL + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("calling Last.fm API: %s", err.Error()),
			Retry:   true,
			Raw:     err,
		}
	}
	defer resp.Body.Close()

	if err := checkHTTPError(resp, "Last.fm"); err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("reading Last.fm response: %s", err.Error()),
			Raw:     err,
		}
	}

	var tagsResp lastfmTagsResponse
	if err := json.Unmarshal(body, &tagsResp); err != nil {
		return nil, &core.PluginError{
			Code:    core.ErrInvalidData,
			Message: fmt.Sprintf("decoding Last.fm response: %s", err.Error()),
			Raw:     err,
		}
	}

	// Last.fm returns error codes in the JSON body for some failures.
	if tagsResp.Error != 0 {
		// Error 6 = "Track not found" / "Artist not found" — not retryable.
		return nil, &core.PluginError{
			Code:    core.ErrInvalidData,
			Message: fmt.Sprintf("Last.fm error %d: %s", tagsResp.Error, tagsResp.Message),
		}
	}

	// Extract and validate tags.
	var validated []string
	for _, t := range tagsResp.TopTags.Tag {
		if t.Count < minTagCount {
			continue
		}
		normalized := normalizeTag(t.Name)
		if core.IsValidTag(normalized) {
			validated = append(validated, normalized)
		}
	}

	return validated, nil
}

// ── MusicBrainz API ─────────────────────────────────────────────────────

// mbSearchResponse is the JSON shape for MusicBrainz recording search.
type mbSearchResponse struct {
	Recordings []mbRecording `json:"recordings"`
}

type mbRecording struct {
	ID     string    `json:"id"`
	Title  string    `json:"title"`
	Genres []mbGenre `json:"genres"`
}

type mbGenre struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// mbRecordingResponse is the JSON shape for a MusicBrainz recording lookup
// with genres included.
type mbRecordingResponse struct {
	ID     string    `json:"id"`
	Title  string    `json:"title"`
	Genres []mbGenre `json:"genres"`
}

// fetchMBGenres searches MusicBrainz for a recording and returns validated
// genre tags. This makes up to 2 requests: one search + one lookup.
func (p *Provider) fetchMBGenres(ctx context.Context, artist, title string) ([]string, error) {
	// Step 1: Search for the recording.
	mbid, err := p.searchMBRecording(ctx, artist, title)
	if err != nil {
		return nil, err
	}
	if mbid == "" {
		return nil, nil // not found
	}

	// Step 2: Get genres for the recording.
	return p.getMBRecordingGenres(ctx, mbid)
}

// searchMBRecording searches for a recording on MusicBrainz and returns its MBID.
func (p *Provider) searchMBRecording(ctx context.Context, artist, title string) (string, error) {
	if err := p.mbLimiter.wait(ctx); err != nil {
		return "", err
	}

	query := fmt.Sprintf("recording:%s AND artist:%s", title, artist)
	reqURL := fmt.Sprintf("%s/recording/?query=%s&fmt=json&limit=1",
		p.mbBaseURL, url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("building MB request: %w", err)
	}
	req.Header.Set("User-Agent", mbUserAgent)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("calling MusicBrainz API: %s", err.Error()),
			Retry:   true,
			Raw:     err,
		}
	}
	defer resp.Body.Close()

	if err := checkHTTPError(resp, "MusicBrainz"); err != nil {
		return "", err
	}

	var searchResp mbSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return "", &core.PluginError{
			Code:    core.ErrInvalidData,
			Message: fmt.Sprintf("decoding MusicBrainz search response: %s", err.Error()),
			Raw:     err,
		}
	}

	if len(searchResp.Recordings) == 0 {
		return "", nil
	}

	return searchResp.Recordings[0].ID, nil
}

// getMBRecordingGenres fetches genres for a MusicBrainz recording by MBID.
func (p *Provider) getMBRecordingGenres(ctx context.Context, mbid string) ([]string, error) {
	if err := p.mbLimiter.wait(ctx); err != nil {
		return nil, err
	}

	reqURL := fmt.Sprintf("%s/recording/%s?inc=genres&fmt=json", p.mbBaseURL, mbid)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building MB recording request: %w", err)
	}
	req.Header.Set("User-Agent", mbUserAgent)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("calling MusicBrainz recording API: %s", err.Error()),
			Retry:   true,
			Raw:     err,
		}
	}
	defer resp.Body.Close()

	if err := checkHTTPError(resp, "MusicBrainz"); err != nil {
		return nil, err
	}

	var recResp mbRecordingResponse
	if err := json.NewDecoder(resp.Body).Decode(&recResp); err != nil {
		return nil, &core.PluginError{
			Code:    core.ErrInvalidData,
			Message: fmt.Sprintf("decoding MusicBrainz recording response: %s", err.Error()),
			Raw:     err,
		}
	}

	var validated []string
	for _, g := range recResp.Genres {
		normalized := normalizeTag(g.Name)
		if core.IsValidTag(normalized) {
			validated = append(validated, normalized)
		}
	}

	return validated, nil
}

// ── Shared helpers ──────────────────────────────────────────────────────

// checkHTTPError inspects the response status and returns a *core.PluginError
// for non-200 responses.
func checkHTTPError(resp *http.Response, source string) error {
	switch {
	case resp.StatusCode == http.StatusOK:
		return nil
	case resp.StatusCode == http.StatusTooManyRequests:
		return &core.PluginError{
			Code:    core.ErrRateLimit,
			Message: fmt.Sprintf("%s rate limit exceeded", source),
			Retry:   true,
			After:   30 * time.Second,
		}
	case resp.StatusCode >= 500:
		body, _ := io.ReadAll(resp.Body)
		return &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("%s returned %d: %s", source, resp.StatusCode, truncate(string(body), 200)),
			Retry:   true,
		}
	default:
		body, _ := io.ReadAll(resp.Body)
		return &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("%s returned unexpected status %d: %s", source, resp.StatusCode, truncate(string(body), 200)),
		}
	}
}

// tagAliases maps common freeform tag variations to their canonical form
// in the fixed tag set. Applied before the standard normalization.
var tagAliases = map[string]string{
	// Hip-hop variants
	"hip hop":  "hip-hop",
	"hiphop":   "hip-hop",
	"hip-hop":  "hip-hop",
	"rap":      "hip-hop",
	"trap":     "hip-hop",
	"boom bap": "hip-hop",

	// R&B variants
	"r&b":                  "r-and-b",
	"rnb":                  "r-and-b",
	"r and b":              "r-and-b",
	"rhythm and blues":     "r-and-b",
	"rhythm-and-blues":     "r-and-b",
	"contemporary r&b":     "r-and-b",
	"contemporary r and b": "r-and-b",

	// Sci-fi variants
	"sci fi": "sci-fi",
	"scifi":  "sci-fi",
	"sci-fi": "sci-fi",

	// Electronic variants
	"electronica":   "electronic",
	"edm":           "electronic",
	"electro":       "electronic",
	"techno":        "electronic",
	"house":         "electronic",
	"trance":        "electronic",
	"dubstep":       "electronic",
	"drum and bass": "electronic",

	// Rock sub-genres → rock
	"prog rock":        "rock",
	"progressive rock": "rock",
	"classic rock":     "rock",
	"hard rock":        "rock",
	"soft rock":        "rock",
	"art rock":         "rock",
	"psychedelic rock": "rock",
	"garage rock":      "rock",
	"stoner rock":      "rock",

	// Indie sub-genres → indie
	"indie rock": "indie",
	"indie pop":  "indie",
	"indie folk": "indie",
	"lo-fi":      "indie",

	// Metal sub-genres → metal
	"heavy metal":  "metal",
	"death metal":  "metal",
	"black metal":  "metal",
	"thrash metal": "metal",
	"doom metal":   "metal",
	"nu metal":     "metal",
	"power metal":  "metal",

	// Punk sub-genres → punk
	"post-punk": "punk",
	"pop punk":  "punk",
	"post punk": "punk",
	"hardcore":  "punk",

	// Alternative variants
	"alt rock": "alternative",
	"alt-rock": "alternative",
	"alt":      "alternative",
	"grunge":   "alternative",
	"new wave": "alternative",
	"shoegaze": "alternative",

	// Other genre mappings
	"bossa nova":    "jazz",
	"bebop":         "jazz",
	"smooth jazz":   "jazz",
	"neo soul":      "soul",
	"neo-soul":      "soul",
	"bluegrass":     "country",
	"americana":     "country",
	"cumbia":        "latin",
	"salsa":         "latin",
	"reggaeton":     "latin",
	"bachata":       "latin",
	"bossa-nova":    "jazz",
	"dub":           "reggae",
	"ska":           "reggae",
	"dancehall":     "reggae",
	"delta blues":   "blues",
	"chicago blues": "blues",
	"deep house":    "electronic",
	"downtempo":     "ambient",
	"chillout":      "chill",
	"chill-out":     "chill",
	"chillwave":     "chill",

	// Mood mappings
	"happy":       "uplifting",
	"sad":         "melancholic",
	"angry":       "aggressive",
	"relaxing":    "peaceful",
	"mellow":      "chill",
	"upbeat":      "energetic",
	"atmospheric": "ambient",
}

// normalizeTag converts a freeform tag string from Last.fm/MusicBrainz
// into a candidate for the fixed tag set.
//
// Steps:
//  1. Lowercase and trim
//  2. Check alias table for known mappings
//  3. Replace spaces with hyphens, strip non-alphanumeric chars (except hyphens)
//  4. Return the result (caller validates with core.IsValidTag)
func normalizeTag(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return ""
	}

	// Check alias table first (before replacing spaces).
	if mapped, ok := tagAliases[s]; ok {
		return mapped
	}

	// Standard normalization: spaces → hyphens.
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")

	// Remove ampersands and other special chars, but keep hyphens.
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	result := b.String()

	// Collapse multiple hyphens.
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	result = strings.Trim(result, "-")

	// Check alias table again after normalization.
	if mapped, ok := tagAliases[result]; ok {
		return mapped
	}

	return result
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

// rateLimiter enforces a minimum interval between API calls using a channel-based
// token bucket (capacity 1). Zero interval means no limiting (for tests).
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
