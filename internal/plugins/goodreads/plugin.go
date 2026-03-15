// Package goodreads implements a SourcePlugin that parses Goodreads
// CSV library exports into normalized MediaItems.
package goodreads

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/justestif/specto/internal/core"
)

// Compile-time interface check.
var _ core.SourcePlugin = (*Plugin)(nil)

// Plugin parses Goodreads CSV library export files.
type Plugin struct{}

// New returns a new Goodreads file-import plugin.
func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string                  { return "goodreads" }
func (p *Plugin) AuthType() core.AuthType       { return core.AuthFileImport }
func (p *Plugin) AuthConfig() *core.OAuthConfig { return nil }

// Sync reads the Goodreads CSV export from creds.File and returns normalized MediaItems.
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
	reader.FieldsPerRecord = -1 // tolerate rows with fewer/more fields than the header

	headers, err := reader.Read()
	if err != nil {
		return core.SyncResult{
			Err: &core.PluginError{
				Code:    core.ErrFileParseError,
				Message: fmt.Sprintf("invalid CSV: %s", err.Error()),
				Raw:     err,
			},
		}
	}

	colIndex := buildColumnIndex(headers)

	var items []core.MediaItem
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return core.SyncResult{
				Items: items,
				Err: &core.PluginError{
					Code:    core.ErrFileParseError,
					Message: fmt.Sprintf("CSV parse error: %s", err.Error()),
					Raw:     err,
				},
			}
		}

		item := mapRecord(record, colIndex)
		items = append(items, item)
	}

	return core.SyncResult{Items: items}
}

// Enrich returns items unchanged — no platform-specific enrichment for file imports.
func (p *Plugin) Enrich(_ context.Context, _ core.Credentials, items []core.MediaItem) ([]core.MediaItem, error) {
	return items, nil
}

// columnIndex maps header names to their positions in the CSV.
type columnIndex map[string]int

// buildColumnIndex creates a lookup from header name to column position.
func buildColumnIndex(headers []string) columnIndex {
	idx := make(columnIndex, len(headers))
	for i, h := range headers {
		idx[strings.TrimSpace(h)] = i
	}
	return idx
}

// get returns the trimmed value at the named column, or empty string if missing.
func (ci columnIndex) get(record []string, name string) string {
	i, ok := ci[name]
	if !ok || i >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[i])
}

// mapRecord converts a single CSV row into a MediaItem.
func mapRecord(record []string, ci columnIndex) core.MediaItem {
	title := ci.get(record, "Title")
	author := ci.get(record, "Author")
	additionalAuthors := ci.get(record, "Additional Authors")

	creator := author
	if additionalAuthors != "" {
		if creator != "" {
			creator += ", " + additionalAuthors
		} else {
			creator = additionalAuthors
		}
	}

	consumedAt := parseDate(ci.get(record, "Date Read"))
	if consumedAt.IsZero() {
		consumedAt = parseDate(ci.get(record, "Date Added"))
	}

	item := core.MediaItem{
		Platform:    "goodreads",
		Type:        core.MediaBook,
		Title:       title,
		Creator:     creator,
		ExternalID:  ci.get(record, "Book Id"),
		ConsumedAt:  consumedAt,
		RawMetadata: buildRawMetadata(record, ci),
	}

	return item
}

// stripISBN removes the Excel-protection `=""` wrapper from ISBN fields.
// `="0451526538"` → `0451526538`
func stripISBN(raw string) string {
	s := strings.TrimPrefix(raw, `="`)
	s = strings.TrimSuffix(s, `"`)
	return s
}

// parseDate parses a Goodreads date in YYYY/MM/DD format.
// Returns zero time if the input is blank or unparseable.
func parseDate(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse("2006/01/02", s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// mapShelfStatus maps a Goodreads exclusive shelf to a normalized status string.
func mapShelfStatus(shelf string) string {
	switch shelf {
	case "read":
		return "completed"
	case "currently-reading":
		return "in-progress"
	case "to-read":
		return "planned"
	default:
		return shelf
	}
}

// buildRawMetadata collects platform-specific fields for storage.
func buildRawMetadata(record []string, ci columnIndex) map[string]any {
	m := make(map[string]any)

	isbn := stripISBN(ci.get(record, "ISBN"))
	if isbn != "" {
		m["isbn"] = isbn
	}

	isbn13 := stripISBN(ci.get(record, "ISBN13"))
	if isbn13 != "" {
		m["isbn13"] = isbn13
	}

	if rating := ci.get(record, "My Rating"); rating != "" {
		if v, err := strconv.Atoi(rating); err == nil {
			if v == 0 {
				m["rating"] = "unrated"
			} else {
				m["rating"] = v
			}
		}
	}

	if avg := ci.get(record, "Average Rating"); avg != "" {
		m["average_rating"] = avg
	}

	shelf := ci.get(record, "Exclusive Shelf")
	if shelf != "" {
		m["status"] = mapShelfStatus(shelf)
	}

	if pages := ci.get(record, "Number of Pages"); pages != "" {
		if v, err := strconv.Atoi(pages); err == nil {
			m["page_count"] = v
		}
	}

	if publisher := ci.get(record, "Publisher"); publisher != "" {
		m["publisher"] = publisher
	}

	if binding := ci.get(record, "Binding"); binding != "" {
		m["format"] = binding
	}

	if year := ci.get(record, "Original Publication Year"); year != "" {
		m["release_year"] = year
	}

	if readCount := ci.get(record, "Read Count"); readCount != "" {
		if v, err := strconv.Atoi(readCount); err == nil {
			m["read_count"] = v
		}
	}

	if review := ci.get(record, "My Review"); review != "" {
		m["review"] = review
	}

	if notes := ci.get(record, "Private Notes"); notes != "" {
		m["private_notes"] = notes
	}

	if shelves := ci.get(record, "Bookshelves"); shelves != "" {
		m["shelves"] = shelves
	}

	return m
}
