package tiktok

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/justestif/specto/internal/core"
)

func TestName(t *testing.T) {
	p := New()
	if got := p.Name(); got != "tiktok" {
		t.Errorf("Name() = %q, want %q", got, "tiktok")
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

func TestSyncValidWatchHistory(t *testing.T) {
	input := `{
		"Activity": {
			"Video Browsing History": {
				"VideoList": [
					{
						"Date": "2024-04-20 23:17:46",
						"VideoLink": "https://www.tiktokv.com/share/video/7359012345678901234/"
					},
					{
						"Date": "2024-04-20 16:35:40",
						"VideoLink": "https://www.tiktokv.com/share/video/7358098765432109876/"
					}
				]
			}
		}
	}`

	p := New()
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader(input)}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}

	// Verify first entry
	item := result.Items[0]
	if item.Platform != "tiktok" {
		t.Errorf("Platform = %q, want %q", item.Platform, "tiktok")
	}
	if item.Type != core.MediaVideo {
		t.Errorf("Type = %q, want %q", item.Type, core.MediaVideo)
	}
	if item.ExternalID != "7359012345678901234" {
		t.Errorf("ExternalID = %q, want %q", item.ExternalID, "7359012345678901234")
	}
	if item.URL != "https://www.tiktokv.com/share/video/7359012345678901234/" {
		t.Errorf("URL = %q, want %q", item.URL, "https://www.tiktokv.com/share/video/7359012345678901234/")
	}

	wantTime, _ := time.Parse("2006-01-02 15:04:05", "2024-04-20 23:17:46")
	if !item.ConsumedAt.Equal(wantTime) {
		t.Errorf("ConsumedAt = %v, want %v", item.ConsumedAt, wantTime)
	}

	// Title and Creator are empty (not in export)
	if item.Title != "" {
		t.Errorf("Title = %q, want empty", item.Title)
	}
	if item.Creator != "" {
		t.Errorf("Creator = %q, want empty", item.Creator)
	}

	// TimeSpent and Duration not available in TikTok exports
	if item.TimeSpent != nil {
		t.Errorf("TimeSpent = %v, want nil", item.TimeSpent)
	}
	if item.Duration != nil {
		t.Errorf("Duration = %v, want nil", item.Duration)
	}

	// Verify second entry
	item2 := result.Items[1]
	if item2.ExternalID != "7358098765432109876" {
		t.Errorf("Items[1].ExternalID = %q, want %q", item2.ExternalID, "7358098765432109876")
	}

	// Cursor and HasMore should be default for file imports
	if result.NextCursor != "" {
		t.Errorf("NextCursor = %q, want empty", result.NextCursor)
	}
	if result.HasMore {
		t.Errorf("HasMore = true, want false")
	}
}

func TestSyncVideoIDExtraction(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "standard tiktokv URL",
			url:  "https://www.tiktokv.com/share/video/7359012345678901234/",
			want: "7359012345678901234",
		},
		{
			name: "URL without trailing slash",
			url:  "https://www.tiktokv.com/share/video/7359012345678901234",
			want: "7359012345678901234",
		},
		{
			name: "invalid URL",
			url:  "https://www.tiktok.com/@user/video/123",
			want: "",
		},
		{
			name: "empty URL",
			url:  "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractVideoID(tt.url)
			if got != tt.want {
				t.Errorf("extractVideoID(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestSyncLikeFavoriteMerging(t *testing.T) {
	input := `{
		"Activity": {
			"Video Browsing History": {
				"VideoList": [
					{
						"Date": "2024-04-20 23:17:46",
						"VideoLink": "https://www.tiktokv.com/share/video/1111111111111111111/"
					},
					{
						"Date": "2024-04-20 16:35:40",
						"VideoLink": "https://www.tiktokv.com/share/video/2222222222222222222/"
					},
					{
						"Date": "2024-04-19 10:00:00",
						"VideoLink": "https://www.tiktokv.com/share/video/3333333333333333333/"
					}
				]
			},
			"Like List": {
				"ItemFavoriteList": [
					{
						"Date": "2024-04-20 23:18:00",
						"VideoLink": "https://www.tiktokv.com/share/video/1111111111111111111/"
					},
					{
						"Date": "2024-04-19 10:01:00",
						"VideoLink": "https://www.tiktokv.com/share/video/3333333333333333333/"
					}
				]
			},
			"Favorite Videos": {
				"FavoriteVideoList": [
					{
						"Date": "2024-04-20 16:36:00",
						"VideoLink": "https://www.tiktokv.com/share/video/2222222222222222222/"
					},
					{
						"Date": "2024-04-19 10:02:00",
						"VideoLink": "https://www.tiktokv.com/share/video/3333333333333333333/"
					}
				]
			}
		}
	}`

	p := New()
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader(input)}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result.Items))
	}

	// Video 1: liked only
	item1 := result.Items[0]
	if item1.RawMetadata["liked"] != true {
		t.Errorf("Items[0] should be liked, got %v", item1.RawMetadata["liked"])
	}
	if _, ok := item1.RawMetadata["favorited"]; ok {
		t.Errorf("Items[0] should not be favorited, got %v", item1.RawMetadata["favorited"])
	}

	// Video 2: favorited only
	item2 := result.Items[1]
	if _, ok := item2.RawMetadata["liked"]; ok {
		t.Errorf("Items[1] should not be liked, got %v", item2.RawMetadata["liked"])
	}
	if item2.RawMetadata["favorited"] != true {
		t.Errorf("Items[1] should be favorited, got %v", item2.RawMetadata["favorited"])
	}

	// Video 3: both liked and favorited
	item3 := result.Items[2]
	if item3.RawMetadata["liked"] != true {
		t.Errorf("Items[2] should be liked, got %v", item3.RawMetadata["liked"])
	}
	if item3.RawMetadata["favorited"] != true {
		t.Errorf("Items[2] should be favorited, got %v", item3.RawMetadata["favorited"])
	}
}

func TestSyncMissingSections(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{
			name:  "missing VideoList",
			input: `{"Activity": {"Video Browsing History": {}}}`,
			want:  0,
		},
		{
			name:  "missing Video Browsing History",
			input: `{"Activity": {}}`,
			want:  0,
		},
		{
			name:  "missing Activity",
			input: `{}`,
			want:  0,
		},
		{
			name:  "missing Like List and Favorites",
			input: `{"Activity": {"Video Browsing History": {"VideoList": [{"Date": "2024-04-20 23:17:46", "VideoLink": "https://www.tiktokv.com/share/video/1234567890123456789/"}]}}}`,
			want:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader(tt.input)}, "")

			if result.Err != nil {
				t.Fatalf("unexpected error: %v", result.Err)
			}
			if len(result.Items) != tt.want {
				t.Errorf("expected %d items, got %d", tt.want, len(result.Items))
			}
		})
	}
}

func TestSyncEmptyVideoList(t *testing.T) {
	input := `{
		"Activity": {
			"Video Browsing History": {
				"VideoList": []
			}
		}
	}`

	p := New()
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader(input)}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
}

func TestSyncInvalidJSON(t *testing.T) {
	p := New()
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader("{not valid json]")}, "")

	if result.Err == nil {
		t.Fatal("expected error for invalid JSON")
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

func TestSyncSkipsBadDatesAndMissingLinks(t *testing.T) {
	input := `{
		"Activity": {
			"Video Browsing History": {
				"VideoList": [
					{
						"Date": "not-a-date",
						"VideoLink": "https://www.tiktokv.com/share/video/1111111111111111111/"
					},
					{
						"Date": "2024-04-20 23:17:46",
						"VideoLink": ""
					},
					{
						"Date": "2024-04-20 23:17:46",
						"VideoLink": "https://www.tiktokv.com/share/video/2222222222222222222/"
					},
					{
						"Date": "",
						"VideoLink": "https://www.tiktokv.com/share/video/3333333333333333333/"
					},
					{
						"Date": "2024-04-20 10:00:00",
						"VideoLink": "https://example.com/not-a-tiktok-url"
					}
				]
			}
		}
	}`

	p := New()
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader(input)}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	// Only the third entry has both a valid date and a valid video link
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 valid item, got %d", len(result.Items))
	}
	if result.Items[0].ExternalID != "2222222222222222222" {
		t.Errorf("ExternalID = %q, want %q", result.Items[0].ExternalID, "2222222222222222222")
	}
}

func TestEnrich(t *testing.T) {
	p := New()
	input := []core.MediaItem{
		{Platform: "tiktok", Title: "Video 1", Type: core.MediaVideo},
		{Platform: "tiktok", Title: "Video 2", Type: core.MediaVideo},
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
	input := `{
		"Activity": {
			"Video Browsing History": {
				"VideoList": [
					{
						"Date": "2024-04-20 23:17:46",
						"VideoLink": "https://www.tiktokv.com/share/video/7359012345678901234/"
					}
				]
			}
		}
	}`

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
