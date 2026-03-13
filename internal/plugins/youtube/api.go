package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/justestif/specto/internal/core"
)

// Compile-time interface check.
var _ core.SourcePlugin = (*APIPlugin)(nil)

// APIPlugin syncs YouTube watch history via the YouTube Data API v3.
// Unlike the file-import Plugin, this uses OAuth to access the user's
// YouTube activity via the API.
type APIPlugin struct {
	EnrichPlugin // embeds file-import + enrichment capabilities
}

// NewAPI returns a YouTube API plugin that calls the production YouTube API.
func NewAPI() *APIPlugin {
	return &APIPlugin{
		EnrichPlugin: EnrichPlugin{
			apiBaseURL: defaultAPIBaseURL,
			httpClient: &http.Client{Timeout: 30 * time.Second},
		},
	}
}

// NewAPIWithBaseURL returns a YouTube API plugin pointing at the given base URL.
// Intended for testing with httptest.Server.
func NewAPIWithBaseURL(baseURL string) *APIPlugin {
	return &APIPlugin{
		EnrichPlugin: EnrichPlugin{
			apiBaseURL: baseURL,
			httpClient: &http.Client{Timeout: 30 * time.Second},
		},
	}
}

func (p *APIPlugin) Name() string            { return "youtube-api" }
func (p *APIPlugin) AuthType() core.AuthType { return core.AuthOAuth }

func (p *APIPlugin) AuthConfig() *core.OAuthConfig {
	return &core.OAuthConfig{
		ProviderName: "YouTube",
		AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		Scopes:       []string{"https://www.googleapis.com/auth/youtube.readonly"},
	}
}

// activityResponse is the top-level YouTube Data API response for
// GET /activities?part=snippet,contentDetails.
type activityResponse struct {
	Items         []activityItem `json:"items"`
	NextPageToken string         `json:"nextPageToken"`
	PageInfo      struct {
		TotalResults   int `json:"totalResults"`
		ResultsPerPage int `json:"resultsPerPage"`
	} `json:"pageInfo"`
}

// activityItem is a single activity resource.
type activityItem struct {
	ID      string          `json:"id"`
	Snippet activitySnippet `json:"snippet"`
}

// activitySnippet contains the snippet part of an activity resource.
type activitySnippet struct {
	PublishedAt  string `json:"publishedAt"`
	Title        string `json:"title"`
	ChannelTitle string `json:"channelTitle"`
	Type         string `json:"type"` // "upload", "playlistItem", etc.
	Thumbnails   struct {
		Medium *thumbnail `json:"medium"`
	} `json:"thumbnails"`
}

// Sync fetches the user's YouTube activity (uploads, likes, etc.) via the API.
// The cursor is a page token for pagination.
func (p *APIPlugin) Sync(ctx context.Context, creds core.Credentials, cursor string) core.SyncResult {
	if creds.AccessToken == "" {
		return core.SyncResult{
			Err: &core.PluginError{
				Code:    core.ErrAuthExpired,
				Message: "no access token provided",
			},
		}
	}

	// Use the activities endpoint to get user's YouTube activity
	url := p.apiBaseURL + "/activities?part=snippet,contentDetails&mine=true&maxResults=50"
	if cursor != "" {
		url += "&pageToken=" + cursor
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
				Message: fmt.Sprintf("calling YouTube API: %s", err.Error()),
				Raw:     err,
			},
		}
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return core.SyncResult{
			Err: &core.PluginError{
				Code:    core.ErrAuthExpired,
				Message: "YouTube returned 401 — access token expired or revoked",
			},
		}
	case resp.StatusCode == http.StatusForbidden:
		return core.SyncResult{
			Err: &core.PluginError{
				Code:    core.ErrPermissionDenied,
				Message: "YouTube returned 403 — insufficient scopes or quota exceeded",
			},
		}
	case resp.StatusCode == http.StatusTooManyRequests:
		return core.SyncResult{
			Err: &core.PluginError{
				Code:    core.ErrRateLimit,
				Message: "YouTube rate limit exceeded",
				Retry:   true,
				After:   30 * time.Second,
			},
		}
	case resp.StatusCode >= 500:
		body, _ := io.ReadAll(resp.Body)
		return core.SyncResult{
			Err: &core.PluginError{
				Code:    core.ErrUpstream,
				Message: fmt.Sprintf("YouTube returned %d: %s", resp.StatusCode, truncateStr(string(body), 200)),
			},
		}
	case resp.StatusCode != http.StatusOK:
		body, _ := io.ReadAll(resp.Body)
		return core.SyncResult{
			Err: &core.PluginError{
				Code:    core.ErrUpstream,
				Message: fmt.Sprintf("unexpected status %d: %s", resp.StatusCode, truncateStr(string(body), 200)),
			},
		}
	}

	var apiResp activityResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return core.SyncResult{
			Err: &core.PluginError{
				Code:    core.ErrInvalidData,
				Message: fmt.Sprintf("decoding YouTube response: %s", err.Error()),
				Raw:     err,
			},
		}
	}

	items := make([]core.MediaItem, 0, len(apiResp.Items))
	for _, activity := range apiResp.Items {
		item := mapActivityEntry(activity)
		items = append(items, item)
	}

	return core.SyncResult{
		Items:      items,
		NextCursor: apiResp.NextPageToken,
		HasMore:    apiResp.NextPageToken != "",
	}
}

// mapActivityEntry converts a YouTube activity to a core.MediaItem.
func mapActivityEntry(a activityItem) core.MediaItem {
	consumedAt, _ := time.Parse(time.RFC3339Nano, a.Snippet.PublishedAt)
	if consumedAt.IsZero() {
		consumedAt, _ = time.Parse(time.RFC3339, a.Snippet.PublishedAt)
	}

	mediaType := core.MediaVideo
	rawMeta := map[string]any{
		"activity_type": a.Snippet.Type,
	}
	if a.Snippet.Thumbnails.Medium != nil {
		rawMeta["thumbnail_url"] = a.Snippet.Thumbnails.Medium.URL
	}

	return core.MediaItem{
		Platform:    "youtube",
		Type:        mediaType,
		Title:       a.Snippet.Title,
		Creator:     a.Snippet.ChannelTitle,
		ConsumedAt:  consumedAt,
		ExternalID:  a.ID,
		RawMetadata: rawMeta,
	}
}
