package youtube

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/justestif/specto/internal/core"
)

func TestName(t *testing.T) {
	p := New()
	if got := p.Name(); got != "youtube" {
		t.Errorf("Name() = %q, want %q", got, "youtube")
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

func TestSyncValidVideo(t *testing.T) {
	input := `[
		{
			"header": "YouTube",
			"title": "Watched Building a REST API in Go",
			"titleUrl": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			"subtitles": [
				{
					"name": "Tech With Tim",
					"url": "https://www.youtube.com/channel/UC4JX40jDee_tINbkjycV4Sg"
				}
			],
			"time": "2025-01-15T14:32:00.000Z",
			"products": ["YouTube"],
			"activityControls": ["YouTube watch history"]
		}
	]`

	p := New()
	result := p.Sync(context.Background(), core.Credentials{
		File: strings.NewReader(input),
	}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.HasMore {
		t.Error("HasMore = true, want false")
	}
	if len(result.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(result.Items))
	}

	item := result.Items[0]

	if item.Platform != "youtube" {
		t.Errorf("Platform = %q, want %q", item.Platform, "youtube")
	}
	if item.Type != core.MediaVideo {
		t.Errorf("Type = %q, want %q", item.Type, core.MediaVideo)
	}
	if item.Title != "Building a REST API in Go" {
		t.Errorf("Title = %q, want %q", item.Title, "Building a REST API in Go")
	}
	if item.Creator != "Tech With Tim" {
		t.Errorf("Creator = %q, want %q", item.Creator, "Tech With Tim")
	}

	wantTime := time.Date(2025, 1, 15, 14, 32, 0, 0, time.UTC)
	if !item.ConsumedAt.Equal(wantTime) {
		t.Errorf("ConsumedAt = %v, want %v", item.ConsumedAt, wantTime)
	}

	if item.URL != "https://www.youtube.com/watch?v=dQw4w9WgXcQ" {
		t.Errorf("URL = %q, want YouTube watch URL", item.URL)
	}
	if item.ExternalID != "dQw4w9WgXcQ" {
		t.Errorf("ExternalID = %q, want %q", item.ExternalID, "dQw4w9WgXcQ")
	}

	// Check RawMetadata
	if item.RawMetadata["header"] != "YouTube" {
		t.Errorf("RawMetadata[header] = %v, want %q", item.RawMetadata["header"], "YouTube")
	}
	if item.RawMetadata["channel_url"] != "https://www.youtube.com/channel/UC4JX40jDee_tINbkjycV4Sg" {
		t.Errorf("RawMetadata[channel_url] = %v, want channel URL", item.RawMetadata["channel_url"])
	}
	if item.RawMetadata["channel_id"] != "UC4JX40jDee_tINbkjycV4Sg" {
		t.Errorf("RawMetadata[channel_id] = %v, want %q", item.RawMetadata["channel_id"], "UC4JX40jDee_tINbkjycV4Sg")
	}
}

func TestSyncYouTubeMusic(t *testing.T) {
	input := `[
		{
			"header": "YouTube Music",
			"title": "Watched Bohemian Rhapsody",
			"titleUrl": "https://www.youtube.com/watch?v=fJ9rUzIMcZQ",
			"subtitles": [
				{
					"name": "Queen - Topic",
					"url": "https://www.youtube.com/channel/UCiMhD4jzUqG-IgPzUmmytRQ"
				}
			],
			"time": "2025-03-10T08:15:30.123Z",
			"products": ["YouTube Music"],
			"activityControls": ["YouTube watch history"]
		}
	]`

	p := New()
	result := p.Sync(context.Background(), core.Credentials{
		File: strings.NewReader(input),
	}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(result.Items))
	}

	item := result.Items[0]

	if item.Type != core.MediaMusic {
		t.Errorf("Type = %q, want %q", item.Type, core.MediaMusic)
	}
	if item.Creator != "Queen" {
		t.Errorf("Creator = %q, want %q (should strip ' - Topic')", item.Creator, "Queen")
	}
	if item.RawMetadata["header"] != "YouTube Music" {
		t.Errorf("RawMetadata[header] = %v, want %q", item.RawMetadata["header"], "YouTube Music")
	}
}

func TestSyncSkipsDeletedVideos(t *testing.T) {
	input := `[
		{
			"header": "YouTube",
			"title": "Watched a video that has been removed",
			"time": "2025-01-10T12:00:00.000Z",
			"products": ["YouTube"],
			"activityControls": ["YouTube watch history"]
		}
	]`

	p := New()
	result := p.Sync(context.Background(), core.Credentials{
		File: strings.NewReader(input),
	}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 0 {
		t.Errorf("got %d items, want 0 (deleted videos should be skipped)", len(result.Items))
	}
}

func TestSyncSkipsAds(t *testing.T) {
	input := `[
		{
			"header": "YouTube",
			"title": "Watched Some Ad",
			"titleUrl": "https://www.youtube.com/watch?v=ad12345",
			"subtitles": [
				{
					"name": "Advertiser",
					"url": "https://www.youtube.com/channel/UCad12345"
				}
			],
			"time": "2025-02-20T10:00:00.000Z",
			"products": ["YouTube"],
			"activityControls": ["YouTube watch history"],
			"details": [
				{"name": "From Google Ads"}
			]
		}
	]`

	p := New()
	result := p.Sync(context.Background(), core.Credentials{
		File: strings.NewReader(input),
	}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 0 {
		t.Errorf("got %d items, want 0 (ads should be skipped)", len(result.Items))
	}
}

func TestSyncMissingSubtitles(t *testing.T) {
	input := `[
		{
			"header": "YouTube",
			"title": "Watched Mystery Video",
			"titleUrl": "https://www.youtube.com/watch?v=abc123",
			"time": "2025-04-01T00:00:00.000Z",
			"products": ["YouTube"],
			"activityControls": ["YouTube watch history"]
		}
	]`

	p := New()
	result := p.Sync(context.Background(), core.Credentials{
		File: strings.NewReader(input),
	}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(result.Items))
	}

	item := result.Items[0]
	if item.Creator != "" {
		t.Errorf("Creator = %q, want empty string for missing subtitles", item.Creator)
	}
	if item.Title != "Mystery Video" {
		t.Errorf("Title = %q, want %q", item.Title, "Mystery Video")
	}

	// channel_url and channel_id should not be in RawMetadata
	if _, ok := item.RawMetadata["channel_url"]; ok {
		t.Error("RawMetadata should not contain channel_url when subtitles are missing")
	}
	if _, ok := item.RawMetadata["channel_id"]; ok {
		t.Error("RawMetadata should not contain channel_id when subtitles are missing")
	}
}

func TestSyncEmptyFile(t *testing.T) {
	p := New()
	result := p.Sync(context.Background(), core.Credentials{
		File: strings.NewReader(""),
	}, "")

	if result.Err == nil {
		t.Fatal("expected error for empty file, got nil")
	}
	if result.Err.Code != core.ErrFileParseError {
		t.Errorf("error code = %q, want %q", result.Err.Code, core.ErrFileParseError)
	}
}

func TestSyncInvalidJSON(t *testing.T) {
	p := New()
	result := p.Sync(context.Background(), core.Credentials{
		File: strings.NewReader("{not valid json!!!"),
	}, "")

	if result.Err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if result.Err.Code != core.ErrFileParseError {
		t.Errorf("error code = %q, want %q", result.Err.Code, core.ErrFileParseError)
	}
	if result.Err.Raw == nil {
		t.Error("expected Raw error to be non-nil")
	}
}

func TestSyncEmptyArray(t *testing.T) {
	p := New()
	result := p.Sync(context.Background(), core.Credentials{
		File: strings.NewReader("[]"),
	}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 0 {
		t.Errorf("got %d items, want 0", len(result.Items))
	}
}

func TestSyncMixedEntries(t *testing.T) {
	input := `[
		{
			"header": "YouTube",
			"title": "Watched Normal Video",
			"titleUrl": "https://www.youtube.com/watch?v=normal1",
			"subtitles": [{"name": "Creator One", "url": "https://www.youtube.com/channel/UC1"}],
			"time": "2025-01-01T00:00:00.000Z",
			"products": ["YouTube"],
			"activityControls": ["YouTube watch history"]
		},
		{
			"header": "YouTube Music",
			"title": "Watched Cool Song",
			"titleUrl": "https://www.youtube.com/watch?v=music1",
			"subtitles": [{"name": "Artist - Topic", "url": "https://www.youtube.com/channel/UC2"}],
			"time": "2025-01-02T00:00:00.000Z",
			"products": ["YouTube Music"],
			"activityControls": ["YouTube watch history"]
		},
		{
			"header": "YouTube",
			"title": "Watched a video that has been removed",
			"time": "2025-01-03T00:00:00.000Z",
			"products": ["YouTube"],
			"activityControls": ["YouTube watch history"]
		},
		{
			"header": "YouTube",
			"title": "Watched Ad Video",
			"titleUrl": "https://www.youtube.com/watch?v=ad1",
			"subtitles": [{"name": "AdCo", "url": "https://www.youtube.com/channel/UC3"}],
			"time": "2025-01-04T00:00:00.000Z",
			"products": ["YouTube"],
			"activityControls": ["YouTube watch history"],
			"details": [{"name": "From Google Ads"}]
		},
		{
			"header": "YouTube",
			"title": "Watched Another Normal Video",
			"titleUrl": "https://www.youtube.com/watch?v=normal2",
			"subtitles": [{"name": "Creator Two", "url": "https://www.youtube.com/channel/UC4"}],
			"time": "2025-01-05T00:00:00.000Z",
			"products": ["YouTube"],
			"activityControls": ["YouTube watch history"]
		}
	]`

	p := New()
	result := p.Sync(context.Background(), core.Credentials{
		File: strings.NewReader(input),
	}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	// Should include: Normal Video, Cool Song, Another Normal Video (3 total)
	// Skipped: deleted video, ad
	if len(result.Items) != 3 {
		t.Fatalf("got %d items, want 3 (deleted + ad skipped)", len(result.Items))
	}

	// First item: normal video
	if result.Items[0].Type != core.MediaVideo {
		t.Errorf("items[0].Type = %q, want %q", result.Items[0].Type, core.MediaVideo)
	}
	if result.Items[0].Title != "Normal Video" {
		t.Errorf("items[0].Title = %q, want %q", result.Items[0].Title, "Normal Video")
	}

	// Second item: music
	if result.Items[1].Type != core.MediaMusic {
		t.Errorf("items[1].Type = %q, want %q", result.Items[1].Type, core.MediaMusic)
	}
	if result.Items[1].Creator != "Artist" {
		t.Errorf("items[1].Creator = %q, want %q", result.Items[1].Creator, "Artist")
	}

	// Third item: another normal video
	if result.Items[2].Title != "Another Normal Video" {
		t.Errorf("items[2].Title = %q, want %q", result.Items[2].Title, "Another Normal Video")
	}
}

func TestSyncBlankTopicCreator(t *testing.T) {
	input := `[
		{
			"header": "YouTube Music",
			"title": "Watched Some Track",
			"titleUrl": "https://www.youtube.com/watch?v=topic1",
			"subtitles": [
				{
					"name": " - Topic",
					"url": "https://www.youtube.com/channel/UCblank"
				}
			],
			"time": "2025-05-01T12:00:00.000Z",
			"products": ["YouTube Music"],
			"activityControls": ["YouTube watch history"]
		}
	]`

	p := New()
	result := p.Sync(context.Background(), core.Credentials{
		File: strings.NewReader(input),
	}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(result.Items))
	}

	if result.Items[0].Creator != "" {
		t.Errorf("Creator = %q, want empty string for blank topic creator", result.Items[0].Creator)
	}
}

func TestEnrich(t *testing.T) {
	p := New()

	items := []core.MediaItem{
		{
			Platform: "youtube",
			Type:     core.MediaVideo,
			Title:    "Test Video",
			Creator:  "Test Creator",
		},
		{
			Platform: "youtube",
			Type:     core.MediaMusic,
			Title:    "Test Song",
			Creator:  "Test Artist",
		},
	}

	got, err := p.Enrich(context.Background(), core.Credentials{}, items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != len(items) {
		t.Fatalf("got %d items, want %d", len(got), len(items))
	}

	for i := range items {
		if got[i].Title != items[i].Title {
			t.Errorf("items[%d].Title = %q, want %q", i, got[i].Title, items[i].Title)
		}
		if got[i].Creator != items[i].Creator {
			t.Errorf("items[%d].Creator = %q, want %q", i, got[i].Creator, items[i].Creator)
		}
	}
}
