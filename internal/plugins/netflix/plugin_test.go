package netflix

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/justestif/specto/internal/core"
)

func TestName(t *testing.T) {
	p := New()
	if got := p.Name(); got != "netflix" {
		t.Errorf("Name() = %q, want %q", got, "netflix")
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

func TestSyncSimpleCSV(t *testing.T) {
	input := "Title,Date\n" +
		"Stranger Things: Season 1: Chapter One,10/1/2019\n" +
		"The Matrix,12/25/2020\n"

	p := New()
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader(input)}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}

	// Verify first entry (TV show)
	item := result.Items[0]
	if item.Platform != "netflix" {
		t.Errorf("Platform = %q, want %q", item.Platform, "netflix")
	}
	if item.Type != core.MediaVideo {
		t.Errorf("Type = %q, want %q", item.Type, core.MediaVideo)
	}
	if item.Title != "Stranger Things: Season 1: Chapter One" {
		t.Errorf("Title = %q, want %q", item.Title, "Stranger Things: Season 1: Chapter One")
	}

	wantTime, _ := time.Parse("1/2/2006", "10/1/2019")
	if !item.ConsumedAt.Equal(wantTime) {
		t.Errorf("ConsumedAt = %v, want %v", item.ConsumedAt, wantTime)
	}

	if item.ExternalID != "Stranger Things: Season 1: Chapter One|10/1/2019" {
		t.Errorf("ExternalID = %q, want %q", item.ExternalID, "Stranger Things: Season 1: Chapter One|10/1/2019")
	}

	// TV title parsing
	if item.RawMetadata["series"] != "Stranger Things" {
		t.Errorf("RawMetadata[series] = %v, want %q", item.RawMetadata["series"], "Stranger Things")
	}
	if item.RawMetadata["season"] != "Season 1" {
		t.Errorf("RawMetadata[season] = %v, want %q", item.RawMetadata["season"], "Season 1")
	}
	if item.RawMetadata["episode"] != "Chapter One" {
		t.Errorf("RawMetadata[episode] = %v, want %q", item.RawMetadata["episode"], "Chapter One")
	}

	// Verify second entry (movie — no TV parsing)
	item2 := result.Items[1]
	if item2.Title != "The Matrix" {
		t.Errorf("Items[1].Title = %q, want %q", item2.Title, "The Matrix")
	}
	if _, ok := item2.RawMetadata["series"]; ok {
		t.Errorf("Items[1].RawMetadata[series] should not be set for a movie")
	}

	// Cursor and HasMore should be default for file imports
	if result.NextCursor != "" {
		t.Errorf("NextCursor = %q, want empty", result.NextCursor)
	}
	if result.HasMore {
		t.Errorf("HasMore = true, want false")
	}
}

func TestSyncGDPRCSV(t *testing.T) {
	input := "Profile Name,Start Time,Duration,Attributes,Title,Supplemental Video Type,Device Type,Bookmark,Latest Bookmark,Country\n" +
		"John,2023-06-01 18:30:00,01:23:45,,Breaking Bad: Season 1: Pilot,,Samsung Smart TV,01:23:45,01:23:45,US\n"

	p := New()
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader(input)}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}

	item := result.Items[0]
	if item.Platform != "netflix" {
		t.Errorf("Platform = %q, want %q", item.Platform, "netflix")
	}
	if item.Type != core.MediaVideo {
		t.Errorf("Type = %q, want %q", item.Type, core.MediaVideo)
	}
	if item.Title != "Breaking Bad: Season 1: Pilot" {
		t.Errorf("Title = %q, want %q", item.Title, "Breaking Bad: Season 1: Pilot")
	}

	wantTime, _ := time.Parse("2006-01-02 15:04:05", "2023-06-01 18:30:00")
	if !item.ConsumedAt.Equal(wantTime) {
		t.Errorf("ConsumedAt = %v, want %v", item.ConsumedAt, wantTime)
	}

	if item.ExternalID != "Breaking Bad: Season 1: Pilot|2023-06-01 18:30:00" {
		t.Errorf("ExternalID = %q, want %q", item.ExternalID, "Breaking Bad: Season 1: Pilot|2023-06-01 18:30:00")
	}

	// Duration
	wantDuration := 1*time.Hour + 23*time.Minute + 45*time.Second
	if item.TimeSpent == nil {
		t.Fatal("TimeSpent is nil")
	}
	if *item.TimeSpent != wantDuration {
		t.Errorf("TimeSpent = %v, want %v", *item.TimeSpent, wantDuration)
	}

	// GDPR metadata
	if item.RawMetadata["profile_name"] != "John" {
		t.Errorf("RawMetadata[profile_name] = %v, want %q", item.RawMetadata["profile_name"], "John")
	}
	if item.RawMetadata["device"] != "Samsung Smart TV" {
		t.Errorf("RawMetadata[device] = %v, want %q", item.RawMetadata["device"], "Samsung Smart TV")
	}
	if item.RawMetadata["country"] != "US" {
		t.Errorf("RawMetadata[country] = %v, want %q", item.RawMetadata["country"], "US")
	}
	if item.RawMetadata["bookmark"] != "01:23:45" {
		t.Errorf("RawMetadata[bookmark] = %v, want %q", item.RawMetadata["bookmark"], "01:23:45")
	}

	// TV title parsing
	if item.RawMetadata["series"] != "Breaking Bad" {
		t.Errorf("RawMetadata[series] = %v, want %q", item.RawMetadata["series"], "Breaking Bad")
	}
	if item.RawMetadata["season"] != "Season 1" {
		t.Errorf("RawMetadata[season] = %v, want %q", item.RawMetadata["season"], "Season 1")
	}
	if item.RawMetadata["episode"] != "Pilot" {
		t.Errorf("RawMetadata[episode] = %v, want %q", item.RawMetadata["episode"], "Pilot")
	}
}

func TestSyncTVTitleParsing(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		series  string
		season  string
		episode string
		isTV    bool
	}{
		{
			name:    "full TV title",
			title:   "Black Mirror: Season 1: The National Anthem",
			series:  "Black Mirror",
			season:  "Season 1",
			episode: "The National Anthem",
			isTV:    true,
		},
		{
			name:   "two-part title (show + season only)",
			title:  "Narcos: Season 3",
			series: "Narcos",
			season: "Season 3",
			isTV:   true,
		},
		{
			name:  "plain movie title",
			title: "Inception",
			isTV:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := "Title,Date\n" + tt.title + ",1/1/2024\n"
			p := New()
			result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader(input)}, "")

			if result.Err != nil {
				t.Fatalf("unexpected error: %v", result.Err)
			}
			if len(result.Items) != 1 {
				t.Fatalf("expected 1 item, got %d", len(result.Items))
			}

			item := result.Items[0]
			if tt.isTV {
				if item.RawMetadata["series"] != tt.series {
					t.Errorf("series = %v, want %q", item.RawMetadata["series"], tt.series)
				}
				if item.RawMetadata["season"] != tt.season {
					t.Errorf("season = %v, want %q", item.RawMetadata["season"], tt.season)
				}
				if tt.episode != "" {
					if item.RawMetadata["episode"] != tt.episode {
						t.Errorf("episode = %v, want %q", item.RawMetadata["episode"], tt.episode)
					}
				}
			} else {
				if _, ok := item.RawMetadata["series"]; ok {
					t.Error("series should not be set for a movie")
				}
			}
		})
	}
}

func TestSyncSupplementalVideoFiltering(t *testing.T) {
	input := "Profile Name,Start Time,Duration,Attributes,Title,Supplemental Video Type,Device Type,Bookmark,Latest Bookmark,Country\n" +
		"John,2023-06-01 18:30:00,01:30:00,,Real Movie,,Chrome on Mac,,,US\n" +
		"John,2023-06-01 19:00:00,00:02:30,,Trailer for Something,TRAILER,Chrome on Mac,,,US\n" +
		"John,2023-06-01 19:05:00,00:01:00,,Preview Clip,PREVIEW,Chrome on Mac,,,US\n"

	p := New()
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader(input)}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item (trailers/previews filtered), got %d", len(result.Items))
	}
	if result.Items[0].Title != "Real Movie" {
		t.Errorf("Title = %q, want %q", result.Items[0].Title, "Real Movie")
	}
}

func TestSyncShortDurationFiltering(t *testing.T) {
	input := "Profile Name,Start Time,Duration,Attributes,Title,Supplemental Video Type,Device Type,Bookmark,Latest Bookmark,Country\n" +
		"John,2023-06-01 18:30:00,00:01:59,,Accidental Click,,TV,,,US\n" +
		"John,2023-06-01 19:00:00,00:02:00,,Just Enough,,TV,,,US\n" +
		"John,2023-06-01 20:00:00,01:45:00,,Full Movie,,TV,,,US\n"

	p := New()
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader(input)}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items (< 2 min filtered), got %d", len(result.Items))
	}
	if result.Items[0].Title != "Just Enough" {
		t.Errorf("Items[0].Title = %q, want %q", result.Items[0].Title, "Just Enough")
	}
	if result.Items[1].Title != "Full Movie" {
		t.Errorf("Items[1].Title = %q, want %q", result.Items[1].Title, "Full Movie")
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
	p := New()
	// Mismatched field counts produce a CSV parse error.
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader("a,b\n1,2,3\n")}, "")

	if result.Err == nil {
		t.Fatal("expected error for invalid CSV")
	}
	if result.Err.Code != core.ErrFileParseError {
		t.Errorf("error code = %q, want %q", result.Err.Code, core.ErrFileParseError)
	}
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

func TestSyncHeaderOnly(t *testing.T) {
	p := New()
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader("Title,Date\n")}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items for header-only CSV, got %d", len(result.Items))
	}
}

func TestEnrich(t *testing.T) {
	p := New()
	input := []core.MediaItem{
		{Platform: "netflix", Title: "Movie A", Type: core.MediaVideo},
		{Platform: "netflix", Title: "Show B", Type: core.MediaVideo},
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
	input := "Title,Date\n" +
		"Some Movie,5/15/2023\n"

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

func TestSyncGDPRColumnOrder(t *testing.T) {
	// Columns in a different order than expected — should still work.
	input := "Country,Title,Duration,Profile Name,Start Time,Supplemental Video Type,Device Type,Attributes,Bookmark,Latest Bookmark\n" +
		"GB,The Crown: Season 1: Wolferton Splash,00:55:00,Jane,2023-08-15 20:00:00,,iPad,,,\n"

	p := New()
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader(input)}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}

	item := result.Items[0]
	if item.Title != "The Crown: Season 1: Wolferton Splash" {
		t.Errorf("Title = %q, want %q", item.Title, "The Crown: Season 1: Wolferton Splash")
	}
	if item.RawMetadata["country"] != "GB" {
		t.Errorf("RawMetadata[country] = %v, want %q", item.RawMetadata["country"], "GB")
	}
	if item.RawMetadata["profile_name"] != "Jane" {
		t.Errorf("RawMetadata[profile_name] = %v, want %q", item.RawMetadata["profile_name"], "Jane")
	}
	if item.RawMetadata["device"] != "iPad" {
		t.Errorf("RawMetadata[device] = %v, want %q", item.RawMetadata["device"], "iPad")
	}
	if item.RawMetadata["series"] != "The Crown" {
		t.Errorf("RawMetadata[series] = %v, want %q", item.RawMetadata["series"], "The Crown")
	}

	wantDuration := 55 * time.Minute
	if item.TimeSpent == nil || *item.TimeSpent != wantDuration {
		t.Errorf("TimeSpent = %v, want %v", item.TimeSpent, wantDuration)
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
		ok    bool
	}{
		{"01:23:45", 1*time.Hour + 23*time.Minute + 45*time.Second, true},
		{"00:02:00", 2 * time.Minute, true},
		{"00:00:30", 30 * time.Second, true},
		{"", 0, false},
		{"invalid", 0, false},
		{"1:2", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := parseDuration(tt.input)
			if ok != tt.ok {
				t.Errorf("parseDuration(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
