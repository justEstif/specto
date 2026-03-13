// Package youtube implements a SourcePlugin that imports watch history
// from a Google Takeout JSON export file.
package youtube

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"github.com/justestif/specto/internal/core"
)

// Compile-time interface check.
var _ core.SourcePlugin = (*Plugin)(nil)

// Plugin implements core.SourcePlugin for YouTube Takeout file imports.
type Plugin struct{}

// New returns a new YouTube plugin instance.
func New() *Plugin {
	return &Plugin{}
}

// takeoutEntry represents a single entry in the YouTube Takeout
// watch-history.json file.
type takeoutEntry struct {
	Header           string            `json:"header"`
	Title            string            `json:"title"`
	TitleURL         string            `json:"titleUrl"`
	Subtitles        []takeoutSubtitle `json:"subtitles"`
	Time             string            `json:"time"`
	Products         []string          `json:"products"`
	ActivityControls []string          `json:"activityControls"`
	Details          []takeoutDetail   `json:"details"`
}

// takeoutSubtitle represents a channel reference in a Takeout entry.
type takeoutSubtitle struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// takeoutDetail represents additional metadata in a Takeout entry.
type takeoutDetail struct {
	Name string `json:"name"`
}

const (
	deletedTitle = "Watched a video that has been removed"
	adDetail     = "From Google Ads"
	topicSuffix  = " - Topic"
	watchPrefix  = "Watched "
	musicHeader  = "YouTube Music"
)

// Name returns the unique identifier for this plugin.
func (p *Plugin) Name() string { return "youtube" }

// AuthType returns the authentication type for this plugin.
func (p *Plugin) AuthType() core.AuthType { return core.AuthFileImport }

// AuthConfig returns nil since this plugin uses file import, not OAuth.
func (p *Plugin) AuthConfig() *core.OAuthConfig { return nil }

// Sync parses a YouTube Takeout watch-history.json file from creds.File
// and returns the parsed media items. The cursor parameter is ignored
// because file imports always process the full file.
func (p *Plugin) Sync(_ context.Context, creds core.Credentials, _ string) core.SyncResult {
	var entries []takeoutEntry
	if err := json.NewDecoder(creds.File).Decode(&entries); err != nil {
		return core.SyncResult{
			Err: &core.PluginError{
				Code:    core.ErrFileParseError,
				Message: "failed to parse YouTube Takeout JSON: " + err.Error(),
				Raw:     err,
			},
		}
	}

	items := make([]core.MediaItem, 0, len(entries))
	for _, e := range entries {
		if shouldSkip(e) {
			continue
		}

		item := mapEntry(e)
		items = append(items, item)
	}

	return core.SyncResult{
		Items:   items,
		HasMore: false,
	}
}

// Enrich returns items unchanged. API-based enrichment is a separate task.
func (p *Plugin) Enrich(_ context.Context, _ core.Credentials, items []core.MediaItem) ([]core.MediaItem, error) {
	return items, nil
}

// shouldSkip returns true if the entry should be excluded from import.
func shouldSkip(e takeoutEntry) bool {
	// Skip deleted videos.
	if e.Title == deletedTitle {
		return true
	}

	// Skip entries without a URL (can't extract video ID).
	if e.TitleURL == "" {
		return true
	}

	// Skip ads.
	for _, d := range e.Details {
		if d.Name == adDetail {
			return true
		}
	}

	return false
}

// mapEntry converts a takeoutEntry into a core.MediaItem.
func mapEntry(e takeoutEntry) core.MediaItem {
	videoID := extractVideoID(e.TitleURL)

	title := strings.TrimPrefix(e.Title, watchPrefix)

	mediaType := core.MediaVideo
	if e.Header == musicHeader {
		mediaType = core.MediaMusic
	}

	creator := ""
	var channelURL, channelID string
	if len(e.Subtitles) > 0 {
		creator = e.Subtitles[0].Name
		channelURL = e.Subtitles[0].URL
		channelID = extractChannelID(channelURL)

		// Strip " - Topic" suffix for YouTube Music auto-generated channels.
		creator = strings.TrimSuffix(creator, topicSuffix)

		// A subtitle name that was just " - Topic" (with leading space)
		// becomes " " after TrimSuffix — treat as empty.
		creator = strings.TrimSpace(creator)
	}

	consumedAt, _ := time.Parse(time.RFC3339Nano, e.Time)

	rawMeta := map[string]any{
		"header":   e.Header,
		"products": e.Products,
	}
	if channelURL != "" {
		rawMeta["channel_url"] = channelURL
	}
	if channelID != "" {
		rawMeta["channel_id"] = channelID
	}

	return core.MediaItem{
		Platform:    "youtube",
		Type:        mediaType,
		Title:       title,
		Creator:     creator,
		ConsumedAt:  consumedAt,
		URL:         e.TitleURL,
		ExternalID:  videoID,
		RawMetadata: rawMeta,
	}
}

// extractVideoID parses a YouTube watch URL and returns the video ID
// from the "v" query parameter.
func extractVideoID(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Query().Get("v")
}

// extractChannelID parses a YouTube channel URL and returns the channel ID
// from the path segment after "/channel/".
func extractChannelID(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	// Path is like /channel/UC4JX40jDee_tINbkjycV4Sg
	parts := strings.Split(u.Path, "/")
	for i, part := range parts {
		if part == "channel" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}
