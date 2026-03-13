package spotify

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/justestif/specto/internal/core"
)

func TestName(t *testing.T) {
	p := New()
	if got := p.Name(); got != "spotify" {
		t.Errorf("Name() = %q, want %q", got, "spotify")
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

func TestSyncValidMusic(t *testing.T) {
	input := `[
		{
			"ts": "2024-11-15T08:32:17Z",
			"username": "johndoe42",
			"platform": "Android OS Free",
			"ms_played": 214523,
			"conn_country": "US",
			"ip_addr_decrypted": "192.168.1.42",
			"user_agent_decrypted": null,
			"master_metadata_track_name": "Bohemian Rhapsody",
			"master_metadata_album_artist_name": "Queen",
			"master_metadata_album_album_name": "A Night at the Opera",
			"spotify_track_uri": "spotify:track:4u7EnebtmKWzUH433cf5Qv",
			"episode_name": null,
			"episode_show_name": null,
			"spotify_episode_uri": null,
			"reason_start": "clickrow",
			"reason_end": "trackdone",
			"shuffle": false,
			"skipped": null,
			"offline": false,
			"offline_timestamp": 0,
			"incognito_mode": false
		},
		{
			"ts": "2024-11-15T09:00:00Z",
			"username": "johndoe42",
			"platform": "Windows Desktop",
			"ms_played": 180000,
			"conn_country": "US",
			"ip_addr_decrypted": null,
			"user_agent_decrypted": null,
			"master_metadata_track_name": "Stairway to Heaven",
			"master_metadata_album_artist_name": "Led Zeppelin",
			"master_metadata_album_album_name": "Led Zeppelin IV",
			"spotify_track_uri": "spotify:track:5CQ30WqJwcep0pYcV4AMNc",
			"episode_name": null,
			"episode_show_name": null,
			"spotify_episode_uri": null,
			"reason_start": "fwdbtn",
			"reason_end": "trackdone",
			"shuffle": true,
			"skipped": false,
			"offline": false,
			"offline_timestamp": 0,
			"incognito_mode": false
		}
	]`

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
	if item.Platform != "spotify" {
		t.Errorf("Platform = %q, want %q", item.Platform, "spotify")
	}
	if item.Type != core.MediaMusic {
		t.Errorf("Type = %q, want %q", item.Type, core.MediaMusic)
	}
	if item.Title != "Bohemian Rhapsody" {
		t.Errorf("Title = %q, want %q", item.Title, "Bohemian Rhapsody")
	}
	if item.Creator != "Queen" {
		t.Errorf("Creator = %q, want %q", item.Creator, "Queen")
	}

	wantTime, _ := time.Parse(time.RFC3339, "2024-11-15T08:32:17Z")
	if !item.ConsumedAt.Equal(wantTime) {
		t.Errorf("ConsumedAt = %v, want %v", item.ConsumedAt, wantTime)
	}

	if item.ExternalID != "spotify:track:4u7EnebtmKWzUH433cf5Qv" {
		t.Errorf("ExternalID = %q, want %q", item.ExternalID, "spotify:track:4u7EnebtmKWzUH433cf5Qv")
	}
	if item.URL != "https://open.spotify.com/track/4u7EnebtmKWzUH433cf5Qv" {
		t.Errorf("URL = %q, want %q", item.URL, "https://open.spotify.com/track/4u7EnebtmKWzUH433cf5Qv")
	}

	wantDuration := 214523 * time.Millisecond
	if item.TimeSpent == nil {
		t.Fatal("TimeSpent is nil")
	}
	if *item.TimeSpent != wantDuration {
		t.Errorf("TimeSpent = %v, want %v", *item.TimeSpent, wantDuration)
	}

	// Check RawMetadata
	if item.RawMetadata["album"] != "A Night at the Opera" {
		t.Errorf("RawMetadata[album] = %v, want %q", item.RawMetadata["album"], "A Night at the Opera")
	}
	if item.RawMetadata["shuffle"] != false {
		t.Errorf("RawMetadata[shuffle] = %v, want false", item.RawMetadata["shuffle"])
	}
	if item.RawMetadata["reason_start"] != "clickrow" {
		t.Errorf("RawMetadata[reason_start] = %v, want %q", item.RawMetadata["reason_start"], "clickrow")
	}
	if item.RawMetadata["reason_end"] != "trackdone" {
		t.Errorf("RawMetadata[reason_end] = %v, want %q", item.RawMetadata["reason_end"], "trackdone")
	}
	if item.RawMetadata["conn_country"] != "US" {
		t.Errorf("RawMetadata[conn_country] = %v, want %q", item.RawMetadata["conn_country"], "US")
	}
	if item.RawMetadata["incognito_mode"] != false {
		t.Errorf("RawMetadata[incognito_mode] = %v, want false", item.RawMetadata["incognito_mode"])
	}
	if item.RawMetadata["username"] != "johndoe42" {
		t.Errorf("RawMetadata[username] = %v, want %q", item.RawMetadata["username"], "johndoe42")
	}
	if item.RawMetadata["platform"] != "Android OS Free" {
		t.Errorf("RawMetadata[platform] = %v, want %q", item.RawMetadata["platform"], "Android OS Free")
	}
	if item.RawMetadata["offline"] != false {
		t.Errorf("RawMetadata[offline] = %v, want false", item.RawMetadata["offline"])
	}

	// Verify second entry
	item2 := result.Items[1]
	if item2.Title != "Stairway to Heaven" {
		t.Errorf("Items[1].Title = %q, want %q", item2.Title, "Stairway to Heaven")
	}
	if item2.Creator != "Led Zeppelin" {
		t.Errorf("Items[1].Creator = %q, want %q", item2.Creator, "Led Zeppelin")
	}
	if item2.RawMetadata["shuffle"] != true {
		t.Errorf("Items[1].RawMetadata[shuffle] = %v, want true", item2.RawMetadata["shuffle"])
	}

	// Cursor and HasMore should be default for file imports
	if result.NextCursor != "" {
		t.Errorf("NextCursor = %q, want empty", result.NextCursor)
	}
	if result.HasMore {
		t.Errorf("HasMore = true, want false")
	}
}

func TestSyncPodcastEntry(t *testing.T) {
	input := `[
		{
			"ts": "2024-12-01T10:00:00Z",
			"username": "podlistener",
			"platform": "iOS",
			"ms_played": 3600000,
			"conn_country": "GB",
			"ip_addr_decrypted": null,
			"user_agent_decrypted": null,
			"master_metadata_track_name": null,
			"master_metadata_album_artist_name": null,
			"master_metadata_album_album_name": null,
			"spotify_track_uri": null,
			"episode_name": "Episode 42: The Meaning of Life",
			"episode_show_name": "The Deep Dive Podcast",
			"spotify_episode_uri": "spotify:episode:7abc123def456",
			"reason_start": "clickrow",
			"reason_end": "trackdone",
			"shuffle": false,
			"skipped": false,
			"offline": true,
			"offline_timestamp": 1701420000,
			"incognito_mode": false
		}
	]`

	p := New()
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader(input)}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}

	item := result.Items[0]
	if item.Type != core.MediaPodcast {
		t.Errorf("Type = %q, want %q", item.Type, core.MediaPodcast)
	}
	if item.Title != "Episode 42: The Meaning of Life" {
		t.Errorf("Title = %q, want %q", item.Title, "Episode 42: The Meaning of Life")
	}
	if item.Creator != "The Deep Dive Podcast" {
		t.Errorf("Creator = %q, want %q", item.Creator, "The Deep Dive Podcast")
	}
	if item.ExternalID != "spotify:episode:7abc123def456" {
		t.Errorf("ExternalID = %q, want %q", item.ExternalID, "spotify:episode:7abc123def456")
	}
	if item.URL != "https://open.spotify.com/episode/7abc123def456" {
		t.Errorf("URL = %q, want %q", item.URL, "https://open.spotify.com/episode/7abc123def456")
	}

	wantDuration := 3600000 * time.Millisecond
	if item.TimeSpent == nil || *item.TimeSpent != wantDuration {
		t.Errorf("TimeSpent = %v, want %v", item.TimeSpent, wantDuration)
	}
}

func TestSyncSkipsLocalFiles(t *testing.T) {
	input := `[
		{
			"ts": "2024-11-15T08:32:17Z",
			"username": "johndoe42",
			"platform": "Android OS Free",
			"ms_played": 120000,
			"conn_country": "US",
			"ip_addr_decrypted": null,
			"user_agent_decrypted": null,
			"master_metadata_track_name": null,
			"master_metadata_album_artist_name": null,
			"master_metadata_album_album_name": null,
			"spotify_track_uri": null,
			"episode_name": null,
			"episode_show_name": null,
			"spotify_episode_uri": null,
			"reason_start": "clickrow",
			"reason_end": "trackdone",
			"shuffle": false,
			"skipped": null,
			"offline": true,
			"offline_timestamp": 0,
			"incognito_mode": false
		}
	]`

	p := New()
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader(input)}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items (local file should be skipped), got %d", len(result.Items))
	}
}

func TestSyncZeroMsPlayed(t *testing.T) {
	input := `[
		{
			"ts": "2024-11-15T08:32:17Z",
			"username": "johndoe42",
			"platform": "Android OS Free",
			"ms_played": 0,
			"conn_country": "US",
			"ip_addr_decrypted": null,
			"user_agent_decrypted": null,
			"master_metadata_track_name": "Quick Skip",
			"master_metadata_album_artist_name": "Some Artist",
			"master_metadata_album_album_name": "Some Album",
			"spotify_track_uri": "spotify:track:zzz111",
			"episode_name": null,
			"episode_show_name": null,
			"spotify_episode_uri": null,
			"reason_start": "clickrow",
			"reason_end": "fwdbtn",
			"shuffle": false,
			"skipped": true,
			"offline": false,
			"offline_timestamp": 0,
			"incognito_mode": false
		}
	]`

	p := New()
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader(input)}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item (zero ms_played should still be included), got %d", len(result.Items))
	}

	item := result.Items[0]
	if item.TimeSpent == nil {
		t.Fatal("TimeSpent is nil")
	}
	if *item.TimeSpent != 0 {
		t.Errorf("TimeSpent = %v, want 0", *item.TimeSpent)
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

func TestSyncEmptyArray(t *testing.T) {
	p := New()
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader("[]")}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
}

func TestSyncMixedEntries(t *testing.T) {
	input := `[
		{
			"ts": "2024-11-15T08:00:00Z",
			"username": "user1",
			"platform": "Desktop",
			"ms_played": 200000,
			"conn_country": "US",
			"ip_addr_decrypted": null,
			"user_agent_decrypted": null,
			"master_metadata_track_name": "Song A",
			"master_metadata_album_artist_name": "Artist A",
			"master_metadata_album_album_name": "Album A",
			"spotify_track_uri": "spotify:track:aaa111",
			"episode_name": null,
			"episode_show_name": null,
			"spotify_episode_uri": null,
			"reason_start": "clickrow",
			"reason_end": "trackdone",
			"shuffle": false,
			"skipped": null,
			"offline": false,
			"offline_timestamp": 0,
			"incognito_mode": false
		},
		{
			"ts": "2024-11-15T09:00:00Z",
			"username": "user1",
			"platform": "Desktop",
			"ms_played": 300000,
			"conn_country": "US",
			"ip_addr_decrypted": null,
			"user_agent_decrypted": null,
			"master_metadata_track_name": null,
			"master_metadata_album_artist_name": null,
			"master_metadata_album_album_name": null,
			"spotify_track_uri": null,
			"episode_name": "Ep 1",
			"episode_show_name": "My Podcast",
			"spotify_episode_uri": "spotify:episode:eee111",
			"reason_start": "clickrow",
			"reason_end": "trackdone",
			"shuffle": false,
			"skipped": null,
			"offline": false,
			"offline_timestamp": 0,
			"incognito_mode": false
		},
		{
			"ts": "2024-11-15T10:00:00Z",
			"username": "user1",
			"platform": "Desktop",
			"ms_played": 50000,
			"conn_country": "US",
			"ip_addr_decrypted": null,
			"user_agent_decrypted": null,
			"master_metadata_track_name": null,
			"master_metadata_album_artist_name": null,
			"master_metadata_album_album_name": null,
			"spotify_track_uri": null,
			"episode_name": null,
			"episode_show_name": null,
			"spotify_episode_uri": null,
			"reason_start": "clickrow",
			"reason_end": "trackdone",
			"shuffle": false,
			"skipped": null,
			"offline": true,
			"offline_timestamp": 0,
			"incognito_mode": false
		}
	]`

	p := New()
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader(input)}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items (1 music + 1 podcast, 1 local skipped), got %d", len(result.Items))
	}

	// First: music
	if result.Items[0].Type != core.MediaMusic {
		t.Errorf("Items[0].Type = %q, want %q", result.Items[0].Type, core.MediaMusic)
	}
	if result.Items[0].Title != "Song A" {
		t.Errorf("Items[0].Title = %q, want %q", result.Items[0].Title, "Song A")
	}

	// Second: podcast
	if result.Items[1].Type != core.MediaPodcast {
		t.Errorf("Items[1].Type = %q, want %q", result.Items[1].Type, core.MediaPodcast)
	}
	if result.Items[1].Title != "Ep 1" {
		t.Errorf("Items[1].Title = %q, want %q", result.Items[1].Title, "Ep 1")
	}
	if result.Items[1].Creator != "My Podcast" {
		t.Errorf("Items[1].Creator = %q, want %q", result.Items[1].Creator, "My Podcast")
	}
}

func TestEnrich(t *testing.T) {
	p := New()
	input := []core.MediaItem{
		{Platform: "spotify", Title: "Song", Type: core.MediaMusic},
		{Platform: "spotify", Title: "Episode", Type: core.MediaPodcast},
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

func TestSyncCursorIgnored(t *testing.T) {
	input := `[
		{
			"ts": "2024-11-15T08:32:17Z",
			"username": "user",
			"platform": "Desktop",
			"ms_played": 100000,
			"conn_country": "US",
			"ip_addr_decrypted": null,
			"user_agent_decrypted": null,
			"master_metadata_track_name": "Test Song",
			"master_metadata_album_artist_name": "Test Artist",
			"master_metadata_album_album_name": "Test Album",
			"spotify_track_uri": "spotify:track:test123",
			"episode_name": null,
			"episode_show_name": null,
			"spotify_episode_uri": null,
			"reason_start": "clickrow",
			"reason_end": "trackdone",
			"shuffle": false,
			"skipped": null,
			"offline": false,
			"offline_timestamp": 0,
			"incognito_mode": false
		}
	]`

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

func TestSyncTrackWithURIButNoName(t *testing.T) {
	// Entry has a track URI but null track name — should still be included
	input := `[
		{
			"ts": "2024-11-15T08:32:17Z",
			"username": "user",
			"platform": "Desktop",
			"ms_played": 100000,
			"conn_country": "US",
			"ip_addr_decrypted": null,
			"user_agent_decrypted": null,
			"master_metadata_track_name": null,
			"master_metadata_album_artist_name": null,
			"master_metadata_album_album_name": null,
			"spotify_track_uri": "spotify:track:xyz789",
			"episode_name": null,
			"episode_show_name": null,
			"spotify_episode_uri": null,
			"reason_start": "clickrow",
			"reason_end": "trackdone",
			"shuffle": false,
			"skipped": null,
			"offline": false,
			"offline_timestamp": 0,
			"incognito_mode": false
		}
	]`

	p := New()
	result := p.Sync(context.Background(), core.Credentials{File: strings.NewReader(input)}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item (has URI even without name), got %d", len(result.Items))
	}
	if result.Items[0].ExternalID != "spotify:track:xyz789" {
		t.Errorf("ExternalID = %q, want %q", result.Items[0].ExternalID, "spotify:track:xyz789")
	}
	if result.Items[0].URL != "https://open.spotify.com/track/xyz789" {
		t.Errorf("URL = %q, want %q", result.Items[0].URL, "https://open.spotify.com/track/xyz789")
	}
}
