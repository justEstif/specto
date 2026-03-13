// Package spotify implements a SourcePlugin that parses Spotify GDPR
// Extended Streaming History JSON exports into normalized MediaItems.
package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/justestif/specto/internal/core"
)

// Compile-time interface check.
var _ core.SourcePlugin = (*Plugin)(nil)

// Plugin parses Spotify GDPR Extended Streaming History JSON files.
type Plugin struct{}

// New returns a new Spotify file-import plugin.
func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string                  { return "spotify" }
func (p *Plugin) AuthType() core.AuthType       { return core.AuthFileImport }
func (p *Plugin) AuthConfig() *core.OAuthConfig { return nil }

// entry represents a single row in the Spotify Extended Streaming History JSON export.
// Nullable fields use pointer types.
type entry struct {
	TS               string  `json:"ts"`
	Username         string  `json:"username"`
	Platform         string  `json:"platform"`
	MsPlayed         int64   `json:"ms_played"`
	ConnCountry      string  `json:"conn_country"`
	IPAddr           *string `json:"ip_addr_decrypted"`
	UserAgent        *string `json:"user_agent_decrypted"`
	TrackName        *string `json:"master_metadata_track_name"`
	ArtistName       *string `json:"master_metadata_album_artist_name"`
	AlbumName        *string `json:"master_metadata_album_album_name"`
	TrackURI         *string `json:"spotify_track_uri"`
	EpisodeName      *string `json:"episode_name"`
	EpisodeShowName  *string `json:"episode_show_name"`
	EpisodeURI       *string `json:"spotify_episode_uri"`
	ReasonStart      *string `json:"reason_start"`
	ReasonEnd        *string `json:"reason_end"`
	Shuffle          *bool   `json:"shuffle"`
	Skipped          *bool   `json:"skipped"`
	Offline          *bool   `json:"offline"`
	OfflineTimestamp *int64  `json:"offline_timestamp"`
	IncognitoMode    *bool   `json:"incognito_mode"`
}

// Sync reads the Spotify JSON export from creds.File and returns normalized MediaItems.
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

	var entries []entry
	if err := json.NewDecoder(creds.File).Decode(&entries); err != nil {
		return core.SyncResult{
			Err: &core.PluginError{
				Code:    core.ErrFileParseError,
				Message: fmt.Sprintf("invalid JSON: %s", err.Error()),
				Raw:     err,
			},
		}
	}

	items := make([]core.MediaItem, 0, len(entries))
	for _, e := range entries {
		item, ok := mapEntry(e)
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

// mapEntry converts a single JSON entry to a MediaItem.
// It returns false if the entry should be skipped (unidentifiable local file).
func mapEntry(e entry) (core.MediaItem, bool) {
	isPodcast := e.EpisodeName != nil && *e.EpisodeName != ""

	if !isPodcast {
		// Music entry: skip if both track name and track URI are missing
		if (e.TrackName == nil || *e.TrackName == "") && (e.TrackURI == nil || *e.TrackURI == "") {
			return core.MediaItem{}, false
		}
	}

	consumedAt, err := time.Parse(time.RFC3339, e.TS)
	if err != nil {
		// Best-effort: if timestamp is unparseable, use zero time rather than skipping
		consumedAt = time.Time{}
	}

	timeSpent := time.Duration(e.MsPlayed) * time.Millisecond

	item := core.MediaItem{
		Platform:    "spotify",
		ConsumedAt:  consumedAt,
		TimeSpent:   &timeSpent,
		RawMetadata: buildRawMetadata(e),
	}

	if isPodcast {
		item.Type = core.MediaPodcast
		item.Title = *e.EpisodeName
		if e.EpisodeShowName != nil {
			item.Creator = *e.EpisodeShowName
		}
		if e.EpisodeURI != nil && *e.EpisodeURI != "" {
			item.ExternalID = *e.EpisodeURI
			// Extract episode ID from URI: "spotify:episode:XXXX" → "XXXX"
			if id := extractID(*e.EpisodeURI); id != "" {
				item.URL = "https://open.spotify.com/episode/" + id
			}
		}
	} else {
		item.Type = core.MediaMusic
		if e.TrackName != nil {
			item.Title = *e.TrackName
		}
		if e.ArtistName != nil {
			item.Creator = *e.ArtistName
		}
		if e.TrackURI != nil && *e.TrackURI != "" {
			item.ExternalID = *e.TrackURI
			if id := extractID(*e.TrackURI); id != "" {
				item.URL = "https://open.spotify.com/track/" + id
			}
		}
	}

	return item, true
}

// extractID returns the last segment of a Spotify URI (e.g. "spotify:track:ABC" → "ABC").
func extractID(uri string) string {
	parts := strings.Split(uri, ":")
	if len(parts) < 3 {
		return ""
	}
	return parts[len(parts)-1]
}

// buildRawMetadata collects platform-specific fields for storage.
func buildRawMetadata(e entry) map[string]any {
	m := map[string]any{
		"username":     e.Username,
		"platform":     e.Platform,
		"conn_country": e.ConnCountry,
	}

	if e.AlbumName != nil {
		m["album"] = *e.AlbumName
	}
	if e.Shuffle != nil {
		m["shuffle"] = *e.Shuffle
	}
	if e.Skipped != nil {
		m["skipped"] = *e.Skipped
	}
	if e.Offline != nil {
		m["offline"] = *e.Offline
	}
	if e.ReasonStart != nil {
		m["reason_start"] = *e.ReasonStart
	}
	if e.ReasonEnd != nil {
		m["reason_end"] = *e.ReasonEnd
	}
	if e.IncognitoMode != nil {
		m["incognito_mode"] = *e.IncognitoMode
	}

	return m
}
