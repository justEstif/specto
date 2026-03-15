// Package netflix implements a SourcePlugin that parses Netflix CSV viewing
// history exports into normalized MediaItems. It supports both the simple
// 2-column CSV (Title, Date) and the 10-column GDPR privacy export.
package netflix

import (
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"github.com/justestif/specto/internal/core"
)

// Compile-time interface check.
var _ core.SourcePlugin = (*Plugin)(nil)

// Plugin parses Netflix CSV viewing history exports.
type Plugin struct{}

// New returns a new Netflix file-import plugin.
func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string                  { return "netflix" }
func (p *Plugin) AuthType() core.AuthType       { return core.AuthFileImport }
func (p *Plugin) AuthConfig() *core.OAuthConfig { return nil }

// minDuration is the minimum watch duration to include from GDPR exports.
// Shorter entries are typically accidental clicks or auto-previews.
const minDuration = 2 * time.Minute

// Sync reads the Netflix CSV export from creds.File and returns normalized MediaItems.
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

	reader := csv.NewReader(creds.File)
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	// Read all records at once so we can inspect the header.
	records, err := reader.ReadAll()
	if err != nil {
		return core.SyncResult{
			Err: &core.PluginError{
				Code:    core.ErrFileParseError,
				Message: fmt.Sprintf("invalid CSV: %s", err.Error()),
				Raw:     err,
			},
		}
	}

	if len(records) == 0 {
		return core.SyncResult{
			Err: &core.PluginError{
				Code:    core.ErrFileParseError,
				Message: "empty CSV file",
			},
		}
	}

	header := records[0]
	colIndex := buildColumnIndex(header)

	// Detect format by checking for GDPR-specific columns.
	isGDPR := colIndex["start time"] >= 0 && colIndex["duration"] >= 0

	items := make([]core.MediaItem, 0, len(records)-1)
	for _, row := range records[1:] {
		var item core.MediaItem
		var ok bool
		if isGDPR {
			item, ok = parseGDPRRow(row, colIndex)
		} else {
			item, ok = parseSimpleRow(row, colIndex)
		}
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

// buildColumnIndex maps lowercase header names to their column indices.
// Missing columns map to -1.
func buildColumnIndex(header []string) map[string]int {
	known := []string{
		"title", "date",
		"profile name", "start time", "duration", "attributes",
		"supplemental video type", "device type", "bookmark",
		"latest bookmark", "country",
	}
	idx := make(map[string]int, len(known))
	for _, k := range known {
		idx[k] = -1
	}
	for i, h := range header {
		key := strings.ToLower(strings.TrimSpace(h))
		if _, ok := idx[key]; ok {
			idx[key] = i
		}
	}
	return idx
}

// col safely returns the value at the given column index, or "" if out of bounds or missing.
func col(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

// parseSimpleRow parses a row from the simple 2-column Netflix CSV (Title, Date).
func parseSimpleRow(row []string, colIndex map[string]int) (core.MediaItem, bool) {
	title := col(row, colIndex["title"])
	if title == "" {
		return core.MediaItem{}, false
	}

	dateStr := col(row, colIndex["date"])
	consumedAt, _ := time.Parse("1/2/2006", dateStr)

	item := core.MediaItem{
		Platform:    "netflix",
		Type:        core.MediaVideo,
		Title:       title,
		ConsumedAt:  consumedAt,
		ExternalID:  title + "|" + dateStr,
		RawMetadata: make(map[string]any),
	}

	parseTVTitle(title, &item)

	return item, true
}

// parseGDPRRow parses a row from the GDPR 10-column Netflix CSV.
func parseGDPRRow(row []string, colIndex map[string]int) (core.MediaItem, bool) {
	title := col(row, colIndex["title"])
	if title == "" {
		return core.MediaItem{}, false
	}

	// Filter out trailers and previews.
	if supplemental := col(row, colIndex["supplemental video type"]); supplemental != "" {
		return core.MediaItem{}, false
	}

	// Parse and filter short durations.
	durationStr := col(row, colIndex["duration"])
	dur, durOK := parseDuration(durationStr)
	if durOK && dur < minDuration {
		return core.MediaItem{}, false
	}

	startTimeStr := col(row, colIndex["start time"])
	consumedAt, _ := time.Parse("2006-01-02 15:04:05", startTimeStr)

	item := core.MediaItem{
		Platform:    "netflix",
		Type:        core.MediaVideo,
		Title:       title,
		ConsumedAt:  consumedAt,
		ExternalID:  title + "|" + startTimeStr,
		RawMetadata: map[string]any{},
	}

	if durOK {
		item.TimeSpent = &dur
	}

	// Store GDPR-specific fields in RawMetadata.
	if v := col(row, colIndex["profile name"]); v != "" {
		item.RawMetadata["profile_name"] = v
	}
	if v := col(row, colIndex["device type"]); v != "" {
		item.RawMetadata["device"] = v
	}
	if v := col(row, colIndex["country"]); v != "" {
		item.RawMetadata["country"] = v
	}
	if v := col(row, colIndex["bookmark"]); v != "" {
		item.RawMetadata["bookmark"] = v
	}
	if v := col(row, colIndex["latest bookmark"]); v != "" {
		item.RawMetadata["latest_bookmark"] = v
	}
	if v := col(row, colIndex["attributes"]); v != "" {
		item.RawMetadata["attributes"] = v
	}

	parseTVTitle(title, &item)

	return item, true
}

// parseTVTitle detects TV show titles in the format "Show: Season X: Episode Title"
// and populates RawMetadata with series, season, and episode fields.
func parseTVTitle(title string, item *core.MediaItem) {
	parts := strings.SplitN(title, ": ", 3)
	if len(parts) < 2 {
		return
	}

	item.RawMetadata["series"] = parts[0]
	item.RawMetadata["season"] = parts[1]
	if len(parts) == 3 {
		item.RawMetadata["episode"] = parts[2]
	}
}

// parseDuration parses a duration string in HH:MM:SS format.
// Returns the duration and true on success, or zero and false on failure.
func parseDuration(s string) (time.Duration, bool) {
	if s == "" {
		return 0, false
	}

	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return 0, false
	}

	var hours, mins, secs int
	if _, err := fmt.Sscanf(parts[0], "%d", &hours); err != nil {
		return 0, false
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &mins); err != nil {
		return 0, false
	}
	if _, err := fmt.Sscanf(parts[2], "%d", &secs); err != nil {
		return 0, false
	}

	d := time.Duration(hours)*time.Hour +
		time.Duration(mins)*time.Minute +
		time.Duration(secs)*time.Second
	return d, true
}
