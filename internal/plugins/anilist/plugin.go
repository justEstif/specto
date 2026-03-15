package anilist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/justestif/specto/internal/core"
)

// Compile-time interface check.
var _ core.SourcePlugin = (*APIPlugin)(nil)

// APIPlugin syncs anime and manga lists from AniList's GraphQL API.
type APIPlugin struct {
	endpoint   string
	httpClient *http.Client
	limiter    *rateLimiter
}

// NewAPI returns an AniList API plugin that calls the production AniList API.
func NewAPI() *APIPlugin {
	return &APIPlugin{
		endpoint:   defaultEndpoint,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		limiter:    newRateLimiter(rateInterval),
	}
}

// NewAPIWithEndpoint returns an AniList API plugin pointing at the given
// GraphQL endpoint. Intended for testing with httptest.Server.
func NewAPIWithEndpoint(endpoint string) *APIPlugin {
	return &APIPlugin{
		endpoint:   endpoint,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		limiter:    newRateLimiter(0), // no rate limiting in tests
	}
}

func (p *APIPlugin) Name() string            { return "anilist-api" }
func (p *APIPlugin) AuthType() core.AuthType { return core.AuthOAuth }

func (p *APIPlugin) AuthConfig() *core.OAuthConfig {
	return &core.OAuthConfig{
		ProviderName: "AniList",
		AuthURL:      "https://anilist.co/api/v2/oauth/authorize",
		TokenURL:     "https://anilist.co/api/v2/oauth/token",
		Scopes:       nil, // AniList doesn't use scopes
	}
}

// ── GraphQL types for the Sync plugin ───────────────────────────────────

// viewerResponse holds the authenticated user's ID.
type viewerResponse struct {
	Data *struct {
		Viewer *struct {
			ID int `json:"id"`
		} `json:"Viewer"`
	} `json:"data"`
	Errors []graphqlError `json:"errors"`
}

// mediaListCollectionResponse holds the user's anime or manga list.
type mediaListCollectionResponse struct {
	Data *struct {
		MediaListCollection *struct {
			Lists []mediaListGroup `json:"lists"`
		} `json:"MediaListCollection"`
	} `json:"data"`
	Errors []graphqlError `json:"errors"`
}

type mediaListGroup struct {
	Entries []mediaListEntry `json:"entries"`
}

type mediaListEntry struct {
	ID              int        `json:"id"`
	MediaID         int        `json:"mediaId"`
	Status          string     `json:"status"`
	Score           float64    `json:"score"`
	Progress        int        `json:"progress"`
	ProgressVolumes int        `json:"progressVolumes"`
	UpdatedAt       int64      `json:"updatedAt"`
	CompletedAt     fuzzyDate  `json:"completedAt"`
	StartedAt       fuzzyDate  `json:"startedAt"`
	Media           mediaEntry `json:"media"`
}

type fuzzyDate struct {
	Year  *int `json:"year"`
	Month *int `json:"month"`
	Day   *int `json:"day"`
}

// isSet returns true if the fuzzy date has at least a year.
func (d fuzzyDate) isSet() bool {
	return d.Year != nil && *d.Year > 0
}

// toTime converts the fuzzy date to a time.Time.
// Missing month/day default to 1.
func (d fuzzyDate) toTime() time.Time {
	if !d.isSet() {
		return time.Time{}
	}
	month := 1
	if d.Month != nil && *d.Month > 0 {
		month = *d.Month
	}
	day := 1
	if d.Day != nil && *d.Day > 0 {
		day = *d.Day
	}
	return time.Date(*d.Year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

type mediaEntry struct {
	ID    int `json:"id"`
	Title struct {
		Romaji  string `json:"romaji"`
		English string `json:"english"`
		Native  string `json:"native"`
	} `json:"title"`
	Format       string   `json:"format"`
	Episodes     int      `json:"episodes"`
	Chapters     int      `json:"chapters"`
	Volumes      int      `json:"volumes"`
	Genres       []string `json:"genres"`
	AverageScore int      `json:"averageScore"`
	SiteURL      string   `json:"siteUrl"`
}

// ── GraphQL queries ─────────────────────────────────────────────────────

const viewerQuery = `query { Viewer { id } }`

const mediaListQuery = `query ($userId: Int, $type: MediaType, $updatedSince: Int) {
  MediaListCollection(userId: $userId, type: $type, sort: UPDATED_TIME, forceSingleCompletedList: true) {
    lists {
      entries {
        id
        mediaId
        status
        score(format: POINT_10)
        progress
        progressVolumes
        updatedAt
        completedAt { year month day }
        startedAt { year month day }
        media {
          id
          title { romaji english native }
          format
          episodes
          chapters
          volumes
          genres
          averageScore
          siteUrl
        }
      }
    }
  }
}`

// ── Sync ────────────────────────────────────────────────────────────────

// Sync fetches the authenticated user's anime and manga lists from AniList.
// The cursor is a Unix timestamp (seconds); entries updated after that
// timestamp are included on incremental syncs.
func (p *APIPlugin) Sync(ctx context.Context, creds core.Credentials, cursor string) core.SyncResult {
	if creds.AccessToken == "" {
		return core.SyncResult{
			Err: &core.PluginError{
				Code:    core.ErrAuthExpired,
				Message: "no access token provided",
			},
		}
	}

	// Step 1: Get the authenticated user's ID.
	userID, err := p.fetchViewerID(ctx, creds.AccessToken)
	if err != nil {
		return core.SyncResult{Err: toPluginError(err)}
	}

	// Parse cursor (Unix timestamp in seconds).
	var cursorTS int64
	if cursor != "" {
		cursorTS, _ = strconv.ParseInt(cursor, 10, 64)
	}

	// Step 2: Fetch anime and manga lists.
	animeItems, animeLatest, err := p.fetchMediaList(ctx, creds.AccessToken, userID, "ANIME", cursorTS)
	if err != nil {
		return core.SyncResult{Err: toPluginError(err)}
	}

	mangaItems, mangaLatest, err := p.fetchMediaList(ctx, creds.AccessToken, userID, "MANGA", cursorTS)
	if err != nil {
		return core.SyncResult{Err: toPluginError(err)}
	}

	// Combine results.
	items := make([]core.MediaItem, 0, len(animeItems)+len(mangaItems))
	items = append(items, animeItems...)
	items = append(items, mangaItems...)

	// Build next cursor from the latest updatedAt across both lists.
	latestTS := animeLatest
	if mangaLatest > latestTS {
		latestTS = mangaLatest
	}

	var nextCursor string
	if latestTS > 0 {
		nextCursor = strconv.FormatInt(latestTS, 10)
	}

	return core.SyncResult{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    false, // AniList returns full lists, not paginated
	}
}

// Enrich returns items unchanged — the existing AniList EnrichmentProvider
// handles enrichment separately via the enrichment pipeline.
func (p *APIPlugin) Enrich(_ context.Context, _ core.Credentials, items []core.MediaItem) ([]core.MediaItem, error) {
	return items, nil
}

// ── Internal helpers ────────────────────────────────────────────────────

// fetchViewerID calls the Viewer query to get the authenticated user's ID.
func (p *APIPlugin) fetchViewerID(ctx context.Context, accessToken string) (int, error) {
	body, err := p.doGraphQL(ctx, accessToken, viewerQuery, nil)
	if err != nil {
		return 0, err
	}

	var resp viewerResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, &core.PluginError{
			Code:    core.ErrInvalidData,
			Message: fmt.Sprintf("decoding AniList Viewer response: %s", err.Error()),
			Raw:     err,
		}
	}

	if len(resp.Errors) > 0 {
		return 0, gqlErrorToPluginError(resp.Errors[0])
	}

	if resp.Data == nil || resp.Data.Viewer == nil {
		return 0, &core.PluginError{
			Code:    core.ErrInvalidData,
			Message: "AniList Viewer response missing user data",
		}
	}

	return resp.Data.Viewer.ID, nil
}

// fetchMediaList fetches all entries for a given media type (ANIME or MANGA).
// Returns the mapped items and the latest updatedAt timestamp.
func (p *APIPlugin) fetchMediaList(ctx context.Context, accessToken string, userID int, mediaType string, cursorTS int64) ([]core.MediaItem, int64, error) {
	vars := map[string]any{
		"userId": userID,
		"type":   mediaType,
	}

	body, err := p.doGraphQL(ctx, accessToken, mediaListQuery, vars)
	if err != nil {
		return nil, 0, err
	}

	var resp mediaListCollectionResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, 0, &core.PluginError{
			Code:    core.ErrInvalidData,
			Message: fmt.Sprintf("decoding AniList MediaListCollection response: %s", err.Error()),
			Raw:     err,
		}
	}

	if len(resp.Errors) > 0 {
		return nil, 0, gqlErrorToPluginError(resp.Errors[0])
	}

	if resp.Data == nil || resp.Data.MediaListCollection == nil {
		return nil, 0, nil
	}

	var items []core.MediaItem
	var latestTS int64

	itemType := core.MediaVideo
	if mediaType == "MANGA" {
		itemType = core.MediaBook
	}

	for _, list := range resp.Data.MediaListCollection.Lists {
		for _, entry := range list.Entries {
			// Incremental sync: skip entries not updated since cursor.
			if cursorTS > 0 && entry.UpdatedAt <= cursorTS {
				continue
			}

			item := mapMediaListEntry(entry, itemType)
			items = append(items, item)

			if entry.UpdatedAt > latestTS {
				latestTS = entry.UpdatedAt
			}
		}
	}

	return items, latestTS, nil
}

// doGraphQL executes a GraphQL request against the AniList endpoint.
func (p *APIPlugin) doGraphQL(ctx context.Context, accessToken string, query string, variables map[string]any) ([]byte, error) {
	if err := p.limiter.wait(ctx); err != nil {
		return nil, &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("rate limiter: %s", err.Error()),
			Raw:     err,
		}
	}

	reqBody := graphqlRequest{
		Query:     query,
		Variables: variables,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("marshaling GraphQL request: %s", err.Error()),
			Raw:     err,
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("building request: %s", err.Error()),
			Raw:     err,
		}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

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

	// Handle HTTP-level errors.
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return nil, &core.PluginError{
			Code:    core.ErrAuthExpired,
			Message: "AniList returned 401 — access token expired or revoked",
		}
	case resp.StatusCode == http.StatusTooManyRequests:
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		return nil, &core.PluginError{
			Code:    core.ErrRateLimit,
			Message: "AniList rate limit exceeded",
			Retry:   true,
			After:   retryAfter,
		}
	case resp.StatusCode >= 500:
		body, _ := io.ReadAll(resp.Body)
		return nil, &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("AniList returned %d: %s", resp.StatusCode, truncate(string(body), 200)),
			Retry:   true,
		}
	case resp.StatusCode != http.StatusOK:
		body, _ := io.ReadAll(resp.Body)
		return nil, &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("AniList returned unexpected status %d: %s", resp.StatusCode, truncate(string(body), 200)),
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("reading AniList response: %s", err.Error()),
			Raw:     err,
		}
	}

	return body, nil
}

// mapMediaListEntry converts an AniList media list entry to a core.MediaItem.
func mapMediaListEntry(entry mediaListEntry, itemType core.MediaType) core.MediaItem {
	title := preferredTitle(entry.Media.Title.English, entry.Media.Title.Romaji, entry.Media.Title.Native)

	// Determine consumed time: prefer completedAt, fall back to updatedAt.
	var consumedAt time.Time
	if entry.CompletedAt.isSet() {
		consumedAt = entry.CompletedAt.toTime()
	} else if entry.UpdatedAt > 0 {
		consumedAt = time.Unix(entry.UpdatedAt, 0).UTC()
	}

	rawMeta := map[string]any{
		"status":   entry.Status,
		"score":    entry.Score,
		"progress": entry.Progress,
		"format":   entry.Media.Format,
	}
	if entry.Media.Episodes > 0 {
		rawMeta["episodes"] = entry.Media.Episodes
	}
	if entry.Media.Chapters > 0 {
		rawMeta["chapters"] = entry.Media.Chapters
	}
	if entry.Media.Volumes > 0 {
		rawMeta["volumes"] = entry.Media.Volumes
	}
	if entry.ProgressVolumes > 0 {
		rawMeta["progress_volumes"] = entry.ProgressVolumes
	}
	if len(entry.Media.Genres) > 0 {
		rawMeta["genres"] = entry.Media.Genres
	}
	if entry.Media.AverageScore > 0 {
		rawMeta["average_score"] = entry.Media.AverageScore
	}
	if entry.StartedAt.isSet() {
		rawMeta["started_at"] = entry.StartedAt.toTime().Format(time.DateOnly)
	}

	return core.MediaItem{
		Platform:    "anilist",
		Type:        itemType,
		Title:       title,
		ExternalID:  strconv.Itoa(entry.Media.ID),
		ConsumedAt:  consumedAt,
		URL:         entry.Media.SiteURL,
		RawMetadata: rawMeta,
	}
}

// preferredTitle returns the first non-empty title from english, romaji, native.
func preferredTitle(english, romaji, native string) string {
	if english != "" {
		return english
	}
	if romaji != "" {
		return romaji
	}
	return native
}

// gqlErrorToPluginError converts a GraphQL error to a *core.PluginError.
func gqlErrorToPluginError(e graphqlError) *core.PluginError {
	switch {
	case e.Status == http.StatusUnauthorized:
		return &core.PluginError{
			Code:    core.ErrAuthExpired,
			Message: fmt.Sprintf("AniList GraphQL: %s", e.Message),
		}
	case e.Status == http.StatusTooManyRequests:
		return &core.PluginError{
			Code:    core.ErrRateLimit,
			Message: fmt.Sprintf("AniList GraphQL: %s", e.Message),
			Retry:   true,
			After:   60 * time.Second,
		}
	case e.Status >= 500:
		return &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("AniList GraphQL: %s", e.Message),
			Retry:   true,
		}
	default:
		return &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("AniList GraphQL: %s", e.Message),
		}
	}
}

// toPluginError ensures an error is wrapped as a *core.PluginError.
func toPluginError(err error) *core.PluginError {
	if pe, ok := err.(*core.PluginError); ok {
		return pe
	}
	return &core.PluginError{
		Code:    core.ErrUpstream,
		Message: err.Error(),
		Raw:     err,
	}
}

// parseRetryAfter parses a Retry-After header value (seconds) into a Duration.
// Defaults to 60 seconds if the header is missing or invalid.
func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return 60 * time.Second
	}
	secs, err := strconv.Atoi(header)
	if err != nil || secs <= 0 {
		return 60 * time.Second
	}
	return time.Duration(secs) * time.Second
}
