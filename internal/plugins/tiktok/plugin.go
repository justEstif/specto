// Package tiktok implements a SourcePlugin that parses TikTok GDPR/DSAR
// JSON data exports into normalized MediaItems.
package tiktok

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/justestif/specto/internal/core"
)

// Compile-time interface check.
var _ core.SourcePlugin = (*Plugin)(nil)

// Plugin parses TikTok GDPR JSON export files.
type Plugin struct{}

// New returns a new TikTok file-import plugin.
func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string                  { return "tiktok" }
func (p *Plugin) AuthType() core.AuthType       { return core.AuthFileImport }
func (p *Plugin) AuthConfig() *core.OAuthConfig { return nil }

// dateFormat is the timestamp format used in TikTok GDPR exports.
const dateFormat = "2006-01-02 15:04:05"

// videoIDPattern extracts the numeric video ID from a TikTok share URL.
var videoIDPattern = regexp.MustCompile(`/share/video/(\d+)/?`)

// export represents the top-level structure of a TikTok GDPR JSON export.
type export struct {
	Activity activity `json:"Activity"`
}

type activity struct {
	VideoBrowsingHistory videoBrowsingHistory `json:"Video Browsing History"`
	LikeList             likeList             `json:"Like List"`
	FavoriteVideos       favoriteVideos       `json:"Favorite Videos"`
}

type videoBrowsingHistory struct {
	VideoList []videoEntry `json:"VideoList"`
}

type likeList struct {
	ItemFavoriteList []videoEntry `json:"ItemFavoriteList"`
}

type favoriteVideos struct {
	FavoriteVideoList []videoEntry `json:"FavoriteVideoList"`
}

type videoEntry struct {
	Date      string `json:"Date"`
	VideoLink string `json:"VideoLink"`
}

// Sync reads the TikTok JSON export from creds.File and returns normalized MediaItems.
// The cursor parameter is ignored — file imports always process the full file.
func (p *Plugin) Sync(_ context.Context, creds core.Credentials, _ string) core.SyncResult {
	if creds.File == nil {
		return core.SyncResult{
			Err: &core.PluginError{
				Code:    core.ErrFileParseError,
				Message: "no file provided",
			},
		}
	}

	var data export
	if err := json.NewDecoder(creds.File).Decode(&data); err != nil {
		return core.SyncResult{
			Err: &core.PluginError{
				Code:    core.ErrFileParseError,
				Message: fmt.Sprintf("invalid JSON: %s", err.Error()),
				Raw:     err,
			},
		}
	}

	// Build lookup sets for liked and favorited video IDs.
	likedIDs := buildVideoIDSet(data.Activity.LikeList.ItemFavoriteList)
	favoritedIDs := buildVideoIDSet(data.Activity.FavoriteVideos.FavoriteVideoList)

	items := make([]core.MediaItem, 0, len(data.Activity.VideoBrowsingHistory.VideoList))
	for _, e := range data.Activity.VideoBrowsingHistory.VideoList {
		item, ok := mapEntry(e, likedIDs, favoritedIDs)
		if !ok {
			continue
		}
		items = append(items, item)
	}

	return core.SyncResult{Items: items}
}

// Enrich returns items unchanged — no platform-specific enrichment for file imports.
func (p *Plugin) Enrich(_ context.Context, _ core.Credentials, items []core.MediaItem) ([]core.MediaItem, error) {
	return items, nil
}

// mapEntry converts a single video entry to a MediaItem.
// It returns false if the entry should be skipped (missing link or unparseable date).
func mapEntry(e videoEntry, likedIDs, favoritedIDs map[string]bool) (core.MediaItem, bool) {
	if e.VideoLink == "" {
		return core.MediaItem{}, false
	}

	consumedAt, err := time.Parse(dateFormat, e.Date)
	if err != nil {
		return core.MediaItem{}, false
	}

	videoID := extractVideoID(e.VideoLink)
	if videoID == "" {
		return core.MediaItem{}, false
	}

	metadata := map[string]any{}
	if likedIDs[videoID] {
		metadata["liked"] = true
	}
	if favoritedIDs[videoID] {
		metadata["favorited"] = true
	}

	return core.MediaItem{
		Platform:    "tiktok",
		Type:        core.MediaVideo,
		ConsumedAt:  consumedAt,
		URL:         e.VideoLink,
		ExternalID:  videoID,
		RawMetadata: metadata,
	}, true
}

// extractVideoID returns the numeric video ID from a TikTok share URL,
// or an empty string if the URL doesn't match the expected pattern.
func extractVideoID(url string) string {
	matches := videoIDPattern.FindStringSubmatch(url)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

// buildVideoIDSet extracts video IDs from a list of entries and returns them as a set.
func buildVideoIDSet(entries []videoEntry) map[string]bool {
	ids := make(map[string]bool, len(entries))
	for _, e := range entries {
		if id := extractVideoID(e.VideoLink); id != "" {
			ids[id] = true
		}
	}
	return ids
}
