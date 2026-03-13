// Package spotify also implements a SourcePlugin for Spotify's Web API
// (recently-played endpoint) using OAuth authentication.
package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/justestif/specto/internal/core"
)

const (
	defaultBaseURL = "https://api.spotify.com/v1"
)

// Compile-time interface check.
var _ core.SourcePlugin = (*APIPlugin)(nil)

// APIPlugin syncs recently-played tracks from Spotify's Web API.
type APIPlugin struct {
	baseURL    string
	httpClient *http.Client
}

// NewAPI returns a Spotify API plugin that calls the production Spotify API.
func NewAPI() *APIPlugin {
	return &APIPlugin{
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// NewAPIWithBaseURL returns a Spotify API plugin pointing at the given base URL.
// Intended for testing with httptest.Server.
func NewAPIWithBaseURL(baseURL string) *APIPlugin {
	return &APIPlugin{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *APIPlugin) Name() string            { return "spotify-api" }
func (p *APIPlugin) AuthType() core.AuthType { return core.AuthOAuth }

func (p *APIPlugin) AuthConfig() *core.OAuthConfig {
	return &core.OAuthConfig{
		ProviderName: "Spotify",
		AuthURL:      "https://accounts.spotify.com/authorize",
		TokenURL:     "https://accounts.spotify.com/api/token",
		Scopes:       []string{"user-read-recently-played"},
	}
}

// recentlyPlayedResponse is the top-level Spotify API response for
// GET /me/player/recently-played.
type recentlyPlayedResponse struct {
	Items   []playHistoryObject `json:"items"`
	Cursors *struct {
		After  string `json:"after"`
		Before string `json:"before"`
	} `json:"cursors"`
	Limit int    `json:"limit"`
	Next  string `json:"next"`
}

// playHistoryObject is a single entry in the recently-played response.
type playHistoryObject struct {
	Track    apiTrack  `json:"track"`
	PlayedAt string    `json:"played_at"`
	Context  *struct { // nullable
		Type string `json:"type"`
		URI  string `json:"uri"`
	} `json:"context"`
}

// apiTrack is a simplified Spotify track object.
type apiTrack struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	DurationMs   int64  `json:"duration_ms"`
	Popularity   int    `json:"popularity"`
	Explicit     bool   `json:"explicit"`
	URI          string `json:"uri"`
	ExternalURLs struct {
		Spotify string `json:"spotify"`
	} `json:"external_urls"`
	Artists []apiArtist `json:"artists"`
	Album   apiAlbum    `json:"album"`
}

// apiArtist is a simplified Spotify artist object.
type apiArtist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// apiAlbum is a simplified Spotify album object.
type apiAlbum struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Sync fetches recently-played tracks from the Spotify API.
// The cursor is a Unix millisecond timestamp; items played after that
// timestamp are returned.
func (p *APIPlugin) Sync(ctx context.Context, creds core.Credentials, cursor string) core.SyncResult {
	if creds.AccessToken == "" {
		return core.SyncResult{
			Err: &core.PluginError{
				Code:    core.ErrAuthExpired,
				Message: "no access token provided",
			},
		}
	}

	// Build request URL
	url := p.baseURL + "/me/player/recently-played?limit=50"
	if cursor != "" {
		url += "&after=" + cursor
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return core.SyncResult{
			Err: &core.PluginError{
				Code:    core.ErrUpstream,
				Message: fmt.Sprintf("building request: %s", err.Error()),
				Raw:     err,
			},
		}
	}
	req.Header.Set("Authorization", "Bearer "+creds.AccessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return core.SyncResult{
			Err: &core.PluginError{
				Code:    core.ErrUpstream,
				Message: fmt.Sprintf("calling Spotify API: %s", err.Error()),
				Raw:     err,
			},
		}
	}
	defer resp.Body.Close()

	// Handle error status codes before reading body
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return core.SyncResult{
			Err: &core.PluginError{
				Code:    core.ErrAuthExpired,
				Message: "Spotify returned 401 — access token expired or revoked",
			},
		}
	case resp.StatusCode == http.StatusForbidden:
		return core.SyncResult{
			Err: &core.PluginError{
				Code:    core.ErrPermissionDenied,
				Message: "Spotify returned 403 — insufficient scopes or app not approved",
			},
		}
	case resp.StatusCode == http.StatusTooManyRequests:
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		return core.SyncResult{
			Err: &core.PluginError{
				Code:    core.ErrRateLimit,
				Message: "Spotify rate limit exceeded",
				Retry:   true,
				After:   retryAfter,
			},
		}
	case resp.StatusCode >= 500:
		body, _ := io.ReadAll(resp.Body)
		return core.SyncResult{
			Err: &core.PluginError{
				Code:    core.ErrUpstream,
				Message: fmt.Sprintf("Spotify returned %d: %s", resp.StatusCode, truncate(string(body), 200)),
			},
		}
	case resp.StatusCode != http.StatusOK:
		body, _ := io.ReadAll(resp.Body)
		return core.SyncResult{
			Err: &core.PluginError{
				Code:    core.ErrUpstream,
				Message: fmt.Sprintf("unexpected status %d: %s", resp.StatusCode, truncate(string(body), 200)),
			},
		}
	}

	// Parse the response
	var apiResp recentlyPlayedResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return core.SyncResult{
			Err: &core.PluginError{
				Code:    core.ErrInvalidData,
				Message: fmt.Sprintf("decoding Spotify response: %s", err.Error()),
				Raw:     err,
			},
		}
	}

	items := make([]core.MediaItem, 0, len(apiResp.Items))
	var latestPlayedAtMs int64

	for _, entry := range apiResp.Items {
		item := mapAPIEntry(entry)
		items = append(items, item)

		// Track the latest played_at timestamp for the cursor
		if playedAtMs := toUnixMs(entry.PlayedAt); playedAtMs > latestPlayedAtMs {
			latestPlayedAtMs = playedAtMs
		}
	}

	// Build next cursor from the latest played_at timestamp.
	// The Spotify API also returns cursors.after, but using our own
	// ensures consistency even if the API changes its cursor format.
	var nextCursor string
	if latestPlayedAtMs > 0 {
		nextCursor = strconv.FormatInt(latestPlayedAtMs, 10)
	}

	return core.SyncResult{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    apiResp.Next != "",
	}
}

// Enrich returns items unchanged — platform-specific enrichment is not
// implemented for the API plugin yet.
func (p *APIPlugin) Enrich(_ context.Context, _ core.Credentials, items []core.MediaItem) ([]core.MediaItem, error) {
	return items, nil
}

// mapAPIEntry converts a Spotify play history object to a core.MediaItem.
func mapAPIEntry(entry playHistoryObject) core.MediaItem {
	track := entry.Track

	consumedAt, _ := time.Parse(time.RFC3339Nano, entry.PlayedAt)
	// Spotify always provides played_at in RFC3339, but fall back to
	// the simpler layout if nano fails.
	if consumedAt.IsZero() {
		consumedAt, _ = time.Parse(time.RFC3339, entry.PlayedAt)
	}

	duration := time.Duration(track.DurationMs) * time.Millisecond

	creator := ""
	var artistIDs []string
	if len(track.Artists) > 0 {
		creator = track.Artists[0].Name
		for _, a := range track.Artists {
			artistIDs = append(artistIDs, a.ID)
		}
	}

	rawMeta := map[string]any{
		"popularity": track.Popularity,
		"explicit":   track.Explicit,
	}
	if track.Album.ID != "" {
		rawMeta["album_id"] = track.Album.ID
		rawMeta["album"] = track.Album.Name
	}
	if len(artistIDs) > 0 {
		rawMeta["artist_ids"] = artistIDs
	}
	if entry.Context != nil {
		rawMeta["context_type"] = entry.Context.Type
		rawMeta["context_uri"] = entry.Context.URI
	}

	url := track.ExternalURLs.Spotify
	if url == "" && track.ID != "" {
		url = "https://open.spotify.com/track/" + track.ID
	}

	return core.MediaItem{
		Platform:    "spotify",
		Type:        core.MediaMusic,
		Title:       track.Name,
		Creator:     creator,
		ConsumedAt:  consumedAt,
		Duration:    &duration,
		ExternalID:  track.URI,
		URL:         url,
		RawMetadata: rawMeta,
	}
}

// toUnixMs parses an RFC3339 timestamp and returns Unix milliseconds, or 0 on error.
func toUnixMs(ts string) int64 {
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		t, err = time.Parse(time.RFC3339, ts)
		if err != nil {
			return 0
		}
	}
	return t.UnixMilli()
}

// parseRetryAfter parses a Retry-After header value (seconds) into a Duration.
// Defaults to 30 seconds if the header is missing or invalid.
func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return 30 * time.Second
	}
	secs, err := strconv.Atoi(header)
	if err != nil || secs <= 0 {
		return 30 * time.Second
	}
	return time.Duration(secs) * time.Second
}

// truncate returns s truncated to maxLen characters, with "..." appended if needed.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
