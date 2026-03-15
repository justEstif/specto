package goodreads

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/justestif/specto/internal/core"
)

func TestName(t *testing.T) {
	p := New()
	if got := p.Name(); got != "goodreads" {
		t.Errorf("Name() = %q, want %q", got, "goodreads")
	}
}

func TestAuthType(t *testing.T) {
	p := New()
	if got := p.AuthType(); got != core.AuthFileImport {
		t.Errorf("AuthType() = %v, want AuthFileImport (%v)", got, core.AuthFileImport)
	}
}

func TestAuthConfig(t *testing.T) {
	p := New()
	if got := p.AuthConfig(); got != nil {
		t.Errorf("AuthConfig() = %v, want nil", got)
	}
}

// csvHeader is the standard Goodreads CSV header row used across tests.
const csvHeader = "Book Id,Title,Author,Author l-f,Additional Authors,ISBN,ISBN13,My Rating,Average Rating,Publisher,Binding,Number of Pages,Year Published,Original Publication Year,Date Read,Date Added,Bookshelves,Bookshelves with positions,Exclusive Shelf,My Review,Spoiler,Private Notes,Read Count,Recommended For,Recommended By,Owned Copies,Original Purchase Date,Original Purchase Location,Condition,Condition Description,BCID\n"

// row builds a 31-field CSV row from named fields, filling blanks for unset columns.
// This avoids fragile hand-counting of commas.
func row(fields map[string]string) string {
	columns := []string{
		"Book Id", "Title", "Author", "Author l-f", "Additional Authors",
		"ISBN", "ISBN13", "My Rating", "Average Rating", "Publisher",
		"Binding", "Number of Pages", "Year Published", "Original Publication Year",
		"Date Read", "Date Added", "Bookshelves", "Bookshelves with positions",
		"Exclusive Shelf", "My Review", "Spoiler", "Private Notes", "Read Count",
		"Recommended For", "Recommended By", "Owned Copies",
		"Original Purchase Date", "Original Purchase Location",
		"Condition", "Condition Description", "BCID",
	}
	vals := make([]string, len(columns))
	for i, col := range columns {
		if v, ok := fields[col]; ok {
			vals[i] = v
		}
	}
	return strings.Join(vals, ",") + "\n"
}

func TestSyncValidBook(t *testing.T) {
	input := csvHeader + row(map[string]string{
		"Book Id":                    "13278990",
		"Title":                      "The Housing Monster",
		"Author":                     "prole.info",
		"Author l-f":                 `"prole.info, prole.info"`,
		"ISBN":                       `="160486530X"`,
		"ISBN13":                     `="9781604865301"`,
		"My Rating":                  "4",
		"Average Rating":             "3.77",
		"Publisher":                  "PM Press",
		"Binding":                    "Paperback",
		"Number of Pages":            "160",
		"Year Published":             "2012",
		"Original Publication Year":  "2011",
		"Date Read":                  "2023/01/15",
		"Date Added":                 "2017/12/07",
		"Bookshelves":                "favorites",
		"Bookshelves with positions": "favorites (#1)",
		"Exclusive Shelf":            "read",
		"My Review":                  "<b>Great read</b>",
		"Private Notes":              "My private note",
		"Read Count":                 "2",
	})

	p := New()
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader(input)}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}

	item := result.Items[0]
	if item.Platform != "goodreads" {
		t.Errorf("Platform = %q, want %q", item.Platform, "goodreads")
	}
	if item.Type != core.MediaBook {
		t.Errorf("Type = %q, want %q", item.Type, core.MediaBook)
	}
	if item.Title != "The Housing Monster" {
		t.Errorf("Title = %q, want %q", item.Title, "The Housing Monster")
	}
	if item.Creator != "prole.info" {
		t.Errorf("Creator = %q, want %q", item.Creator, "prole.info")
	}
	if item.ExternalID != "13278990" {
		t.Errorf("ExternalID = %q, want %q", item.ExternalID, "13278990")
	}

	wantTime, _ := time.Parse("2006/01/02", "2023/01/15")
	if !item.ConsumedAt.Equal(wantTime) {
		t.Errorf("ConsumedAt = %v, want %v", item.ConsumedAt, wantTime)
	}

	// Check RawMetadata
	if item.RawMetadata["isbn"] != "160486530X" {
		t.Errorf("RawMetadata[isbn] = %v, want %q", item.RawMetadata["isbn"], "160486530X")
	}
	if item.RawMetadata["isbn13"] != "9781604865301" {
		t.Errorf("RawMetadata[isbn13] = %v, want %q", item.RawMetadata["isbn13"], "9781604865301")
	}
	if item.RawMetadata["rating"] != 4 {
		t.Errorf("RawMetadata[rating] = %v, want 4", item.RawMetadata["rating"])
	}
	if item.RawMetadata["average_rating"] != "3.77" {
		t.Errorf("RawMetadata[average_rating] = %v, want %q", item.RawMetadata["average_rating"], "3.77")
	}
	if item.RawMetadata["status"] != "completed" {
		t.Errorf("RawMetadata[status] = %v, want %q", item.RawMetadata["status"], "completed")
	}
	if item.RawMetadata["page_count"] != 160 {
		t.Errorf("RawMetadata[page_count] = %v, want 160", item.RawMetadata["page_count"])
	}
	if item.RawMetadata["publisher"] != "PM Press" {
		t.Errorf("RawMetadata[publisher] = %v, want %q", item.RawMetadata["publisher"], "PM Press")
	}
	if item.RawMetadata["format"] != "Paperback" {
		t.Errorf("RawMetadata[format] = %v, want %q", item.RawMetadata["format"], "Paperback")
	}
	if item.RawMetadata["release_year"] != "2011" {
		t.Errorf("RawMetadata[release_year] = %v, want %q", item.RawMetadata["release_year"], "2011")
	}
	if item.RawMetadata["read_count"] != 2 {
		t.Errorf("RawMetadata[read_count] = %v, want 2", item.RawMetadata["read_count"])
	}
	if item.RawMetadata["review"] != "<b>Great read</b>" {
		t.Errorf("RawMetadata[review] = %v, want %q", item.RawMetadata["review"], "<b>Great read</b>")
	}
	if item.RawMetadata["private_notes"] != "My private note" {
		t.Errorf("RawMetadata[private_notes] = %v, want %q", item.RawMetadata["private_notes"], "My private note")
	}
	if item.RawMetadata["shelves"] != "favorites" {
		t.Errorf("RawMetadata[shelves] = %v, want %q", item.RawMetadata["shelves"], "favorites")
	}

	// Cursor and HasMore should be default for file imports
	if result.NextCursor != "" {
		t.Errorf("NextCursor = %q, want empty", result.NextCursor)
	}
	if result.HasMore {
		t.Errorf("HasMore = true, want false")
	}
}

func TestSyncISBNStripping(t *testing.T) {
	input := csvHeader + row(map[string]string{
		"Book Id":         "1",
		"Title":           "Test Book",
		"Author":          "Author",
		"ISBN":            `="0451526538"`,
		"ISBN13":          `="9780451526533"`,
		"My Rating":       "0",
		"Average Rating":  "4.00",
		"Binding":         "Hardcover",
		"Date Added":      "2023/01/01",
		"Exclusive Shelf": "read",
		"Read Count":      "1",
	})

	p := New()
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader(input)}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}

	if result.Items[0].RawMetadata["isbn"] != "0451526538" {
		t.Errorf("RawMetadata[isbn] = %v, want %q", result.Items[0].RawMetadata["isbn"], "0451526538")
	}
	if result.Items[0].RawMetadata["isbn13"] != "9780451526533" {
		t.Errorf("RawMetadata[isbn13] = %v, want %q", result.Items[0].RawMetadata["isbn13"], "9780451526533")
	}
}

func TestSyncRatingZeroIsUnrated(t *testing.T) {
	input := csvHeader + row(map[string]string{
		"Book Id":         "2",
		"Title":           "Unrated Book",
		"Author":          "Author",
		"My Rating":       "0",
		"Average Rating":  "3.50",
		"Date Added":      "2023/01/01",
		"Exclusive Shelf": "read",
		"Read Count":      "0",
	})

	p := New()
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader(input)}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}

	if result.Items[0].RawMetadata["rating"] != "unrated" {
		t.Errorf("RawMetadata[rating] = %v, want %q", result.Items[0].RawMetadata["rating"], "unrated")
	}
}

func TestSyncDateReadBlankFallsBackToDateAdded(t *testing.T) {
	input := csvHeader + row(map[string]string{
		"Book Id":         "3",
		"Title":           "Future Book",
		"Author":          "Author",
		"Average Rating":  "4.00",
		"Date Added":      "2022/06/15",
		"Exclusive Shelf": "to-read",
		"Read Count":      "0",
	})

	p := New()
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader(input)}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}

	wantTime, _ := time.Parse("2006/01/02", "2022/06/15")
	if !result.Items[0].ConsumedAt.Equal(wantTime) {
		t.Errorf("ConsumedAt = %v, want %v (should fall back to Date Added)", result.Items[0].ConsumedAt, wantTime)
	}
}

func TestSyncExclusiveShelfMapping(t *testing.T) {
	tests := []struct {
		shelf string
		want  string
	}{
		{"read", "completed"},
		{"currently-reading", "in-progress"},
		{"to-read", "planned"},
	}

	for _, tt := range tests {
		t.Run(tt.shelf, func(t *testing.T) {
			input := csvHeader + row(map[string]string{
				"Book Id":         "4",
				"Title":           "Book",
				"Author":          "Author",
				"Average Rating":  "3.00",
				"Date Added":      "2023/01/01",
				"Exclusive Shelf": tt.shelf,
				"Read Count":      "0",
			})

			p := New()
			result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader(input)}, "")

			if result.Err != nil {
				t.Fatalf("unexpected error: %v", result.Err)
			}
			if len(result.Items) != 1 {
				t.Fatalf("expected 1 item, got %d", len(result.Items))
			}

			if result.Items[0].RawMetadata["status"] != tt.want {
				t.Errorf("RawMetadata[status] = %v, want %q", result.Items[0].RawMetadata["status"], tt.want)
			}
		})
	}
}

func TestSyncMultipleAuthorsJoined(t *testing.T) {
	input := csvHeader + row(map[string]string{
		"Book Id":            "5",
		"Title":              "Coauthored Book",
		"Author":             "Jane Smith",
		"Author l-f":         `"Smith, Jane"`,
		"Additional Authors": "John Doe",
		"My Rating":          "0",
		"Average Rating":     "4.00",
		"Date Added":         "2023/01/01",
		"Exclusive Shelf":    "read",
		"Read Count":         "0",
	})

	p := New()
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader(input)}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}

	if result.Items[0].Creator != "Jane Smith, John Doe" {
		t.Errorf("Creator = %q, want %q", result.Items[0].Creator, "Jane Smith, John Doe")
	}
}

func TestSyncEmptyFile(t *testing.T) {
	p := New()
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader("")}, "")

	if result.Err == nil {
		t.Fatal("expected error for empty file")
	}
	if result.Err.Code != core.ErrFileParseError {
		t.Errorf("error code = %q, want %q", result.Err.Code, core.ErrFileParseError)
	}
}

func TestSyncInvalidCSV(t *testing.T) {
	// A bare quote mid-field without LazyQuotes won't trip since we enable LazyQuotes.
	// Use a file that isn't valid CSV at all — just binary garbage with no header.
	p := New()
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader("not,a,valid\x00header\n\x00\x00")}, "")

	// Even if parsing "succeeds", the header won't match any expected columns,
	// so items will have empty fields. Verify no crash. The real invalid-CSV test
	// is ensuring the error path works for truly broken input.
	// Use a reader that errors on read.
	result = p.Sync(context.Background(), core.Credentials{File: &errorReader{}}, "")

	if result.Err == nil {
		t.Fatal("expected error for invalid CSV")
	}
	if result.Err.Code != core.ErrFileParseError {
		t.Errorf("error code = %q, want %q", result.Err.Code, core.ErrFileParseError)
	}
}

// errorReader is an io.Reader that always returns an error.
type errorReader struct{}

func (e *errorReader) Read([]byte) (int, error) {
	return 0, fmt.Errorf("simulated read error")
}

func TestSyncNilFile(t *testing.T) {
	p := New()
	result := p.Sync(context.Background(), core.Credentials{}, "")

	if result.Err == nil {
		t.Fatal("expected error for nil file")
	}
	if result.Err.Code != core.ErrFileParseError {
		t.Errorf("error code = %q, want %q", result.Err.Code, core.ErrFileParseError)
	}
}

func TestEnrich(t *testing.T) {
	p := New()
	input := []core.MediaItem{
		{Platform: "goodreads", Title: "Book One", Type: core.MediaBook},
		{Platform: "goodreads", Title: "Book Two", Type: core.MediaBook},
	}

	got, err := p.Enrich(context.Background(), core.Credentials{}, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(input) {
		t.Fatalf("expected %d items, got %d", len(input), len(got))
	}
	for i := range input {
		if got[i].Title != input[i].Title {
			t.Errorf("Items[%d].Title = %q, want %q", i, got[i].Title, input[i].Title)
		}
	}
}

func TestSyncCursorIgnored(t *testing.T) {
	input := csvHeader + row(map[string]string{
		"Book Id":         "6",
		"Title":           "Test Book",
		"Author":          "Author",
		"My Rating":       "3",
		"Average Rating":  "4.00",
		"Date Added":      "2023/01/01",
		"Exclusive Shelf": "read",
		"Read Count":      "1",
	})

	p := New()
	// Cursor should be completely ignored for file imports
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader(input)}, "some-cursor-value")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(result.Items))
	}
}

func TestSyncHeaderOnly(t *testing.T) {
	p := New()
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader(csvHeader)}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items for header-only CSV, got %d", len(result.Items))
	}
}

func TestStripISBN(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`="0451526538"`, "0451526538"},
		{`="9781604865301"`, "9781604865301"},
		{"", ""},
		{"0451526538", "0451526538"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := stripISBN(tt.input); got != tt.want {
				t.Errorf("stripISBN(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMapShelfStatus(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"read", "completed"},
		{"currently-reading", "in-progress"},
		{"to-read", "planned"},
		{"custom-shelf", "custom-shelf"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := mapShelfStatus(tt.input); got != tt.want {
				t.Errorf("mapShelfStatus(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
