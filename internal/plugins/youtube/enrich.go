package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/justestif/specto/internal/core"
)

const (
	defaultAPIBaseURL = "https://www.googleapis.com/youtube/v3"
	maxBatchSize      = 50
)

// EnrichPlugin wraps Plugin and adds YouTube Data API v3 enrichment.
// It is used by NewWithBaseURL for testing and by the production plugin.
type EnrichPlugin struct {
	Plugin
	apiBaseURL string
	httpClient *http.Client
}

// Compile-time interface check.
var _ core.SourcePlugin = (*EnrichPlugin)(nil)

// NewWithEnrich returns a YouTube plugin with API enrichment enabled.
func NewWithEnrich() *EnrichPlugin {
	return &EnrichPlugin{
		apiBaseURL: defaultAPIBaseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// NewWithBaseURL returns a YouTube plugin pointing at the given API base URL.
// Intended for testing with httptest.Server.
func NewWithBaseURL(baseURL string) *EnrichPlugin {
	return &EnrichPlugin{
		apiBaseURL: baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// videosResponse is the top-level YouTube Data API response for
// GET /videos?part=snippet,contentDetails,statistics.
type videosResponse struct {
	Items []videoResource `json:"items"`
}

// videoResource is a single video resource returned by the YouTube API.
type videoResource struct {
	ID             string         `json:"id"`
	Snippet        videoSnippet   `json:"snippet"`
	ContentDetails contentDetails `json:"contentDetails"`
	Statistics     videoStats     `json:"statistics"`
}

// videoSnippet contains the snippet part of a video resource.
type videoSnippet struct {
	Title        string          `json:"title"`
	ChannelTitle string          `json:"channelTitle"`
	PublishedAt  string          `json:"publishedAt"`
	Tags         []string        `json:"tags"`
	CategoryID   string          `json:"categoryId"`
	Description  string          `json:"description"`
	Thumbnails   videoThumbnails `json:"thumbnails"`
}

// videoThumbnails contains thumbnail URLs at various sizes.
type videoThumbnails struct {
	Medium *thumbnail `json:"medium"`
	High   *thumbnail `json:"high"`
}

// thumbnail is a single thumbnail URL and size.
type thumbnail struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// contentDetails contains the content details part of a video resource.
type contentDetails struct {
	Duration string `json:"duration"` // ISO 8601 duration, e.g. "PT12M34S"
}

// videoStats contains the statistics part of a video resource.
type videoStats struct {
	ViewCount    string `json:"viewCount"`
	LikeCount    string `json:"likeCount"`
	CommentCount string `json:"commentCount"`
}

// Enrich calls the YouTube Data API to fill in metadata for imported videos.
// Items without an ExternalID (video ID) are returned unchanged. Deleted or
// private videos that the API doesn't return are also left unchanged.
// Enrichment failure is non-fatal — on error the original items are returned.
func (p *EnrichPlugin) Enrich(ctx context.Context, creds core.Credentials, items []core.MediaItem) ([]core.MediaItem, error) {
	if creds.AccessToken == "" {
		return nil, &core.PluginError{
			Code:    core.ErrAuthExpired,
			Message: "no access token provided for YouTube enrichment",
		}
	}

	// Collect video IDs and their indices for lookup after API call.
	type idIndex struct {
		id  string
		idx int
	}
	var toEnrich []idIndex
	for i, item := range items {
		if item.ExternalID != "" && item.Platform == "youtube" {
			toEnrich = append(toEnrich, idIndex{id: item.ExternalID, idx: i})
		}
	}

	if len(toEnrich) == 0 {
		return items, nil
	}

	// Make a copy so we don't mutate the original slice.
	enriched := make([]core.MediaItem, len(items))
	copy(enriched, items)

	// Batch fetch in groups of 50.
	for batchStart := 0; batchStart < len(toEnrich); batchStart += maxBatchSize {
		batchEnd := batchStart + maxBatchSize
		if batchEnd > len(toEnrich) {
			batchEnd = len(toEnrich)
		}
		batch := toEnrich[batchStart:batchEnd]

		ids := make([]string, len(batch))
		for i, entry := range batch {
			ids[i] = entry.id
		}

		videos, err := p.fetchVideos(ctx, creds.AccessToken, ids)
		if err != nil {
			return nil, err
		}

		// Build lookup by video ID.
		videoMap := make(map[string]videoResource, len(videos))
		for _, v := range videos {
			videoMap[v.ID] = v
		}

		// Apply enrichment to matched items.
		for _, entry := range batch {
			v, ok := videoMap[entry.id]
			if !ok {
				// Video is deleted/private — API didn't return it. Leave unchanged.
				continue
			}
			enriched[entry.idx] = applyVideoMetadata(enriched[entry.idx], v)
		}
	}

	return enriched, nil
}

// fetchVideos calls GET /videos?part=snippet,contentDetails,statistics&id={ids}
// and returns the video resources.
func (p *EnrichPlugin) fetchVideos(ctx context.Context, accessToken string, ids []string) ([]videoResource, error) {
	url := p.apiBaseURL + "/videos?part=snippet,contentDetails,statistics&id=" + strings.Join(ids, ",")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("building request: %s", err.Error()),
			Raw:     err,
		}
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("calling YouTube API: %s", err.Error()),
			Raw:     err,
		}
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return nil, &core.PluginError{
			Code:    core.ErrAuthExpired,
			Message: "YouTube returned 401 — access token expired or revoked",
		}
	case resp.StatusCode == http.StatusForbidden:
		return nil, &core.PluginError{
			Code:    core.ErrPermissionDenied,
			Message: "YouTube returned 403 — insufficient scopes or quota exceeded",
		}
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, &core.PluginError{
			Code:    core.ErrRateLimit,
			Message: "YouTube rate limit exceeded",
			Retry:   true,
			After:   30 * time.Second,
		}
	case resp.StatusCode >= 500:
		body, _ := io.ReadAll(resp.Body)
		return nil, &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("YouTube returned %d: %s", resp.StatusCode, truncateStr(string(body), 200)),
		}
	case resp.StatusCode != http.StatusOK:
		body, _ := io.ReadAll(resp.Body)
		return nil, &core.PluginError{
			Code:    core.ErrUpstream,
			Message: fmt.Sprintf("unexpected status %d: %s", resp.StatusCode, truncateStr(string(body), 200)),
		}
	}

	var apiResp videosResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, &core.PluginError{
			Code:    core.ErrInvalidData,
			Message: fmt.Sprintf("decoding YouTube response: %s", err.Error()),
			Raw:     err,
		}
	}

	return apiResp.Items, nil
}

// applyVideoMetadata enriches a MediaItem with data from the YouTube API.
func applyVideoMetadata(item core.MediaItem, v videoResource) core.MediaItem {
	// Update title to canonical API version.
	if v.Snippet.Title != "" {
		item.Title = v.Snippet.Title
	}

	// Update creator to canonical channel title.
	if v.Snippet.ChannelTitle != "" {
		item.Creator = v.Snippet.ChannelTitle
	}

	// Parse and set duration.
	if d, err := parseISO8601Duration(v.ContentDetails.Duration); err == nil && d > 0 {
		item.Duration = &d
	}

	// Map snippet.tags to the fixed tag taxonomy.
	var validTags []string
	for _, tag := range v.Snippet.Tags {
		normalized := normalizeTags(tag)
		for _, nt := range normalized {
			if core.IsValidTag(nt) {
				validTags = append(validTags, nt)
			}
		}
	}

	// Map categoryId to a known tag.
	if catTag := categoryToTag(v.Snippet.CategoryID); catTag != "" {
		validTags = append(validTags, catTag)
	}

	// Merge with existing tags, deduplicating.
	if len(validTags) > 0 {
		seen := make(map[string]bool, len(item.Tags)+len(validTags))
		for _, t := range item.Tags {
			seen[t] = true
		}
		for _, t := range validTags {
			if !seen[t] {
				item.Tags = append(item.Tags, t)
				seen[t] = true
			}
		}
	}

	// Ensure RawMetadata map exists.
	if item.RawMetadata == nil {
		item.RawMetadata = make(map[string]any)
	}

	// Add enrichment metadata.
	if v.Statistics.ViewCount != "" {
		if count, err := strconv.ParseInt(v.Statistics.ViewCount, 10, 64); err == nil {
			item.RawMetadata["view_count"] = count
		}
	}
	if v.Statistics.LikeCount != "" {
		if count, err := strconv.ParseInt(v.Statistics.LikeCount, 10, 64); err == nil {
			item.RawMetadata["like_count"] = count
		}
	}
	if v.Snippet.PublishedAt != "" {
		item.RawMetadata["published_at"] = v.Snippet.PublishedAt
	}
	if v.Snippet.CategoryID != "" {
		item.RawMetadata["category_id"] = v.Snippet.CategoryID
		if name := categoryName(v.Snippet.CategoryID); name != "" {
			item.RawMetadata["category_name"] = name
		}
	}
	if v.Snippet.Description != "" {
		item.RawMetadata["description"] = truncateStr(v.Snippet.Description, 500)
	}

	// Prefer medium thumbnail, fall back to high.
	if v.Snippet.Thumbnails.Medium != nil && v.Snippet.Thumbnails.Medium.URL != "" {
		item.RawMetadata["thumbnail_url"] = v.Snippet.Thumbnails.Medium.URL
	} else if v.Snippet.Thumbnails.High != nil && v.Snippet.Thumbnails.High.URL != "" {
		item.RawMetadata["thumbnail_url"] = v.Snippet.Thumbnails.High.URL
	}

	return item
}

// iso8601DurationRegex matches ISO 8601 durations like PT1H2M3S, PT5M, PT30S, etc.
var iso8601DurationRegex = regexp.MustCompile(`^PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?$`)

// parseISO8601Duration parses an ISO 8601 duration string (e.g. "PT12M34S")
// into a time.Duration. Returns 0 for empty or unparseable strings.
func parseISO8601Duration(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	matches := iso8601DurationRegex.FindStringSubmatch(s)
	if matches == nil {
		return 0, fmt.Errorf("invalid ISO 8601 duration: %s", s)
	}

	var d time.Duration

	if matches[1] != "" {
		h, _ := strconv.Atoi(matches[1])
		d += time.Duration(h) * time.Hour
	}
	if matches[2] != "" {
		m, _ := strconv.Atoi(matches[2])
		d += time.Duration(m) * time.Minute
	}
	if matches[3] != "" {
		sec, _ := strconv.Atoi(matches[3])
		d += time.Duration(sec) * time.Second
	}

	return d, nil
}

// normalizeTags converts a YouTube tag string into lowercase, hyphenated
// candidates for matching against the fixed tag taxonomy.
func normalizeTags(tag string) []string {
	lower := strings.ToLower(strings.TrimSpace(tag))
	if lower == "" {
		return nil
	}

	// Replace spaces and underscores with hyphens.
	normalized := strings.NewReplacer(
		" ", "-",
		"_", "-",
	).Replace(lower)

	// Return both the original lowercased and the hyphenated form.
	results := []string{normalized}
	if lower != normalized {
		results = append(results, lower)
	}
	return results
}

// YouTube video category IDs mapped to their standard names.
// These are the standard categories used globally by YouTube.
// https://developers.google.com/youtube/v3/docs/videoCategories/list
var categoryNames = map[string]string{
	"1":  "Film & Animation",
	"2":  "Autos & Vehicles",
	"10": "Music",
	"15": "Pets & Animals",
	"17": "Sports",
	"18": "Short Movies",
	"19": "Travel & Events",
	"20": "Gaming",
	"21": "Videoblogging",
	"22": "People & Blogs",
	"23": "Comedy",
	"24": "Entertainment",
	"25": "News & Politics",
	"26": "Howto & Style",
	"27": "Education",
	"28": "Science & Technology",
	"29": "Nonprofits & Activism",
	"30": "Movies",
	"31": "Anime/Animation",
	"32": "Action/Adventure",
	"33": "Classics",
	"34": "Comedy",
	"35": "Documentary",
	"36": "Drama",
	"37": "Family",
	"38": "Foreign",
	"39": "Horror",
	"40": "Sci-Fi/Fantasy",
	"41": "Thriller",
	"42": "Shorts",
	"43": "Shows",
	"44": "Trailers",
}

// categoryToTag maps a YouTube category ID to a tag from the fixed taxonomy.
// Returns empty string if no mapping exists.
func categoryToTag(categoryID string) string {
	tagMap := map[string]string{
		"1":  "animation",   // Film & Animation
		"2":  "automotive",  // Autos & Vehicles
		"10": "",            // Music — too generic, skip
		"15": "nature",      // Pets & Animals
		"17": "sports",      // Sports
		"19": "travel",      // Travel & Events
		"20": "gaming",      // Gaming
		"23": "comedy",      // Comedy
		"24": "",            // Entertainment — too generic, skip
		"25": "politics",    // News & Politics
		"26": "",            // Howto & Style — no direct match
		"27": "education",   // Education
		"28": "technology",  // Science & Technology
		"30": "film",        // Movies
		"31": "animation",   // Anime/Animation
		"32": "action",      // Action/Adventure
		"35": "documentary", // Documentary
		"36": "drama",       // Drama
		"39": "horror",      // Horror
		"40": "sci-fi",      // Sci-Fi/Fantasy
		"41": "thriller",    // Thriller
		"44": "trailer",     // Trailers
	}

	tag := tagMap[categoryID]
	if tag != "" && core.IsValidTag(tag) {
		return tag
	}
	return ""
}

// categoryName returns the human-readable name for a YouTube category ID.
func categoryName(categoryID string) string {
	return categoryNames[categoryID]
}

// truncateStr returns s truncated to maxLen characters, with "..." appended if needed.
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
